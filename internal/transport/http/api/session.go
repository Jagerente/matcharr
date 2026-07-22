package api

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/domain"
	"github.com/Jagerente/matcharr/internal/service/session"
)

type SessionService interface {
	Create(in session.CreateInput) (session.Credentials, error)
	Join(code string, in session.JoinInput) (session.Credentials, error)
	State(code, token string) (session.PersonalView, error)
	Deck(code, token string) ([]domain.Movie, error)
	Start(ctx context.Context, code, token string) error
	Swipe(code, token string, movieID int64, verdict domain.Verdict) error
	Undo(code, token string) error
	ProposeContinue(code, token string, limit int) error
	VoteContinue(ctx context.Context, code, token string, accept bool) error
	CancelContinue(code, token string) error
	Finish(code, token string) error
	Kick(code, token, participantID string) error
	Leave(code, token string) error
}

type SessionHandler struct {
	sessions SessionService
}

func NewSessionHandler(sessions SessionService) *SessionHandler {
	return &SessionHandler{sessions: sessions}
}

type createRequest struct {
	Name     string      `json:"name"`
	Avatar   string      `json:"avatar"`
	DeviceID string      `json:"deviceId"`
	Settings SettingsDTO `json:"settings"`
}

func (h *SessionHandler) Create(c echo.Context) error {
	var req createRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	creds, err := h.sessions.Create(session.CreateInput{
		Name:     req.Name,
		Avatar:   req.Avatar,
		DeviceID: req.DeviceID,
		Settings: settingsFromDTO(req.Settings),
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, mapCredentials(creds))
}

type joinRequest struct {
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	DeviceID string `json:"deviceId"`
}

func (h *SessionHandler) Join(c echo.Context) error {
	var req joinRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	creds, err := h.sessions.Join(c.Param("code"), session.JoinInput{
		Name:     req.Name,
		Avatar:   req.Avatar,
		DeviceID: req.DeviceID,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapCredentials(creds))
}

func (h *SessionHandler) State(c echo.Context) error {
	view, err := h.sessions.State(c.Param("code"), token(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PersonalStateDTO{State: MapState(view.State), You: MapYou(view.You)})
}

func (h *SessionHandler) Deck(c echo.Context) error {
	movies, err := h.sessions.Deck(c.Param("code"), token(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, DeckDTO{Movies: mapMovies(movies)})
}

func (h *SessionHandler) Start(c echo.Context) error {
	if err := h.sessions.Start(c.Request().Context(), c.Param("code"), token(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type swipeRequest struct {
	MovieID int64  `json:"movieId" validate:"gt=0"`
	Verdict string `json:"verdict" validate:"oneof=like nope"`
}

func (h *SessionHandler) Swipe(c echo.Context) error {
	var req swipeRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.sessions.Swipe(c.Param("code"), token(c), req.MovieID, domain.Verdict(req.Verdict)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionHandler) Undo(c echo.Context) error {
	if err := h.sessions.Undo(c.Param("code"), token(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type proposeContinueRequest struct {
	Limit int `json:"limit" validate:"gte=0"`
}

func (h *SessionHandler) ProposeContinue(c echo.Context) error {
	var req proposeContinueRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.sessions.ProposeContinue(c.Param("code"), token(c), req.Limit); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

type voteContinueRequest struct {
	Accept bool `json:"accept"`
}

func (h *SessionHandler) VoteContinue(c echo.Context) error {
	var req voteContinueRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := h.sessions.VoteContinue(c.Request().Context(), c.Param("code"), token(c), req.Accept); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionHandler) CancelContinue(c echo.Context) error {
	if err := h.sessions.CancelContinue(c.Param("code"), token(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionHandler) Finish(c echo.Context) error {
	if err := h.sessions.Finish(c.Param("code"), token(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionHandler) Kick(c echo.Context) error {
	if err := h.sessions.Kick(c.Param("code"), token(c), c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *SessionHandler) Leave(c echo.Context) error {
	if err := h.sessions.Leave(c.Param("code"), token(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
