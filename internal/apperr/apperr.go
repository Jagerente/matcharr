package apperr

import (
	"errors"
	"fmt"
)

type Kind int

const (
	KindInvalid Kind = iota + 1
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooMany
	KindBusy
	KindUpstream
	KindNotImplemented
	KindInternal
)

type Error struct {
	Kind   Kind
	Reason Reason
	msg    string
	cause  error
}

func New(kind Kind, reason Reason, format string, args ...any) *Error {
	return &Error{Kind: kind, Reason: reason, msg: fmt.Sprintf(format, args...)}
}

func Wrap(err error, kind Kind, reason Reason, format string, args ...any) *Error {
	return &Error{Kind: kind, Reason: reason, msg: fmt.Sprintf(format, args...), cause: err}
}

func (e *Error) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return e.msg + ": " + e.cause.Error()
}

func (e *Error) Unwrap() error { return e.cause }

func From(err error) *Error {
	if appErr, ok := errors.AsType[*Error](err); ok {
		return appErr
	}
	return nil
}
