package session

import (
	"maps"
	"slices"

	"github.com/Jagerente/matcharr/internal/domain"
)

type ParticipantView struct {
	ID       string
	Name     string
	Avatar   string
	IsHost   bool
	Online   bool
	Progress int
	Done     bool
}

type ContinueView struct {
	ProposedLimit int
	Votes         map[string]bool
}

// StateView never carries per-movie votes: they stay hidden until a match.
type StateView struct {
	Code            string
	Status          domain.Status
	Round           int
	Settings        domain.Settings
	Participants    []ParticipantView
	Departed        []domain.Departed
	DeckSize        int
	SourceExhausted bool
	AllDone         bool
	NearMissCount   int
	Matches         []domain.Match
	Requested       []int64
	Continue        *ContinueView
	Results         *domain.Results
}

type You struct {
	ID     string
	IsHost bool
	Cursor int
}

type PersonalView struct {
	State StateView
	You   You
}

type Credentials struct {
	Code  string
	Token string
	You   You
}

func buildView(s *domain.Session) StateView {
	participants := make([]ParticipantView, 0, len(s.Participants))
	for _, p := range s.Participants {
		participants = append(participants, ParticipantView{
			ID:       p.ID,
			Name:     p.Name,
			Avatar:   p.Avatar,
			IsHost:   p.IsHost,
			Online:   p.Online,
			Progress: p.Cursor,
			Done:     s.Status != domain.StatusLobby && p.Cursor >= len(s.Deck),
		})
	}

	var voting *ContinueView
	if s.Continue != nil {
		voting = &ContinueView{
			ProposedLimit: s.Continue.ProposedLimit,
			Votes:         maps.Clone(s.Continue.Votes),
		}
	}

	return StateView{
		Code:            s.Code,
		Status:          s.Status,
		Round:           s.Round,
		Settings:        s.Settings,
		Participants:    participants,
		Departed:        slices.Clone(s.Departed),
		DeckSize:        len(s.Deck),
		SourceExhausted: s.SourceDone,
		AllDone:         s.AllDone(),
		NearMissCount:   s.NearMissCount(),
		Matches:         slices.Clone(s.Matches),
		Requested:       slices.Clone(s.Requested),
		Continue:        voting,
		Results:         s.Results,
	}
}

func you(p *domain.Participant) You {
	return You{ID: p.ID, IsHost: p.IsHost, Cursor: p.Cursor}
}
