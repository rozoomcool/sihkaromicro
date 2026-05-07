package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
	"github.com/rozoomcool/sihkaromicro/auth/pkg/logger/sl"
	pb "github.com/rozoomcool/sihkaromicro/proto/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler implements auth grpc service
type AuthHandler struct {
	pb.UnimplementedAuthServer
	cfg *config.Config
	log *slog.Logger
}

func NewAuthGrpc(cfg *config.Config, log *slog.Logger) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) Register(server *grpc.Server) {
	pb.RegisterAuthServer(server, h)
}

func (h *AuthHandler) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("%v/realms/%v/protocol/openid-connect/token", h.cfg.Keycloak.Url, h.cfg.Keycloak.Realm),
		url.Values{
			"grant_type": {"password"},
			"client_id":  {h.cfg.Keycloak.Client.ID},
			"username":   {in.Username},
			"password":   {in.Password},
		},
	)
	if err != nil {
		h.log.Error("Keycloak request error", sl.Err(err))
		return nil, status.Error(codes.Internal, "keycloak unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		h.log.Error("Invalid creadentials", slog.String("Username", in.Username))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}

func (h *AuthHandler) Refresh(ctx context.Context, in *pb.RefreshRequest) (*pb.LoginResponse, error) {
	resp, err := http.PostForm(
		fmt.Sprintf("%v/realms/%v", h.cfg.Keycloak.Url, h.cfg.Keycloak.Realm),
		url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {h.cfg.Keycloak.Client.ID},
			"refresh_token": {in.RefreshToken},
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "keycloak unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int32  `json:"expires_in"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return &pb.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	}, nil
}
