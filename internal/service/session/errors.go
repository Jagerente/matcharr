package session

import (
	"errors"

	"github.com/Jagerente/matcharr/internal/apperr"
)

var (
	errNotFound     = apperr.New(apperr.KindNotFound, apperr.ReasonSessionNotFound, "session not found")
	errTokenInvalid = apperr.New(apperr.KindUnauthorized, apperr.ReasonInvalidToken, "invalid participant token")
	errHostOnly     = apperr.New(apperr.KindForbidden, apperr.ReasonHostOnly, "only the host can do that")
	errBadState     = apperr.New(apperr.KindConflict, apperr.ReasonBadState, "action not available in the current session state")

	// errNoop aborts a transaction that must neither mutate further nor
	// emit, without surfacing an error to the caller.
	errNoop = errors.New("no effect")
)

func invalid(reason apperr.Reason, format string, args ...any) error {
	return apperr.New(apperr.KindInvalid, reason, format, args...)
}
