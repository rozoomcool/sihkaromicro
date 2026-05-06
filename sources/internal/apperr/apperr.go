package apperr

import (
	"errors"
	"fmt"
)

// Types of errors for business logic
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrPermissionDenied = errors.New("permission denied")
	ErrLimitExceeded    = errors.New("limit exceeded")
	ErrUnavailable      = errors.New("service unavailable")
)

// Оборачиваем с контекстом
func NotFound(msg string) error {
	return fmt.Errorf("%w: %s", ErrNotFound, msg)
}

func InvalidArgument(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, msg)
}

func LimitExceeded(msg string) error {
	return fmt.Errorf("%w: %s", ErrLimitExceeded, msg)
}

func AlreadyExists(msg string) error {
	return fmt.Errorf("%w: %s", ErrAlreadyExists, msg)
}

func Unavailable(msg string) error {
	return fmt.Errorf("%w: %s", ErrUnavailable, msg)
}
