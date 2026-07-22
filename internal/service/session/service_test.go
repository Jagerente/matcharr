package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

var testStart = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type fakeDecks struct {
	mu    sync.Mutex
	build func(req deck.Request) (deck.Result, error)
	calls []deck.Request
}

func (f *fakeDecks) HasPreset(id string) bool { return id == "p" }

func (f *fakeDecks) Build(_ context.Context, req deck.Request) (deck.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req)
	build := f.build
	f.mu.Unlock()
	return build(req)
}

func (f *fakeDecks) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeDecks) lastCall(t *testing.T) deck.Request {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		t.Fatal("no deck builds recorded")
	}
	return f.calls[len(f.calls)-1]
}

func servesMovies(from, count int) func(deck.Request) (deck.Result, error) {
	return func(req deck.Request) (deck.Result, error) {
		movies := make([]domain.Movie, 0, count)
		for i := range count {
			movies = append(movies, domain.Movie{ID: int64(from + i), Title: fmt.Sprintf("Movie %d", from+i)})
		}
		return deck.Result{Movies: movies, NextPage: req.StartPage + 1}, nil
	}
}

type recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *recorder) record(_ string, ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *recorder) types() []EventType {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]EventType, 0, len(r.events))
	for _, ev := range r.events {
		out = append(out, ev.Type)
	}
	return out
}

func (r *recorder) last(t *testing.T, evType EventType) Event {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ev := range slices.Backward(r.events) {
		if ev.Type == evType {
			return ev
		}
	}
	t.Fatalf("no %q event recorded, got %v", evType, r.events)
	return Event{}
}

func (r *recorder) waitFor(t *testing.T, evType EventType) Event {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for _, ev := range r.events {
			if ev.Type == evType {
				r.mu.Unlock()
				return ev
			}
		}
		r.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for a %q event, got %v", evType, r.types())
	return Event{}
}

func testLimits() Limits {
	return Limits{
		TTL:             24 * time.Hour,
		EmptyGrace:      15 * time.Minute,
		FinishedTTL:     time.Hour,
		MaxParticipants: 8,
		MaxDeck:         200,
	}
}

func newTestService(t *testing.T, decks *fakeDecks, limits Limits) (*Service, *recorder, *fakeClock) {
	t.Helper()
	if decks.build == nil {
		decks.build = servesMovies(1, 2)
	}
	clock := &fakeClock{t: testStart}
	svc := New(decks, limits, slog.New(slog.DiscardHandler), WithClock(clock.Now))
	events := &recorder{}
	svc.Subscribe(events.record)
	return svc, events, clock
}

func mustCreate(t *testing.T, svc *Service, name string, settings domain.Settings) Credentials {
	t.Helper()
	if settings.PresetID == "" {
		settings.PresetID = "p"
	}
	creds, err := svc.Create(CreateInput{Name: name, Settings: settings})
	if err != nil {
		t.Fatal(err)
	}
	return creds
}

func mustJoin(t *testing.T, svc *Service, code, name, deviceID string) Credentials {
	t.Helper()
	creds, err := svc.Join(code, JoinInput{Name: name, DeviceID: deviceID})
	if err != nil {
		t.Fatal(err)
	}
	return creds
}

func startedPair(t *testing.T, svc *Service, settings domain.Settings) (host, guest Credentials) {
	t.Helper()
	host = mustCreate(t, svc, "Host", settings)
	guest = mustJoin(t, svc, host.Code, "Guest", "")
	if err := svc.Start(context.Background(), host.Code, host.Token); err != nil {
		t.Fatal(err)
	}
	return host, guest
}

func swipeAll(t *testing.T, svc *Service, code string, creds Credentials, verdicts ...domain.Verdict) {
	t.Helper()
	movies, err := svc.Deck(code, creds.Token)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range verdicts {
		if err := svc.Swipe(code, creds.Token, movies[i].ID, v); err != nil {
			t.Fatal(err)
		}
	}
}

func wantReason(t *testing.T, err error, reason apperr.Reason) {
	t.Helper()
	appErr := apperr.From(err)
	if appErr == nil {
		t.Fatalf("error %v is not an application error, want reason %q", err, reason)
	}
	if appErr.Reason != reason {
		t.Fatalf("reason = %q (%v), want %q", appErr.Reason, err, reason)
	}
}

