package session

import "github.com/Jagerente/matcharr/internal/domain"

type EventType string

const (
	EventLobbyUpdate       EventType = "lobby_update"
	EventSessionStarted    EventType = "session_started"
	EventProgressUpdate    EventType = "progress_update"
	EventMatchFound        EventType = "match_found"
	EventDeckExhausted     EventType = "deck_exhausted"
	EventContinueProposed  EventType = "continue_proposed"
	EventContinueVote      EventType = "continue_vote"
	EventContinueCancelled EventType = "continue_cancelled"
	EventContinueResult    EventType = "continue_result"
	EventRequestSent       EventType = "request_sent"
	EventParticipantKicked EventType = "participant_kicked"
	EventParticipantLeft   EventType = "participant_left"
	EventSessionFinished   EventType = "session_finished"
)

// Event carries a detached state snapshot taken at emit time, so consumers
// serialize it without racing later mutations.
type Event struct {
	Type  EventType
	State StateView
	Data  any
}

type MatchFoundData struct {
	Movie domain.Movie
}

type ContinueVoteData struct {
	ParticipantID string
	Accept        bool
}

type ContinueResultData struct {
	Accepted bool
	Extended bool
	Added    int
}

type RequestSentData struct {
	MovieID int64
}

type ParticipantData struct {
	ParticipantID string
	Name          string
}

type Subscriber func(code string, ev Event)
