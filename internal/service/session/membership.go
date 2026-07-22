package session

import (
	"context"
	"time"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type CreateInput struct {
	Name     string
	Avatar   string
	DeviceID string
	Settings domain.Settings
}

type JoinInput struct {
	Name     string
	Avatar   string
	DeviceID string
}

func (s *Service) Create(in CreateInput) (Credentials, error) {
	settings := in.Settings
	if err := s.normalizeSettings(&settings); err != nil {
		return Credentials{}, err
	}
	who, err := validateIdentity(in.Name, in.Avatar, in.DeviceID)
	if err != nil {
		return Credentials{}, err
	}

	host := &domain.Participant{
		ID:       newID(),
		Name:     who.name,
		Avatar:   who.avatar,
		IsHost:   true,
		Token:    newToken(),
		DeviceID: who.deviceID,
	}
	code, err := s.store.Create(func(code string, now time.Time) *domain.Session {
		return domain.NewSession(code, settings, host, now)
	})
	if err != nil {
		return Credentials{}, err
	}
	return Credentials{Code: code, Token: host.Token, You: you(host)}, nil
}

func (s *Service) Join(code string, in JoinInput) (Credentials, error) {
	who, err := validateIdentity(in.Name, in.Avatar, in.DeviceID)
	if err != nil {
		return Credentials{}, err
	}

	var creds Credentials
	err = s.exec(code, require{}, func(tx *Tx) error {
		sess := tx.Session()
		switch sess.Status {
		case domain.StatusLobby:
		case domain.StatusFinished:
			return invalid(apperr.ReasonSessionFinished, "session finished")
		default:
			return invalid(apperr.ReasonSessionStarted, "session already started")
		}

		// A returning device reclaims its own participant (the token
		// rotates, name and avatar refresh), joining twice from one browser
		// must never stack duplicates in the lobby.
		if existing := sess.ParticipantByDevice(who.deviceID); existing != nil {
			existing.Name = who.name
			existing.Avatar = who.avatar
			existing.Token = newToken()
			sess.Touch(tx.Now())
			tx.Emit(EventLobbyUpdate, nil)
			creds = Credentials{Code: code, Token: existing.Token, You: you(existing)}
			return nil
		}

		if len(sess.Participants) >= s.limits.MaxParticipants {
			return invalid(apperr.ReasonSessionFull, "session full at %d participants", s.limits.MaxParticipants)
		}
		joined := &domain.Participant{
			ID:       newID(),
			Name:     who.name,
			Avatar:   who.avatar,
			Token:    newToken(),
			DeviceID: who.deviceID,
		}
		sess.AddParticipant(joined)
		sess.Touch(tx.Now())
		tx.Emit(EventLobbyUpdate, nil)
		creds = Credentials{Code: code, Token: joined.Token, You: you(joined)}
		return nil
	})
	return creds, err
}

func (s *Service) Kick(code, token, participantID string) error {
	return s.remove(code, require{token: token, host: true}, EventParticipantKicked,
		func(tx *Tx) (*domain.Participant, error) {
			if tx.Actor().ID == participantID {
				return nil, invalid(apperr.ReasonSelfKick, "the host cannot kick themselves")
			}
			target := tx.Session().ParticipantByID(participantID)
			if target == nil {
				return nil, invalid(apperr.ReasonNoParticipant, "participant not found")
			}
			return target, nil
		})
}

func (s *Service) Leave(code, token string) error {
	return s.remove(code, require{token: token, statuses: beforeFinish}, EventParticipantLeft,
		func(tx *Tx) (*domain.Participant, error) {
			if tx.Actor().IsHost {
				return nil, invalid(apperr.ReasonHostCannotLeave, "the host cannot leave")
			}
			return tx.Actor(), nil
		})
}

// remove completes a resolved continuation vote detached: the deck fetch
// must not hold the caller's request.
func (s *Service) remove(code string, req require, evType EventType, pick func(tx *Tx) (*domain.Participant, error)) error {
	var continueReady bool
	err := s.exec(code, req, func(tx *Tx) error {
		target, err := pick(tx)
		if err != nil {
			return err
		}

		sess := tx.Session()
		wasVoting := sess.Status == domain.StatusVoting
		sess.RemoveParticipant(target)
		sess.SyncPresence(tx.Now())
		tx.Emit(evType, ParticipantData{ParticipantID: target.ID, Name: target.Name})

		switch {
		case sess.Status == domain.StatusLobby:
			tx.Emit(EventLobbyUpdate, nil)
		case len(sess.Participants) < 2:
			s.finish(tx)
		default:
			// Without that participant the threshold may have dropped,
			// movies could have ripened into matches.
			if wasVoting {
				continueReady = sess.AllAccepted()
			}
			s.settle(tx)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if continueReady {
		go s.completeContinueDetached(code)
	}
	return nil
}

func (s *Service) completeContinueDetached(code string) {
	ctx, cancel := context.WithTimeout(context.Background(), asyncFetchTimeout)
	defer cancel()
	s.completeContinue(ctx, code)
}

func (s *Service) SetPresence(code, participantID string, online bool) {
	_ = s.exec(code, require{}, func(tx *Tx) error {
		sess := tx.Session()
		p := sess.ParticipantByID(participantID)
		if p == nil || p.Online == online {
			return errNoop
		}
		p.Online = online
		// Arriving is what refreshes a session, leaving is what starts its
		// death clock, so presence must never pass through LastSeen.
		sess.SyncPresence(tx.Now())
		tx.Emit(EventLobbyUpdate, nil)
		return nil
	})
}
