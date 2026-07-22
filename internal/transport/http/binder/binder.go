package binder

import "github.com/labstack/echo/v4"

// ValidatingBinder chains echo's default binding with validation, so a
// handler that got its request struct can trust it.
type ValidatingBinder struct {
	base echo.DefaultBinder
}

func New() *ValidatingBinder { return &ValidatingBinder{} }

func (b *ValidatingBinder) Bind(i any, c echo.Context) error {
	if err := b.base.Bind(i, c); err != nil {
		return err
	}
	return c.Validate(i)
}
