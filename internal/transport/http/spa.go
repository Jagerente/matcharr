package http

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func spaHandler(static fs.FS) echo.HandlerFunc {
	files := http.FileServerFS(static)
	return func(c echo.Context) error {
		path := strings.TrimPrefix(c.Param("*"), "/")
		if path != "" {
			if info, err := fs.Stat(static, path); err == nil && !info.IsDir() {
				// The embedded FS has no timestamps, so browsers cache
				// heuristically without explicit headers; hashed assets
				// are safe to cache forever.
				if strings.HasPrefix(path, "assets/") {
					c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					c.Response().Header().Set("Cache-Control", "no-cache")
				}
				req := c.Request().Clone(c.Request().Context())
				req.URL.Path = "/" + path
				files.ServeHTTP(c.Response(), req)
				return nil
			}
		}

		index, err := fs.ReadFile(static, "index.html")
		if err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "frontend not built: run `npm run build` in web/")
		}
		// A stale index.html keeps phones on an old bundle after deploys.
		c.Response().Header().Set("Cache-Control", "no-cache")
		return c.HTMLBlob(http.StatusOK, index)
	}
}
