package api

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/domain"
)

type RequestService interface {
	Profiles(ctx context.Context, code, token string) (domain.QualityProfiles, error)
	Submit(ctx context.Context, code, token string, movieID int64, profileID int) error
}

type RequestHandler struct {
	requests RequestService
}

func NewRequestHandler(requests RequestService) *RequestHandler {
	return &RequestHandler{requests: requests}
}

func (h *RequestHandler) Profiles(c echo.Context) error {
	profiles, err := h.requests.Profiles(c.Request().Context(), c.Param("code"), token(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapProfiles(profiles))
}

type submitRequest struct {
	MovieID   int64 `json:"movieId" validate:"gt=0"`
	ProfileID int   `json:"profileId" validate:"gte=0"`
}

func (h *RequestHandler) Submit(c echo.Context) error {
	var req submitRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.requests.Submit(c.Request().Context(), c.Param("code"), token(c), req.MovieID, req.ProfileID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
