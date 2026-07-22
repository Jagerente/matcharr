package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// BodyLimit caps every request body. Each endpoint takes a handful of
// short fields, so anything larger is someone probing how much memory a
// stranger can make the server allocate.
func BodyLimit(maxBytes int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if maxBytes > 0 && c.Request().Body != nil {
				c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxBytes)
			}
			return next(c)
		}
	}
}
