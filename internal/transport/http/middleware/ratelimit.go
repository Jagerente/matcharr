package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
)

type RateLimiter interface {
	Allow(key string) bool
}

// RateLimit requires a non-nil limiter: an unlimited route skips this
// middleware entirely instead of passing one that allows everything.
func RateLimit(limiter RateLimiter, ip ClientIP) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !limiter.Allow(ip.FromRequest(c.Request())) {
				return apperr.New(apperr.KindTooMany, apperr.ReasonRateLimited, "too many requests")
			}
			return next(c)
		}
	}
}
