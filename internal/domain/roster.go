package domain

type Participant struct {
	ID     string
	Name   string
	Avatar string
	IsHost bool
	// Token is the participant's authentication secret, never shared with others.
	Token string
	// DeviceID is the browser's stable anonymous identity. A returning
	// device reclaims its participant instead of stacking a duplicate.
	DeviceID string
	Cursor   int
	Online   bool
}

type Departed struct {
	ID       string
	Name     string
	Avatar   string
	Progress int
}

func (s *Session) AddParticipant(p *Participant) {
	s.Participants = append(s.Participants, p)
}

func (s *Session) RemoveParticipant(target *Participant) {
	kept := s.Participants[:0]
	for _, p := range s.Participants {
		if p.ID != target.ID {
			kept = append(kept, p)
		}
	}
	s.Participants = kept
	for _, votes := range s.Votes {
		delete(votes, target.ID)
	}
	if s.Continue != nil {
		delete(s.Continue.Votes, target.ID)
	}
	// Lobby departures leave no trace: people wander in and out before the
	// start, and ghosts there would read as duplicates.
	if s.Status != StatusLobby {
		s.Departed = append(s.Departed, Departed{
			ID:       target.ID,
			Name:     target.Name,
			Avatar:   target.Avatar,
			Progress: target.Cursor,
		})
	}
}

func (s *Session) ParticipantByToken(token string) *Participant {
	for _, p := range s.Participants {
		if p.Token == token {
			return p
		}
	}
	return nil
}

func (s *Session) ParticipantByID(id string) *Participant {
	for _, p := range s.Participants {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (s *Session) ParticipantByDevice(deviceID string) *Participant {
	if deviceID == "" {
		return nil
	}
	for _, p := range s.Participants {
		if p.DeviceID == deviceID {
			return p
		}
	}
	return nil
}

func (s *Session) AnyOnline() bool {
	for _, p := range s.Participants {
		if p.Online {
			return true
		}
	}
	return false
}
