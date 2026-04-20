package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	pb "github.com/rozoomcool/sihkaromicro/auth/gen/proto/auth"
	"github.com/rozoomcool/sihkaromicro/auth/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler реализует gRPC сервис
type AuthHandler struct {
	pb.UnimplementedAuthServer
	cfg *config.Config
}

func NewAuthGrpc(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) Register(server *grpc.Server) {
	pb.RegisterAuthServer(server, h)
}

func (h *AuthHandler) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	resp, err := http.PostForm(
		h.cfg.ProviderURL,
		url.Values{
			"grant_type": {"password"},
			"client_id":  {h.cfg.ClientID},
			"username":   {in.Username},
			"password":   {in.Password},
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "keycloak unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
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
		h.cfg.ProviderURL,
		url.Values{
			"grant_type":    {"refresh_token"},
			"client_id":     {h.cfg.ClientID},
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
