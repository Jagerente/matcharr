package httperr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantReason apperr.Reason
	}{
		{"invalid", apperr.New(apperr.KindInvalid, apperr.ReasonBadSettings, "x"), http.StatusBadRequest, apperr.ReasonBadSettings},
		{"unauthorized", apperr.New(apperr.KindUnauthorized, apperr.ReasonInvalidToken, "x"), http.StatusUnauthorized, apperr.ReasonInvalidToken},
		{"forbidden", apperr.New(apperr.KindForbidden, apperr.ReasonHostOnly, "x"), http.StatusForbidden, apperr.ReasonHostOnly},
		{"not found", apperr.New(apperr.KindNotFound, apperr.ReasonSessionNotFound, "x"), http.StatusNotFound, apperr.ReasonSessionNotFound},
		{"conflict", apperr.New(apperr.KindConflict, apperr.ReasonBadState, "x"), http.StatusConflict, apperr.ReasonBadState},
		{"too many", apperr.New(apperr.KindTooMany, apperr.ReasonRateLimited, "x"), http.StatusTooManyRequests, apperr.ReasonRateLimited},
		{"busy", apperr.New(apperr.KindBusy, apperr.ReasonServerBusy, "x"), http.StatusServiceUnavailable, apperr.ReasonServerBusy},
		{"upstream", apperr.New(apperr.KindUpstream, apperr.ReasonSourceUnavailable, "x"), http.StatusBadGateway, apperr.ReasonSourceUnavailable},
		{"not implemented", apperr.New(apperr.KindNotImplemented, apperr.ReasonRequestsDisabled, "x"), http.StatusNotImplemented, apperr.ReasonRequestsDisabled},
		{"wrapped cause survives", apperr.Wrap(errors.New("cause"), apperr.KindUpstream, apperr.ReasonRequestFailed, "ctx"), http.StatusBadGateway, apperr.ReasonRequestFailed},
		{"echo 404", echo.NewHTTPError(http.StatusNotFound, "no route"), http.StatusNotFound, apperr.ReasonBadRequest},
		{"echo 500", echo.NewHTTPError(http.StatusInternalServerError, "boom"), http.StatusInternalServerError, apperr.ReasonInternal},
		{"plain error", errors.New("boom"), http.StatusInternalServerError, apperr.ReasonInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, reason, message := Map(tt.err)
			if status != tt.wantStatus || reason != tt.wantReason {
				t.Fatalf("Map() = %d %q, want %d %q", status, reason, tt.wantStatus, tt.wantReason)
			}
			if message == "" {
				t.Fatal("the message must carry the error text for debugging")
			}
		})
	}
}
