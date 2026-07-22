package domain

import "time"

type Status string

const (
	StatusLobby    Status = "lobby"
	StatusActive   Status = "active"
	StatusVoting   Status = "voting"
	StatusFinished Status = "finished"
)

func AllStatuses() []Status {
	return []Status{StatusLobby, StatusActive, StatusVoting, StatusFinished}
}

type MatchMode string

const (
	MatchModeFirst MatchMode = "first"
	MatchModeCount MatchMode = "count"
	MatchModeAll   MatchMode = "all"
)

type Verdict string

const (
	VerdictLike Verdict = "like"
	VerdictNope Verdict = "nope"
)

func (v Verdict) Valid() bool { return v == VerdictLike || v == VerdictNope }

type Settings struct {
	MediaType   MediaType
	PresetID    string
	MatchMode   MatchMode
	MatchTarget int
	// Threshold is how many likes make a match; 0 means unanimous.
	Threshold int
	DeckLimit int
	Filters   Filters
}

type ContinueVote struct {
	ProposedLimit int
	Votes         map[string]bool
}

type Session struct {
	Code         string
	Settings     Settings
	Status       Status
	Round        int
	Deck         []Movie
	deckIDs      map[int64]struct{}
	Participants []*Participant
	Departed     []Departed
	Votes        map[int64]map[string]Verdict
	Matches      []Match
	Requested    []int64
	Continue     *ContinueVote
	Results      *Results

	SourcePage int
	SourceDone bool

	// LastSeen is contact from any client, reads included. EmptySince is
	// when the last socket dropped, zero while anyone is connected.
	// FinishedAt survives the reads that keep LastSeen fresh, so results
	// expire on their own clock.
	LastSeen   time.Time
	EmptySince time.Time
	FinishedAt time.Time
}

func NewSession(code string, settings Settings, host *Participant, now time.Time) *Session {
	s := &Session{
		Code:         code,
		Settings:     settings,
		Status:       StatusLobby,
		Participants: []*Participant{host},
		Votes:        make(map[int64]map[string]Verdict),
		deckIDs:      make(map[int64]struct{}),
		SourcePage:   1,
		LastSeen:     now,
	}
	s.SyncPresence(now)
	return s
}

func (s *Session) Touch(now time.Time) { s.LastSeen = now }

// SyncPresence keeps EmptySince in step with who is connected. Every path
// that changes the online set has to run it, the reaper trusts the field.
func (s *Session) SyncPresence(now time.Time) {
	if s.AnyOnline() {
		s.EmptySince = time.Time{}
		return
	}
	if s.EmptySince.IsZero() {
		s.EmptySince = now
	}
}

func (s *Session) BeginRound(movies []Movie, nextPage int, exhausted bool) {
	s.addToDeck(movies)
	s.SourcePage = nextPage
	s.SourceDone = exhausted
	s.Status = StatusActive
	s.Round = 1
}

func (s *Session) ExtendDeck(movies []Movie, nextPage int, exhausted bool) {
	s.addToDeck(movies)
	s.SourcePage = nextPage
	s.SourceDone = exhausted
	s.Continue = nil
	s.Status = StatusActive
	s.Round++
}

// ExhaustContinue returns the session to active instead of finishing it:
// the caller must emit the vote result from the active state first, then
// finish.
func (s *Session) ExhaustContinue() {
	s.SourceDone = true
	s.Continue = nil
	s.Status = StatusActive
}

func (s *Session) StartContinueVote(host *Participant, limit int) {
	s.Status = StatusVoting
	s.Continue = &ContinueVote{
		ProposedLimit: limit,
		Votes:         map[string]bool{host.ID: true},
	}
}

func (s *Session) CancelContinueVote() {
	s.Continue = nil
	s.Status = StatusActive
}

func (s *Session) RecordContinueVote(p *Participant, accept bool) {
	s.Continue.Votes[p.ID] = accept
}

func (s *Session) AllAccepted() bool {
	if s.Continue == nil {
		return false
	}
	for _, p := range s.Participants {
		if !s.Continue.Votes[p.ID] {
			return false
		}
	}
	return true
}

func (s *Session) Finish(now time.Time) bool {
	if s.Status == StatusFinished {
		return false
	}
	s.Status = StatusFinished
	s.Continue = nil
	s.Results = s.BuildResults()
	s.FinishedAt = now
	return true
}

func (s *Session) MarkRequested(movieID int64) {
	s.Requested = append(s.Requested, movieID)
}

func (s *Session) addToDeck(movies []Movie) {
	s.Deck = append(s.Deck, movies...)
	for _, m := range movies {
		s.deckIDs[m.ID] = struct{}{}
	}
}

func (s *Session) InDeck(movieID int64) bool {
	_, ok := s.deckIDs[movieID]
	return ok
}

func (s *Session) DeckIDs() map[int64]struct{} {
	out := make(map[int64]struct{}, len(s.deckIDs))
	for id := range s.deckIDs {
		out[id] = struct{}{}
	}
	return out
}
