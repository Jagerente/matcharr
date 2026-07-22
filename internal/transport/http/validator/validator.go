package validator

import (
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/Jagerente/matcharr/internal/apperr"
)

type EchoValidator struct {
	validate *validator.Validate
}

func New() *EchoValidator {
	return &EchoValidator{validate: validator.New(validator.WithRequiredStructEnabled())}
}

func (v *EchoValidator) Validate(i any) error {
	value := reflect.ValueOf(i)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil
	}
	if err := v.validate.Struct(i); err != nil {
		return apperr.New(apperr.KindInvalid, apperr.ReasonBadRequest, "%s", err)
	}
	return nil
}
