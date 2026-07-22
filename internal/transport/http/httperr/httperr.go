package httperr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
)

// Envelope is the error wire format. The message is for logs and debugging
// only: clients pick their wording from the reason.
type Envelope struct {
	Reason  apperr.Reason `json:"reason"`
	Message string        `json:"message"`
}

func Map(err error) (status int, reason apperr.Reason, message string) {
	if appErr := apperr.From(err); appErr != nil {
		return statusOf(appErr.Kind), appErr.Reason, appErr.Error()
	}
	if httpErr, ok := errors.AsType[*echo.HTTPError](err); ok {
		return httpErr.Code, reasonForStatus(httpErr.Code), fmt.Sprint(httpErr.Message)
	}
	return http.StatusInternalServerError, apperr.ReasonInternal, err.Error()
}

func Handle(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status, reason, message := Map(err)
	if status == http.StatusTooManyRequests {
		c.Response().Header().Set("Retry-After", "5")
	}
	if writeErr := c.JSON(status, Envelope{Reason: reason, Message: message}); writeErr != nil {
		c.Logger().Error(writeErr)
	}
}

func statusOf(kind apperr.Kind) int {
	switch kind {
	case apperr.KindInvalid:
		return http.StatusBadRequest
	case apperr.KindUnauthorized:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindTooMany:
		return http.StatusTooManyRequests
	case apperr.KindBusy:
		return http.StatusServiceUnavailable
	case apperr.KindUpstream:
		return http.StatusBadGateway
	case apperr.KindNotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func reasonForStatus(status int) apperr.Reason {
	if status >= http.StatusInternalServerError {
		return apperr.ReasonInternal
	}
	return apperr.ReasonBadRequest
}
