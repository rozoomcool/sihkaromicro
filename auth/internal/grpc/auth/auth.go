package authgrpc

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/auth/gen/proto/sso"
	pb "github.com/rozoomcool/sihkaromicro/auth/gen/proto/sso"
	"github.com/rozoomcool/sihkaromicro/auth/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Auth реализует gRPC сервис
type Auth struct {
	pb.UnimplementedAuthServer
	service service.AuthService
}

func NewAuthGrpc(service service.AuthService) *Auth {
	return &Auth{
		service: service,
	}
}

func Register(gRPCServer *grpc.Server, authSrv service.AuthService) {
	sso.RegisterAuthServer(gRPCServer, &Auth{service: authSrv})
}

func (h *Auth) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	id, err := h.service.Register(ctx, in.Username, in.Password)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResponse{UserId: id}, nil
}
func (h *Auth) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	auth, err := h.service.Login(ctx, in.Username, in.Password)
	if err != nil {
		return nil, err
	}
	return &pb.LoginResponse{
		Token:        auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		UserId:       auth.UserID,
	}, err
}
func (h *Auth) IsAdmin(context.Context, *pb.IsAdminRequest) (*pb.IsAdminResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method IsAdmin not implemented")
}

// func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
// 	// Валидация входных данных
// 	if err := validateRegisterRequest(req); err != nil {
// 		return nil, err
// 	}

// 	input := service.RegisterInput{
// 		Email:    req.Email,
// 		Password: req.Password,
// 		Name:     req.Name,
// 	}

// 	resp, err := h.service.Register(ctx, input)
// 	if err != nil {
// 		return nil, err // Ошибки уже в формате gRPC status
// 	}

// 	return &pb.AuthResponse{
// 		AccessToken:  resp.AccessToken,
// 		RefreshToken: resp.RefreshToken,
// 		UserId:       resp.UserID,
// 	}, nil
// }

// func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
// 	if err := validateLoginRequest(req); err != nil {
// 		return nil, err
// 	}

// 	resp, err := h.service.Login(ctx, req.Email, req.Password)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.AuthResponse{
// 		AccessToken:  resp.AccessToken,
// 		RefreshToken: resp.RefreshToken,
// 		UserId:       resp.UserID,
// 	}, nil
// }

// func (h *AuthHandler) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.AuthResponse, error) {
// 	if req.RefreshToken == "" {
// 		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
// 	}

// 	resp, err := h.service.Refresh(ctx, req.RefreshToken)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.AuthResponse{
// 		AccessToken:  resp.AccessToken,
// 		RefreshToken: resp.RefreshToken,
// 		UserId:       resp.UserID,
// 	}, nil
// }

// func (h *AuthHandler) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
// 	if req.AccessToken == "" {
// 		return &pb.ValidateResponse{Valid: false}, nil
// 	}

// 	valid, userID, email, err := h.service.Validate(ctx, req.AccessToken)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &pb.ValidateResponse{
// 		Valid:  valid,
// 		UserId: userID,
// 		Email:  email,
// 	}, nil
// }

// func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
// 	if req.RefreshToken == "" {
// 		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
// 	}

// 	if err := h.service.Logout(ctx, req.RefreshToken); err != nil {
// 		return nil, err
// 	}

// 	return &pb.LogoutResponse{Success: true}, nil
// }

// // --- Валидация запросов ---

// func validateRegisterRequest(req *pb.RegisterRequest) error {
// 	if req.Email == "" {
// 		return status.Error(codes.InvalidArgument, "email is required")
// 	}
// 	if req.Password == "" {
// 		return status.Error(codes.InvalidArgument, "password is required")
// 	}
// 	if len(req.Password) < 8 {
// 		return status.Error(codes.InvalidArgument, "password must be at least 8 characters")
// 	}
// 	return nil
// }

// func validateLoginRequest(req *pb.LoginRequest) error {
// 	if req.Email == "" {
// 		return status.Error(codes.InvalidArgument, "email is required")
// 	}
// 	if req.Password == "" {
// 		return status.Error(codes.InvalidArgument, "password is required")
// 	}
// 	return nil
// }