func TestCreateValidation(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())

	tests := []struct {
		name   string
		input  CreateInput
		reason apperr.Reason
	}{
		{"empty name", CreateInput{Settings: domain.Settings{PresetID: "p"}}, apperr.ReasonNameRequired},
		{"long name", CreateInput{Name: string(make([]rune, 25)), Settings: domain.Settings{PresetID: "p"}}, apperr.ReasonNameTooLong},
		{"unknown preset", CreateInput{Name: "A", Settings: domain.Settings{PresetID: "nope"}}, apperr.ReasonUnknownPreset},
		{"deck limit too high", CreateInput{Name: "A", Settings: domain.Settings{PresetID: "p", DeckLimit: 500}}, apperr.ReasonBadSettings},
		{"threshold of one", CreateInput{Name: "A", Settings: domain.Settings{PresetID: "p", Threshold: 1}}, apperr.ReasonBadSettings},
		{"count mode without target", CreateInput{Name: "A", Settings: domain.Settings{PresetID: "p", MatchMode: domain.MatchModeCount}}, apperr.ReasonBadSettings},
		{"inverted years", CreateInput{Name: "A", Settings: domain.Settings{PresetID: "p", Filters: domain.Filters{YearFrom: 2000, YearTo: 1990}}}, apperr.ReasonBadSettings},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(tt.input)
			wantReason(t, err, tt.reason)
		})
	}
}

func TestCreateAppliesDefaults(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	creds := mustCreate(t, svc, "Host", domain.Settings{})

	view, err := svc.State(creds.Code, creds.Token)
	if err != nil {
		t.Fatal(err)
	}
	settings := view.State.Settings
	if settings.MatchMode != domain.MatchModeFirst || settings.DeckLimit != 50 || settings.MediaType != domain.MediaTypeMovie {
		t.Fatalf("defaults not applied: %+v", settings)
	}
	if !view.You.IsHost || view.State.Status != domain.StatusLobby {
		t.Fatalf("host view = %+v", view)
	}
}

func TestJoinGuards(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())

	_, err := svc.Join("XXXX", JoinInput{Name: "A"})
	wantReason(t, err, apperr.ReasonSessionNotFound)

	host, _ := startedPair(t, svc, domain.Settings{})
	_, err = svc.Join(host.Code, JoinInput{Name: "Late"})
	wantReason(t, err, apperr.ReasonSessionStarted)

	if err := svc.Finish(host.Code, host.Token); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Join(host.Code, JoinInput{Name: "Late"})
	wantReason(t, err, apperr.ReasonSessionFinished)
}

func TestJoinSessionFull(t *testing.T) {
	limits := testLimits()
	limits.MaxParticipants = 2
	svc, _, _ := newTestService(t, &fakeDecks{}, limits)

	host := mustCreate(t, svc, "Host", domain.Settings{})
	mustJoin(t, svc, host.Code, "Guest", "")
	_, err := svc.Join(host.Code, JoinInput{Name: "Third"})
	wantReason(t, err, apperr.ReasonSessionFull)
}

func TestJoinDeviceReclaims(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})

	first := mustJoin(t, svc, host.Code, "Guest", "device-1")
	second := mustJoin(t, svc, host.Code, "Renamed", "device-1")

	if first.You.ID != second.You.ID {
		t.Fatal("the same device must reclaim its participant")
	}
	if first.Token == second.Token {
		t.Fatal("a reclaim must rotate the token")
	}
	view, err := svc.State(host.Code, second.Token)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.State.Participants) != 2 {
		t.Fatalf("participants = %v, want no duplicates", view.State.Participants)
	}
	if _, err := svc.State(host.Code, first.Token); err == nil {
		t.Fatal("the old token must die on reclaim")
	}
}

func TestStartGuards(t *testing.T) {
	decks := &fakeDecks{}
	svc, _, _ := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})

	err := svc.Start(context.Background(), host.Code, host.Token)
	wantReason(t, err, apperr.ReasonNotEnoughPlayers)

	guest := mustJoin(t, svc, host.Code, "Guest", "")
	err = svc.Start(context.Background(), host.Code, guest.Token)
	wantReason(t, err, apperr.ReasonHostOnly)
}

func TestStartBuilderFailure(t *testing.T) {
	decks := &fakeDecks{build: func(deck.Request) (deck.Result, error) {
		return deck.Result{}, errors.New("tmdb down")
	}}
	svc, _, _ := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})
	mustJoin(t, svc, host.Code, "Guest", "")

	err := svc.Start(context.Background(), host.Code, host.Token)
	wantReason(t, err, apperr.ReasonSourceUnavailable)

	view, _ := svc.State(host.Code, host.Token)
	if view.State.Status != domain.StatusLobby {
		t.Fatal("a failed start must leave the lobby intact")
	}
}

