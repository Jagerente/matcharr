package domain

import "time"

const maxNearMatches = 10

func (s *Session) CurrentMovie(p *Participant) (Movie, bool) {
	if p.Cursor >= len(s.Deck) {
		return Movie{}, false
	}
	return s.Deck[p.Cursor], true
}

func (s *Session) LastSwiped(p *Participant) (Movie, bool) {
	if p.Cursor == 0 {
		return Movie{}, false
	}
	return s.Deck[p.Cursor-1], true
}

func (s *Session) RecordSwipe(p *Participant, verdict Verdict) {
	movie := s.Deck[p.Cursor]
	if s.Votes[movie.ID] == nil {
		s.Votes[movie.ID] = make(map[string]Verdict)
	}
	s.Votes[movie.ID][p.ID] = verdict
	p.Cursor++
}

func (s *Session) RevokeSwipe(p *Participant) {
	movie := s.Deck[p.Cursor-1]
	delete(s.Votes[movie.ID], p.ID)
	p.Cursor--
}

func (s *Session) Threshold() int {
	n := len(s.Participants)
	if t := s.Settings.Threshold; t > 0 && t < n {
		return t
	}
	return n
}

func (s *Session) Likes(movieID int64) int {
	count := 0
	for _, p := range s.Participants {
		if s.Votes[movieID][p.ID] == VerdictLike {
			count++
		}
	}
	return count
}

func (s *Session) IsMatched(movieID int64) bool {
	for _, m := range s.Matches {
		if m.Movie.ID == movieID {
			return true
		}
	}
	return false
}

// MovieVotes filters the raw ballot down to current participants, so a
// departed swiper never shows up in a reveal.
func (s *Session) MovieVotes(movieID int64) map[string]Verdict {
	out := make(map[string]Verdict)
	for _, p := range s.Participants {
		if v, ok := s.Votes[movieID][p.ID]; ok {
			out[p.ID] = v
		}
	}
	return out
}

func (s *Session) AllDone() bool {
	if s.Status == StatusLobby || len(s.Participants) == 0 {
		return false
	}
	for _, p := range s.Participants {
		if p.Cursor < len(s.Deck) {
			return false
		}
	}
	return true
}

func (s *Session) TargetReached() bool {
	switch s.Settings.MatchMode {
	case MatchModeFirst:
		return len(s.Matches) >= 1
	case MatchModeCount:
		return len(s.Matches) >= s.Settings.MatchTarget
	default:
		return false
	}
}

func (s *Session) NearMissCount() int {
	if s.Status == StatusLobby {
		return 0
	}
	threshold := s.Threshold()
	count := 0
	for _, movie := range s.Deck {
		if s.IsMatched(movie.ID) {
			continue
		}
		if s.Likes(movie.ID) == threshold-1 {
			count++
		}
	}
	return count
}

type Outcome struct {
	Matches       []Match
	Finished      bool
	DeckExhausted bool
}

func (s *Session) Evaluate(now time.Time) Outcome {
	if s.Status != StatusActive {
		return Outcome{}
	}

	out := Outcome{Matches: s.ripenMatches(now)}
	if s.TargetReached() {
		s.Finish(now)
		out.Finished = true
		return out
	}

	if s.AllDone() {
		if s.Settings.MatchMode == MatchModeAll || s.SourceDone {
			s.Finish(now)
			out.Finished = true
			return out
		}
		out.DeckExhausted = true
	}
	return out
}

// ripenMatches turns every movie that reached the like threshold into a
// match, scanning in deck order so "until the first match" picks the
// earliest card. The scan stops once the match target is reached.
func (s *Session) ripenMatches(now time.Time) []Match {
	threshold := s.Threshold()
	var ripened []Match
	for _, movie := range s.Deck {
		if s.IsMatched(movie.ID) || s.Likes(movie.ID) < threshold {
			continue
		}
		match := Match{Movie: movie, MatchedAt: now, Votes: s.MovieVotes(movie.ID)}
		s.Matches = append(s.Matches, match)
		ripened = append(ripened, match)
		if s.TargetReached() {
			break
		}
	}
	return ripened
}

func (s *Session) BuildResults() *Results {
	threshold := s.Threshold()

	// Near-matches are strictly one vote short, per the design the backups
	// worth naming the hold-out for.
	near := make([]NearMatch, 0)
	for _, movie := range s.Deck {
		if s.IsMatched(movie.ID) {
			continue
		}
		if likes := s.Likes(movie.ID); likes == threshold-1 && likes > 0 {
			near = append(near, NearMatch{
				Movie:  movie,
				Likes:  likes,
				Needed: 1,
				Votes:  s.MovieVotes(movie.ID),
			})
		}
	}
	if len(near) > maxNearMatches {
		near = near[:maxNearMatches]
	}

	stats := make([]ParticipantStats, 0, len(s.Participants))
	for _, p := range s.Participants {
		likes := 0
		for _, votes := range s.Votes {
			if votes[p.ID] == VerdictLike {
				likes++
			}
		}
		stats = append(stats, ParticipantStats{ID: p.ID, Name: p.Name, Likes: likes, Seen: p.Cursor})
	}

	matches := make([]Match, len(s.Matches))
	copy(matches, s.Matches)

	return &Results{Matches: matches, NearMatches: near, Stats: stats}
}
