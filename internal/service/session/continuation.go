package session

import (
	"context"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

func (s *Service) ProposeContinue(code, token string, limit int) error {
	return s.exec(code, require{token: token, host: true, statuses: inActive}, func(tx *Tx) error {
		sess := tx.Session()
		if !sess.AllDone() || sess.TargetReached() || sess.SourceDone {
			return errBadState
		}
		if sess.Settings.MatchMode == domain.MatchModeAll {
			return errBadState
		}
		if limit == 0 {
			limit = sess.Settings.DeckLimit
		}
		low, high := s.limits.deckRange()
		if limit < low || limit > high {
			return invalid(apperr.ReasonBadDeckLimit, "extension out of the %d..%d range", low, high)
		}

		sess.StartContinueVote(tx.Actor(), limit)
		tx.Emit(EventContinueProposed, nil)
		return nil
	})
}

func (s *Service) CancelContinue(code, token string) error {
	return s.exec(code, require{token: token, host: true, statuses: inVoting, idle: true}, func(tx *Tx) error {
		tx.Session().CancelContinueVote()
		tx.Emit(EventContinueCancelled, nil)
		return nil
	})
}

func (s *Service) VoteContinue(ctx context.Context, code, token string, accept bool) error {
	var allAccepted bool
	err := s.exec(code, require{token: token, statuses: inVoting, idle: true}, func(tx *Tx) error {
		sess := tx.Session()
		sess.RecordContinueVote(tx.Actor(), accept)
		tx.Emit(EventContinueVote, ContinueVoteData{ParticipantID: tx.Actor().ID, Accept: accept})

		if !accept {
			tx.Emit(EventContinueResult, ContinueResultData{Accepted: false})
			s.finish(tx)
			return nil
		}
		allAccepted = sess.AllAccepted()
		return nil
	})
	if err != nil {
		return err
	}
	if allAccepted {
		s.completeContinue(ctx, code)
	}
	return nil
}

// completeContinue extends the deck after a unanimous "continue". It runs
// from the last accepting vote, or detached after removing the last
// non-voter.
func (s *Service) completeContinue(ctx context.Context, code string) {
	_ = s.withDeckFetch(ctx, code,
		require{statuses: inVoting},
		func(tx *Tx) (deck.Request, error) {
			sess := tx.Session()
			if !sess.AllAccepted() {
				return deck.Request{}, errNoop
			}
			return deck.Request{
				PresetID:  sess.Settings.PresetID,
				Filters:   sess.Settings.Filters,
				StartPage: sess.SourcePage,
				Want:      sess.Continue.ProposedLimit,
				Exclude:   sess.DeckIDs(),
			}, nil
		},
		func(tx *Tx, result deck.Result, buildErr error) error {
			sess := tx.Session()
			if sess.Status != domain.StatusVoting {
				// The session was finished while the deck was fetching.
				return errNoop
			}
			if buildErr != nil || len(result.Movies) == 0 {
				if buildErr != nil {
					s.logger.Error("continuation: building the deck", "code", code, "err", buildErr)
				}
				sess.ExhaustContinue()
				tx.Emit(EventContinueResult, ContinueResultData{Accepted: true})
				s.finish(tx)
				return nil
			}

			sess.ExtendDeck(result.Movies, result.NextPage, result.Exhausted)
			sess.Touch(tx.Now())
			tx.Emit(EventContinueResult, ContinueResultData{Accepted: true, Extended: true, Added: len(result.Movies)})
			return nil
		})
}
