package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/domain"
)

type CatalogService interface {
	Presets() []domain.Preset
	Genres(ctx context.Context) ([]domain.Genre, error)
	MovieDetails(ctx context.Context, code, token string, movieID int64) (domain.Movie, error)
}

type CatalogHandler struct {
	catalog         CatalogService
	requestsEnabled bool
}

func NewCatalogHandler(catalog CatalogService, requestsEnabled bool) *CatalogHandler {
	return &CatalogHandler{catalog: catalog, requestsEnabled: requestsEnabled}
}

func (h *CatalogHandler) Config(c echo.Context) error {
	return c.JSON(http.StatusOK, ConfigDTO{RequestsEnabled: h.requestsEnabled})
}

func (h *CatalogHandler) Presets(c echo.Context) error {
	return c.JSON(http.StatusOK, mapPresets(h.catalog.Presets()))
}

func (h *CatalogHandler) Genres(c echo.Context) error {
	genres, err := h.catalog.Genres(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapGenres(genres))
}

func (h *CatalogHandler) MovieDetails(c echo.Context) error {
	movieID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || movieID <= 0 {
		return apperr.New(apperr.KindInvalid, apperr.ReasonBadRequest, "invalid movie id")
	}
	movie, err := h.catalog.MovieDetails(c.Request().Context(), c.Param("code"), token(c), movieID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, MapMovie(movie))
}
