package ws

import (
	"github.com/Jagerente/matcharr/internal/service/session"
	"github.com/Jagerente/matcharr/internal/transport/http/api"
)

type eventMessage struct {
	Type  string       `json:"type"`
	State api.StateDTO `json:"state"`
	Data  any          `json:"data,omitempty"`
}

type snapshotMessage struct {
	Type  string       `json:"type"`
	State api.StateDTO `json:"state"`
	You   api.YouDTO   `json:"you"`
}

type matchFoundData struct {
	Movie api.MovieDTO `json:"movie"`
}

type continueVoteData struct {
	ParticipantID string `json:"participantId"`
	Accept        bool   `json:"accept"`
}

type continueResultData struct {
	Accepted bool `json:"accepted"`
	Extended bool `json:"extended"`
	Added    int  `json:"added,omitempty"`
}

type requestSentData struct {
	MovieID int64 `json:"movieId"`
}

type participantData struct {
	ParticipantID string `json:"participantId"`
	Name          string `json:"name"`
}

func mapEvent(ev session.Event) eventMessage {
	return eventMessage{
		Type:  string(ev.Type),
		State: api.MapState(ev.State),
		Data:  mapEventData(ev.Data),
	}
}

func mapEventData(data any) any {
	switch d := data.(type) {
	case session.MatchFoundData:
		return matchFoundData{Movie: api.MapMovie(d.Movie)}
	case session.ContinueVoteData:
		return continueVoteData{ParticipantID: d.ParticipantID, Accept: d.Accept}
	case session.ContinueResultData:
		return continueResultData{Accepted: d.Accepted, Extended: d.Extended, Added: d.Added}
	case session.RequestSentData:
		return requestSentData{MovieID: d.MovieID}
	case session.ParticipantData:
		return participantData{ParticipantID: d.ParticipantID, Name: d.Name}
	default:
		return nil
	}
}
