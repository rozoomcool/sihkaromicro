package apperr

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToGRPC — calls in handlers only
func ToGRPC(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, "already exists")
	case errors.Is(err, ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, "invalid argument")
	case errors.Is(err, ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, ErrLimitExceeded):
		return status.Error(codes.FailedPrecondition, "limit exceeded")
	case errors.Is(err, ErrUnavailable):
		return status.Error(codes.Unavailable, "service unavailable")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