func TestStartEmptySource(t *testing.T) {
	decks := &fakeDecks{build: func(deck.Request) (deck.Result, error) {
		return deck.Result{Exhausted: true}, nil
	}}
	svc, _, _ := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})
	mustJoin(t, svc, host.Code, "Guest", "")

	err := svc.Start(context.Background(), host.Code, host.Token)
	wantReason(t, err, apperr.ReasonEmptySource)
}

func TestSwipeToFirstMatch(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst})

	swipeAll(t, svc, host.Code, host, domain.VerdictLike)
	swipeAll(t, svc, host.Code, guest, domain.VerdictLike)

	match := events.last(t, EventMatchFound)
	if data := match.Data.(MatchFoundData); data.Movie.ID != 1 {
		t.Fatalf("match data = %+v, want movie 1", match.Data)
	}
	finished := events.last(t, EventSessionFinished)
	if finished.State.Status != domain.StatusFinished || finished.State.Results == nil {
		t.Fatalf("final state = %+v", finished.State)
	}
	if len(finished.State.Matches) != 1 {
		t.Fatalf("matches = %v", finished.State.Matches)
	}
}

func TestSwipeGuards(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, _ := startedPair(t, svc, domain.Settings{})

	err := svc.Swipe(host.Code, host.Token, 1, "maybe")
	wantReason(t, err, apperr.ReasonBadRequest)

	err = svc.Swipe(host.Code, host.Token, 2, domain.VerdictLike)
	wantReason(t, err, apperr.ReasonOutOfOrder)

	swipeAll(t, svc, host.Code, host, domain.VerdictNope, domain.VerdictNope)
	err = svc.Swipe(host.Code, host.Token, 1, domain.VerdictLike)
	wantReason(t, err, apperr.ReasonDeckFinished)
}

func TestUndo(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst})

	err := svc.Undo(host.Code, host.Token)
	wantReason(t, err, apperr.ReasonNothingToUndo)

	swipeAll(t, svc, host.Code, host, domain.VerdictLike)
	if err := svc.Undo(host.Code, host.Token); err != nil {
		t.Fatal(err)
	}

	swipeAll(t, svc, host.Code, host, domain.VerdictLike)
	swipeAll(t, svc, host.Code, guest, domain.VerdictLike)
	err = svc.Undo(host.Code, host.Token)
	wantReason(t, err, apperr.ReasonBadState)
}

func TestUndoMatchedMovie(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeAll})

	swipeAll(t, svc, host.Code, host, domain.VerdictLike)
	swipeAll(t, svc, host.Code, guest, domain.VerdictLike)

	err := svc.Undo(host.Code, guest.Token)
	wantReason(t, err, apperr.ReasonAlreadyMatched)
}

func TestRunToResults(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeAll})

	swipeAll(t, svc, host.Code, host, domain.VerdictLike, domain.VerdictLike)
	swipeAll(t, svc, host.Code, guest, domain.VerdictLike, domain.VerdictNope)

	finished := events.last(t, EventSessionFinished)
	results := finished.State.Results
	if results == nil || len(results.Matches) != 1 || len(results.NearMatches) != 1 {
		t.Fatalf("results = %+v", results)
	}
}

func TestContinueFlow(t *testing.T) {
	decks := &fakeDecks{}
	svc, events, _ := newTestService(t, decks, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst, DeckLimit: 10})

	swipeAll(t, svc, host.Code, host, domain.VerdictNope, domain.VerdictNope)
	swipeAll(t, svc, host.Code, guest, domain.VerdictNope, domain.VerdictNope)
	events.last(t, EventDeckExhausted)

	err := svc.ProposeContinue(host.Code, guest.Token, 0)
	wantReason(t, err, apperr.ReasonHostOnly)

	if err := svc.ProposeContinue(host.Code, host.Token, 0); err != nil {
		t.Fatal(err)
	}
	proposed := events.last(t, EventContinueProposed)
	if proposed.State.Continue == nil || proposed.State.Continue.ProposedLimit != 10 {
		t.Fatalf("proposal state = %+v, want the session deck limit as default", proposed.State.Continue)
	}

	decks.build = servesMovies(3, 2)
	if err := svc.VoteContinue(context.Background(), host.Code, guest.Token, true); err != nil {
		t.Fatal(err)
	}

	result := events.last(t, EventContinueResult)
	data := result.Data.(ContinueResultData)
	if !data.Accepted || !data.Extended || data.Added != 2 {
		t.Fatalf("continue result = %+v", data)
	}
	if result.State.Status != domain.StatusActive || result.State.Round != 2 || result.State.DeckSize != 4 {
		t.Fatalf("state after extension = %+v", result.State)
	}
	if req := decks.lastCall(t); len(req.Exclude) != 2 || req.StartPage != 2 {
		t.Fatalf("extension request = %+v, want the dealt movies excluded and paging resumed", req)
	}
}

