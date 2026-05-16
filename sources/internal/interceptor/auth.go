package interceptor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const UserIDKey contextKey = "user_id"

type Claims struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Azp               string `json:"azp"`
}

type AuthInterceptor struct {
	verifier *oidc.IDTokenVerifier
}

func NewAuthInterceptor(ctx context.Context, cfg config.KeycloakCfg) (*AuthInterceptor, error) {
	provider, err := oidc.NewProvider(ctx, fmt.Sprintf("%v/realms/%v", cfg.Url, cfg.Realm))
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.Client.ID,
	})

	return &AuthInterceptor{verifier: verifier}, nil
}

func (a *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, err := a.authorize(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (a *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := a.authorize(ss.Context())
		if err != nil {
			return err
		}
		return handler(srv, &wrappedStream{ServerStream: ss, ctx: ctx})
	}
}

// wrappedStream replaces the context on an existing ServerStream.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func (a *AuthInterceptor) authorize(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization token")
	}

	rawToken := strings.TrimPrefix(values[0], "Bearer ")

	token, err := a.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	var claims Claims
	if err := token.Claims(&claims); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid claims")
	}

	ctx = context.WithValue(ctx, UserIDKey, claims.Sub)
	return ctx, nil
}

func MustUserIDFromCtx(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		panic(status.Error(codes.Unauthenticated, "user_id not found in context"))
	}
	return userID
}

func UserIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok || userID == "" {
		return "", errors.New("user id key not present")
	}
	return userID, nil
}
