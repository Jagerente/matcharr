package domain

import (
	"fmt"
	"testing"
	"time"
)

var testStart = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

func testSession(t *testing.T, participants int, settings Settings) (*Session, []*Participant) {
	t.Helper()
	host := &Participant{ID: "p1", Name: "Host", IsHost: true, Token: "t1"}
	s := NewSession("ABCD", settings, host, testStart)
	people := []*Participant{host}
	for i := 2; i <= participants; i++ {
		p := &Participant{ID: fmt.Sprintf("p%d", i), Name: fmt.Sprintf("P%d", i), Token: fmt.Sprintf("t%d", i)}
		s.AddParticipant(p)
		people = append(people, p)
	}
	return s, people
}

func deal(s *Session, n int) []Movie {
	movies := make([]Movie, 0, n)
	for i := 1; i <= n; i++ {
		movies = append(movies, Movie{ID: int64(i), Title: fmt.Sprintf("Movie %d", i)})
	}
	s.BeginRound(movies, 2, false)
	return movies
}

func TestThreshold(t *testing.T) {
	tests := []struct {
		name         string
		participants int
		threshold    int
		want         int
	}{
		{"unanimous by default", 3, 0, 3},
		{"explicit below the group size", 4, 2, 2},
		{"threshold at group size means unanimous", 3, 3, 3},
		{"threshold above group size collapses to unanimous", 2, 5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := testSession(t, tt.participants, Settings{Threshold: tt.threshold})
			if got := s.Threshold(); got != tt.want {
				t.Fatalf("Threshold() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSwipeCursorAndVotes(t *testing.T) {
	s, people := testSession(t, 2, Settings{})
	deal(s, 2)

	if _, ok := s.LastSwiped(people[0]); ok {
		t.Fatal("LastSwiped() before any swipe should report nothing")
	}
	current, ok := s.CurrentMovie(people[0])
	if !ok || current.ID != 1 {
		t.Fatalf("CurrentMovie() = %v, %v, want movie 1", current, ok)
	}

	s.RecordSwipe(people[0], VerdictLike)
	if people[0].Cursor != 1 || s.Likes(1) != 1 {
		t.Fatalf("after a like: cursor %d, likes %d", people[0].Cursor, s.Likes(1))
	}

	s.RevokeSwipe(people[0])
	if people[0].Cursor != 0 || s.Likes(1) != 0 {
		t.Fatalf("after undo: cursor %d, likes %d", people[0].Cursor, s.Likes(1))
	}
}

func TestEvaluateRipensAMatch(t *testing.T) {
	s, people := testSession(t, 2, Settings{MatchMode: MatchModeAll})
	deal(s, 2)

	for _, p := range people {
		s.RecordSwipe(p, VerdictLike)
	}
	outcome := s.Evaluate(testStart)
	if len(outcome.Matches) != 1 || outcome.Matches[0].Movie.ID != 1 {
		t.Fatalf("Evaluate() matches = %v, want movie 1", outcome.Matches)
	}
	if !s.IsMatched(1) || outcome.Matches[0].MatchedAt != testStart {
		t.Fatal("match not recorded with its timestamp")
	}
	if len(outcome.Matches[0].Votes) != 2 {
		t.Fatalf("match votes = %v, want both participants", outcome.Matches[0].Votes)
	}
	if outcome.Finished {
		t.Fatal("half a deck must not finish an all-mode session")
	}
	if again := s.Evaluate(testStart); len(again.Matches) != 0 {
		t.Fatalf("second Evaluate() = %v, want no new matches", again.Matches)
	}
}

func TestEvaluateStopsAtTarget(t *testing.T) {
	s, people := testSession(t, 2, Settings{MatchMode: MatchModeFirst})
	deal(s, 2)

	// Both movies are fully liked before evaluation, the "first match" mode
	// must still produce exactly one match and finish there.
	for _, p := range people {
		s.RecordSwipe(p, VerdictLike)
		s.RecordSwipe(p, VerdictLike)
	}
	outcome := s.Evaluate(testStart)
	if len(outcome.Matches) != 1 || outcome.Matches[0].Movie.ID != 1 {
		t.Fatalf("Evaluate() matches = %v, want only the first movie", outcome.Matches)
	}
	if !outcome.Finished || s.Status != StatusFinished {
		t.Fatal("reaching the target must finish the session")
	}
}

func TestEvaluateThresholdMatch(t *testing.T) {
	s, people := testSession(t, 3, Settings{MatchMode: MatchModeAll, Threshold: 2})
	deal(s, 1)

	s.RecordSwipe(people[0], VerdictLike)
	s.RecordSwipe(people[1], VerdictLike)
	s.RecordSwipe(people[2], VerdictNope)

	outcome := s.Evaluate(testStart)
	if len(outcome.Matches) != 1 {
		t.Fatalf("Evaluate() = %v, want one threshold match", outcome.Matches)
	}
	if !outcome.Finished {
		t.Fatal("a fully swiped all-mode deck must finish")
	}
}

func TestEvaluateDeckExhausted(t *testing.T) {
	s, people := testSession(t, 2, Settings{MatchMode: MatchModeFirst})
	deal(s, 1)

	for _, p := range people {
		s.RecordSwipe(p, VerdictNope)
	}
	outcome := s.Evaluate(testStart)
	if !outcome.DeckExhausted || outcome.Finished {
		t.Fatalf("outcome = %+v, want exhausted but not finished while the source has more", outcome)
	}

	// With the source dry the same spot finishes the session instead.
	s.SourceDone = true
	outcome = s.Evaluate(testStart)
	if !outcome.Finished || s.Status != StatusFinished {
		t.Fatalf("outcome = %+v, want finished once the source is dry", outcome)
	}
}

func TestAllDone(t *testing.T) {
	s, people := testSession(t, 2, Settings{})
	if s.AllDone() {
		t.Fatal("AllDone() must be false in the lobby")
	}
	deal(s, 1)
	s.RecordSwipe(people[0], VerdictNope)
	if s.AllDone() {
		t.Fatal("AllDone() with one participant mid-deck")
	}
	s.RecordSwipe(people[1], VerdictNope)
	if !s.AllDone() {
		t.Fatal("AllDone() after everyone swiped the whole deck")
	}
}

func TestNearMissAndResults(t *testing.T) {
	s, people := testSession(t, 3, Settings{MatchMode: MatchModeAll})
	deal(s, 2)

	// Movie 1: one like short of unanimous. Movie 2: fully rejected.
	s.RecordSwipe(people[0], VerdictLike)
	s.RecordSwipe(people[1], VerdictLike)
	s.RecordSwipe(people[2], VerdictNope)
	for _, p := range people {
		s.RecordSwipe(p, VerdictNope)
	}

	if got := s.NearMissCount(); got != 1 {
		t.Fatalf("NearMissCount() = %d, want 1", got)
	}

	results := s.BuildResults()
	if len(results.NearMatches) != 1 || results.NearMatches[0].Movie.ID != 1 {
		t.Fatalf("NearMatches = %v, want movie 1", results.NearMatches)
	}
	near := results.NearMatches[0]
	if near.Likes != 2 || near.Needed != 1 || len(near.Votes) != 3 {
		t.Fatalf("near match = %+v, want 2 likes, 1 needed, 3 votes revealed", near)
	}
	if len(results.Stats) != 3 || results.Stats[0].Likes != 1 || results.Stats[0].Seen != 2 {
		t.Fatalf("stats = %v", results.Stats)
	}
}

func TestFinishIsIdempotent(t *testing.T) {
	s, _ := testSession(t, 2, Settings{})
	deal(s, 1)

	if !s.Finish(testStart) {
		t.Fatal("first Finish() should transition")
	}
	if s.Status != StatusFinished || s.Results == nil || s.FinishedAt != testStart {
		t.Fatalf("finish left status %q, results %v", s.Status, s.Results)
	}
	if s.Finish(testStart.Add(time.Hour)) {
		t.Fatal("second Finish() must be a no-op")
	}
}

func TestRemoveParticipant(t *testing.T) {
	s, people := testSession(t, 3, Settings{})
	deal(s, 1)
	s.RecordSwipe(people[2], VerdictLike)
	s.StartContinueVote(people[0], 50)
	s.RecordContinueVote(people[2], true)

	s.RemoveParticipant(people[2])

	if len(s.Participants) != 2 || s.ParticipantByID("p3") != nil {
		t.Fatalf("participants after removal = %v", s.Participants)
	}
	if len(s.Votes[1]) != 0 {
		t.Fatalf("votes of the removed participant survived: %v", s.Votes[1])
	}
	if _, voted := s.Continue.Votes["p3"]; voted {
		t.Fatal("continue vote of the removed participant survived")
	}
	if len(s.Departed) != 1 || s.Departed[0].ID != "p3" || s.Departed[0].Progress != 1 {
		t.Fatalf("departed = %v, want p3 with progress 1", s.Departed)
	}
}

func TestRemoveParticipantInLobbyLeavesNoTrace(t *testing.T) {
	s, people := testSession(t, 2, Settings{})
	s.RemoveParticipant(people[1])
	if len(s.Departed) != 0 {
		t.Fatalf("lobby departure left a trace: %v", s.Departed)
	}
}

func TestSyncPresence(t *testing.T) {
	s, people := testSession(t, 2, Settings{})
	if s.EmptySince != testStart {
		t.Fatalf("a fresh session with nobody online must start its empty clock, got %v", s.EmptySince)
	}

	people[0].Online = true
	s.SyncPresence(testStart.Add(time.Minute))
	if !s.EmptySince.IsZero() {
		t.Fatal("EmptySince must clear while anyone is online")
	}

	people[0].Online = false
	emptyAt := testStart.Add(2 * time.Minute)
	s.SyncPresence(emptyAt)
	s.SyncPresence(emptyAt.Add(time.Minute))
	if s.EmptySince != emptyAt {
		t.Fatalf("EmptySince = %v, want the first moment the room emptied %v", s.EmptySince, emptyAt)
	}
}

func TestDeckMembership(t *testing.T) {
	s, _ := testSession(t, 2, Settings{})
	movies := deal(s, 2)
	if !s.InDeck(movies[0].ID) || s.InDeck(99) {
		t.Fatal("InDeck() membership is wrong")
	}
	ids := s.DeckIDs()
	ids[99] = struct{}{}
	if s.InDeck(99) {
		t.Fatal("DeckIDs() must return a copy, not the live set")
	}
}

func TestMovieVotesHidesDeparted(t *testing.T) {
	s, people := testSession(t, 2, Settings{})
	deal(s, 1)
	s.RecordSwipe(people[0], VerdictLike)
	s.RecordSwipe(people[1], VerdictLike)
	s.RemoveParticipant(people[1])

	votes := s.MovieVotes(1)
	if len(votes) != 1 {
		t.Fatalf("MovieVotes() = %v, want only current participants", votes)
	}
}