func TestContinueRejected(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst})

	swipeAll(t, svc, host.Code, host, domain.VerdictNope, domain.VerdictNope)
	swipeAll(t, svc, host.Code, guest, domain.VerdictNope, domain.VerdictNope)
	if err := svc.ProposeContinue(host.Code, host.Token, 0); err != nil {
		t.Fatal(err)
	}

	if err := svc.VoteContinue(context.Background(), host.Code, guest.Token, false); err != nil {
		t.Fatal(err)
	}
	result := events.last(t, EventContinueResult)
	if data := result.Data.(ContinueResultData); data.Accepted {
		t.Fatalf("continue result = %+v, want rejected", data)
	}
	events.last(t, EventSessionFinished)
}

func TestContinueSourceDry(t *testing.T) {
	decks := &fakeDecks{}
	svc, events, _ := newTestService(t, decks, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst})

	swipeAll(t, svc, host.Code, host, domain.VerdictNope, domain.VerdictNope)
	swipeAll(t, svc, host.Code, guest, domain.VerdictNope, domain.VerdictNope)
	if err := svc.ProposeContinue(host.Code, host.Token, 0); err != nil {
		t.Fatal(err)
	}

	decks.build = func(deck.Request) (deck.Result, error) {
		return deck.Result{Exhausted: true}, nil
	}
	if err := svc.VoteContinue(context.Background(), host.Code, guest.Token, true); err != nil {
		t.Fatal(err)
	}

	result := events.last(t, EventContinueResult)
	if data := result.Data.(ContinueResultData); !data.Accepted || data.Extended {
		t.Fatalf("continue result = %+v, want accepted but not extended", data)
	}
	finished := events.last(t, EventSessionFinished)
	if !finished.State.SourceExhausted {
		t.Fatal("a dry source must be recorded")
	}
}

func TestCancelContinue(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{MatchMode: domain.MatchModeFirst})

	swipeAll(t, svc, host.Code, host, domain.VerdictNope, domain.VerdictNope)
	swipeAll(t, svc, host.Code, guest, domain.VerdictNope, domain.VerdictNope)
	if err := svc.ProposeContinue(host.Code, host.Token, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.CancelContinue(host.Code, host.Token); err != nil {
		t.Fatal(err)
	}
	cancelled := events.last(t, EventContinueCancelled)
	if cancelled.State.Status != domain.StatusActive || cancelled.State.Continue != nil {
		t.Fatalf("state after cancel = %+v", cancelled.State)
	}
}

func TestKickResolvesContinueVote(t *testing.T) {
	decks := &fakeDecks{}
	svc, events, _ := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{MatchMode: domain.MatchModeFirst})
	g2 := mustJoin(t, svc, host.Code, "G2", "")
	g3 := mustJoin(t, svc, host.Code, "G3", "")
	if err := svc.Start(context.Background(), host.Code, host.Token); err != nil {
		t.Fatal(err)
	}

	for _, creds := range []Credentials{host, g2, g3} {
		swipeAll(t, svc, host.Code, creds, domain.VerdictNope, domain.VerdictNope)
	}
	if err := svc.ProposeContinue(host.Code, host.Token, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.VoteContinue(context.Background(), host.Code, g2.Token, true); err != nil {
		t.Fatal(err)
	}

	decks.build = servesMovies(3, 2)
	if err := svc.Kick(host.Code, host.Token, g3.You.ID); err != nil {
		t.Fatal(err)
	}

	result := events.waitFor(t, EventContinueResult)
	if data := result.Data.(ContinueResultData); !data.Extended {
		t.Fatalf("continue result = %+v, want the kick to complete the vote", data)
	}
}

