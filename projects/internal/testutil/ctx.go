package testutil

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/projects/internal/interceptor"
)

func CtxWithUserID(userID string) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, userID)
}
