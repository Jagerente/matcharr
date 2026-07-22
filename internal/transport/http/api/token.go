package api

import "github.com/labstack/echo/v4"

const TokenHeader = "X-Session-Token"

func token(c echo.Context) string {
	return c.Request().Header.Get(TokenHeader)
}