func TestKickDropsThresholdAndRipens(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{MatchMode: domain.MatchModeFirst})
	g2 := mustJoin(t, svc, host.Code, "G2", "")
	g3 := mustJoin(t, svc, host.Code, "G3", "")
	if err := svc.Start(context.Background(), host.Code, host.Token); err != nil {
		t.Fatal(err)
	}

	swipeAll(t, svc, host.Code, host, domain.VerdictLike)
	swipeAll(t, svc, host.Code, g2, domain.VerdictLike)
	swipeAll(t, svc, host.Code, g3, domain.VerdictNope)

	if err := svc.Kick(host.Code, host.Token, g3.You.ID); err != nil {
		t.Fatal(err)
	}
	events.last(t, EventMatchFound)
	events.last(t, EventSessionFinished)
}

func TestKickGuards(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})
	guest := mustJoin(t, svc, host.Code, "Guest", "")

	err := svc.Kick(host.Code, guest.Token, host.You.ID)
	wantReason(t, err, apperr.ReasonHostOnly)
	err = svc.Kick(host.Code, host.Token, host.You.ID)
	wantReason(t, err, apperr.ReasonSelfKick)
	err = svc.Kick(host.Code, host.Token, "ghost")
	wantReason(t, err, apperr.ReasonNoParticipant)
}

func TestLeave(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{})

	err := svc.Leave(host.Code, host.Token)
	wantReason(t, err, apperr.ReasonHostCannotLeave)

	if err := svc.Leave(host.Code, guest.Token); err != nil {
		t.Fatal(err)
	}
	left := events.last(t, EventParticipantLeft)
	if data := left.Data.(ParticipantData); data.ParticipantID != guest.You.ID {
		t.Fatalf("leave data = %+v", data)
	}
	// One participant cannot match anything: the session ends.
	finished := events.last(t, EventSessionFinished)
	if len(finished.State.Departed) != 1 {
		t.Fatalf("departed = %v", finished.State.Departed)
	}
}

func TestMarkRequested(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{})

	err := svc.MarkRequested(host.Code, guest.Token, 1)
	wantReason(t, err, apperr.ReasonHostOnly)

	if err := svc.MarkRequested(host.Code, host.Token, 1); err != nil {
		t.Fatal(err)
	}
	sent := events.last(t, EventRequestSent)
	if data := sent.Data.(RequestSentData); data.MovieID != 1 {
		t.Fatalf("request data = %+v", data)
	}

	before := len(events.types())
	if err := svc.MarkRequested(host.Code, host.Token, 1); err != nil {
		t.Fatal(err)
	}
	if len(events.types()) != before {
		t.Fatal("a repeated mark must not emit again")
	}
}

func TestAuthorization(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host, guest := startedPair(t, svc, domain.Settings{})

	_, err := svc.State(host.Code, "bad-token")
	wantReason(t, err, apperr.ReasonInvalidToken)

	err = svc.AuthorizeMovie(host.Code, guest.Token, 999)
	wantReason(t, err, apperr.ReasonUnknownMovie)
	if err := svc.AuthorizeMovie(host.Code, guest.Token, 1); err != nil {
		t.Fatal(err)
	}

	err = svc.AuthorizeRequest(host.Code, guest.Token, 1)
	wantReason(t, err, apperr.ReasonHostOnly)
	err = svc.AuthorizeHost(host.Code, guest.Token)
	wantReason(t, err, apperr.ReasonHostOnly)
}

func TestDeckGuards(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})

	_, err := svc.Deck(host.Code, host.Token)
	wantReason(t, err, apperr.ReasonBadState)
}

