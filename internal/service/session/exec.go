package session

import (
	"context"
	"slices"

	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/deck"
)

// require declares what a command needs before its body runs: a valid
// token, the host role, an allowed set of statuses, an idle session with
// no deck fetch in flight. Taken together, the declarations on the
// commands are the session's transition rules.
type require struct {
	token    string
	host     bool
	statuses []domain.Status
	idle     bool
}

var (
	inLobby      = []domain.Status{domain.StatusLobby}
	inActive     = []domain.Status{domain.StatusActive}
	inVoting     = []domain.Status{domain.StatusVoting}
	afterLobby   = []domain.Status{domain.StatusActive, domain.StatusVoting, domain.StatusFinished}
	beforeFinish = []domain.Status{domain.StatusLobby, domain.StatusActive, domain.StatusVoting}
)

func (s *Service) exec(code string, req require, fn func(tx *Tx) error) error {
	return s.store.Update(code, func(tx *Tx) error {
		if req.token != "" {
			p, err := tx.Authenticate(req.token)
			if err != nil {
				return err
			}
			if req.host && !p.IsHost {
				return errHostOnly
			}
		}
		if len(req.statuses) > 0 && !slices.Contains(req.statuses, tx.Session().Status) {
			return errBadState
		}
		if req.idle && tx.Fetching() {
			return errBadState
		}
		return fn(tx)
	})
}

// withDeckFetch is the only pattern that leaves the lock mid-command:
// reserve shapes the request and marks the session fetching, the network
// build runs unlocked, then commit applies the outcome. The mark clears
// before commit runs, and commit sees the session as it is after the
// fetch, so it must recheck the status itself.
func (s *Service) withDeckFetch(
	ctx context.Context,
	code string,
	req require,
	reserve func(tx *Tx) (deck.Request, error),
	commit func(tx *Tx, result deck.Result, buildErr error) error,
) error {
	req.idle = true
	var request deck.Request
	err := s.exec(code, req, func(tx *Tx) error {
		r, err := reserve(tx)
		if err != nil {
			return err
		}
		request = r
		tx.MarkFetching(true)
		return nil
	})
	if err != nil {
		return err
	}

	result, buildErr := s.decks.Build(ctx, request)

	return s.exec(code, require{}, func(tx *Tx) error {
		tx.MarkFetching(false)
		return commit(tx, result, buildErr)
	})
}
