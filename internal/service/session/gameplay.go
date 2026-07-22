package session

import (
	"context"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

func (s *Service) Start(ctx context.Context, code, token string) error {
	return s.withDeckFetch(ctx, code,
		require{token: token, host: true, statuses: inLobby},
		func(tx *Tx) (deck.Request, error) {
			sess := tx.Session()
			if len(sess.Participants) < 2 {
				return deck.Request{}, invalid(apperr.ReasonNotEnoughPlayers, "at least two participants are needed")
			}
			return deck.Request{
				PresetID:  sess.Settings.PresetID,
				Filters:   sess.Settings.Filters,
				StartPage: sess.SourcePage,
				Want:      sess.Settings.DeckLimit,
			}, nil
		},
		func(tx *Tx, result deck.Result, buildErr error) error {
			sess := tx.Session()
			if sess.Status != domain.StatusLobby {
				// The host cancelled the session while the deck was fetching.
				return errBadState
			}
			if buildErr != nil {
				return apperr.Wrap(buildErr, apperr.KindUpstream, apperr.ReasonSourceUnavailable, "building the deck")
			}
			if len(result.Movies) == 0 {
				return invalid(apperr.ReasonEmptySource, "the source returned no movies")
			}
			sess.BeginRound(result.Movies, result.NextPage, result.Exhausted)
			tx.Emit(EventSessionStarted, nil)
			return nil
		})
}

func (s *Service) Swipe(code, token string, movieID int64, verdict domain.Verdict) error {
	if !verdict.Valid() {
		return invalid(apperr.ReasonBadRequest, "unknown verdict %q", verdict)
	}

	return s.exec(code, require{token: token, statuses: inActive}, func(tx *Tx) error {
		sess := tx.Session()
		current, ok := sess.CurrentMovie(tx.Actor())
		if !ok {
			return invalid(apperr.ReasonDeckFinished, "the deck is finished")
		}
		// Only the movie at the current cursor can be swiped, protection
		// against a client drifting out of sync with the server.
		if current.ID != movieID {
			return invalid(apperr.ReasonOutOfOrder, "expected movie %d, got %d", current.ID, movieID)
		}

		sess.RecordSwipe(tx.Actor(), verdict)
		tx.Emit(EventProgressUpdate, nil)
		s.settle(tx)
		return nil
	})
}

func (s *Service) Undo(code, token string) error {
	return s.exec(code, require{token: token, statuses: inActive}, func(tx *Tx) error {
		sess := tx.Session()
		movie, ok := sess.LastSwiped(tx.Actor())
		if !ok {
			return invalid(apperr.ReasonNothingToUndo, "nothing to undo")
		}
		if sess.IsMatched(movie.ID) {
			return invalid(apperr.ReasonAlreadyMatched, "movie has already matched")
		}
		sess.RevokeSwipe(tx.Actor())
		tx.Emit(EventProgressUpdate, nil)
		return nil
	})
}

func (s *Service) Finish(code, token string) error {
	return s.exec(code, require{token: token, host: true, statuses: beforeFinish}, func(tx *Tx) error {
		s.finish(tx)
		return nil
	})
}

// settle translates a domain evaluation into events, keeping the emission
// order the clients rely on: matches first, then the terminal event.
func (s *Service) settle(tx *Tx) {
	outcome := tx.Session().Evaluate(tx.Now())
	for _, match := range outcome.Matches {
		tx.Emit(EventMatchFound, MatchFoundData{Movie: match.Movie})
	}
	if outcome.Finished {
		tx.Emit(EventSessionFinished, nil)
		return
	}
	if outcome.DeckExhausted {
		tx.Emit(EventDeckExhausted, nil)
	}
}

func (s *Service) finish(tx *Tx) {
	if tx.Session().Finish(tx.Now()) {
		tx.Emit(EventSessionFinished, nil)
	}
}