func TestExpiryReasons(t *testing.T) {
	t.Run("ttl", func(t *testing.T) {
		svc, _, clock := newTestService(t, &fakeDecks{}, testLimits())
		host := mustCreate(t, svc, "Host", domain.Settings{})
		svc.SetPresence(host.Code, host.You.ID, true)

		clock.Advance(25 * time.Hour)
		svc.store.Cleanup()
		_, err := svc.State(host.Code, host.Token)
		wantReason(t, err, apperr.ReasonSessionNotFound)
	})

	t.Run("abandoned", func(t *testing.T) {
		svc, _, clock := newTestService(t, &fakeDecks{}, testLimits())
		host := mustCreate(t, svc, "Host", domain.Settings{})

		clock.Advance(16 * time.Minute)
		svc.store.Cleanup()
		_, err := svc.State(host.Code, host.Token)
		wantReason(t, err, apperr.ReasonSessionNotFound)
	})

	t.Run("a connected client keeps the session alive", func(t *testing.T) {
		svc, _, clock := newTestService(t, &fakeDecks{}, testLimits())
		host := mustCreate(t, svc, "Host", domain.Settings{})
		svc.SetPresence(host.Code, host.You.ID, true)

		clock.Advance(16 * time.Minute)
		svc.store.Cleanup()
		if _, err := svc.State(host.Code, host.Token); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("finished results expire on their own clock", func(t *testing.T) {
		svc, _, clock := newTestService(t, &fakeDecks{}, testLimits())
		host := mustCreate(t, svc, "Host", domain.Settings{})
		svc.SetPresence(host.Code, host.You.ID, true)
		if err := svc.Finish(host.Code, host.Token); err != nil {
			t.Fatal(err)
		}

		clock.Advance(2 * time.Hour)
		svc.store.Cleanup()
		_, err := svc.State(host.Code, host.Token)
		wantReason(t, err, apperr.ReasonSessionNotFound)
	})
}

func TestSessionCeiling(t *testing.T) {
	limits := testLimits()
	limits.MaxSessions = 1
	svc, _, clock := newTestService(t, &fakeDecks{}, limits)

	first := mustCreate(t, svc, "Host", domain.Settings{})
	svc.SetPresence(first.Code, first.You.ID, true)

	_, err := svc.Create(CreateInput{Name: "B", Settings: domain.Settings{PresetID: "p"}})
	wantReason(t, err, apperr.ReasonServerBusy)

	// Once the first session rots, creating cleans it up instead of turning
	// a live host away over a dead slot.
	clock.Advance(25 * time.Hour)
	mustCreate(t, svc, "B", domain.Settings{})
}

func TestStartWhileFetching(t *testing.T) {
	release := make(chan struct{})
	decks := &fakeDecks{build: func(req deck.Request) (deck.Result, error) {
		<-release
		return servesMovies(1, 2)(req)
	}}
	svc, _, clock := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})
	mustJoin(t, svc, host.Code, "Guest", "")

	done := make(chan error, 1)
	go func() { done <- svc.Start(context.Background(), host.Code, host.Token) }()
	for decks.callCount() == 0 {
		time.Sleep(time.Millisecond)
	}

	err := svc.Start(context.Background(), host.Code, host.Token)
	wantReason(t, err, apperr.ReasonBadState)

	// The janitor must not reap a session whose deck is being fetched.
	clock.Advance(25 * time.Hour)
	svc.store.Cleanup()

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	view, err := svc.State(host.Code, host.Token)
	if err != nil {
		t.Fatal(err)
	}
	if view.State.Status != domain.StatusActive {
		t.Fatalf("status = %q, want active", view.State.Status)
	}
}

func TestFinishDuringFetchWins(t *testing.T) {
	release := make(chan struct{})
	decks := &fakeDecks{build: func(req deck.Request) (deck.Result, error) {
		<-release
		return servesMovies(1, 2)(req)
	}}
	svc, _, _ := newTestService(t, decks, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})
	mustJoin(t, svc, host.Code, "Guest", "")

	done := make(chan error, 1)
	go func() { done <- svc.Start(context.Background(), host.Code, host.Token) }()
	for decks.callCount() == 0 {
		time.Sleep(time.Millisecond)
	}
	if err := svc.Finish(host.Code, host.Token); err != nil {
		t.Fatal(err)
	}

	close(release)
	err := <-done
	wantReason(t, err, apperr.ReasonBadState)

	view, stateErr := svc.State(host.Code, host.Token)
	if stateErr != nil {
		t.Fatal(stateErr)
	}
	if view.State.Status != domain.StatusFinished || view.State.DeckSize != 0 {
		t.Fatalf("a cancelled session got resurrected: %+v", view.State)
	}
}

func TestPresenceEvents(t *testing.T) {
	svc, events, _ := newTestService(t, &fakeDecks{}, testLimits())
	host := mustCreate(t, svc, "Host", domain.Settings{})

	svc.SetPresence(host.Code, host.You.ID, true)
	update := events.last(t, EventLobbyUpdate)
	if !update.State.Participants[0].Online {
		t.Fatal("presence not reflected in the state")
	}

	before := len(events.types())
	svc.SetPresence(host.Code, host.You.ID, true)
	if len(events.types()) != before {
		t.Fatal("an unchanged presence must not emit")
	}
}

func TestStats(t *testing.T) {
	svc, _, _ := newTestService(t, &fakeDecks{}, testLimits())
	mustCreate(t, svc, "Host", domain.Settings{})

	stats := svc.Stats()
	if stats.Sessions[domain.StatusLobby] != 1 || stats.Participants != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if _, seeded := stats.Sessions[domain.StatusFinished]; !seeded {
		t.Fatal("every status must be seeded, even at zero")
	}
}
