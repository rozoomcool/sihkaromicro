package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/rozoomcool/sihkaromicro/projects/internal/config"
	"github.com/rozoomcool/sihkaromicro/projects/internal/handler"
	"github.com/rozoomcool/sihkaromicro/projects/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/projects/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	cfg        *config.Config
}

// New creates new gRPC server app.
func New(
	log *slog.Logger,
	projectSrv service.ProjectService,
	cfg *config.Config,
) *App {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			//logging.StartCall, logging.FinishCall,
			logging.PayloadReceived, logging.PayloadSent,
		),
	}

	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p any) error {
			log.Info("Recovered from panic", slog.Any("panic", p))
			if st, ok := p.(interface{ GRPCStatus() *status.Status }); ok {
				return st.GRPCStatus().Err()
			}

			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	authInterceptor, err := interceptor.NewAuthInterceptor(context.Background(), cfg.Auth)
	if err != nil {
		panic(fmt.Sprintf("failed to create auth interceptor: %v", err))
	}

	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...),
		authInterceptor.Unary(),
	))

	projectHandler := handler.NewProjectGRPCHandler(projectSrv, log)
	projectHandler.Register(gRPCServer)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		cfg:        cfg,
	}
}

// InterceptorLogger adapts slog logger to interceptor logger.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

// MustRun runs gRPC server and panics if any error occurs.
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run runs gRPC server.
func (a *App) Run() error {
	const op = "grpcapp.Run"

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", a.cfg.GRPC.Port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	a.log.Info("grpc server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop stops gRPC server.
func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping gRPC server", slog.String("port", a.cfg.GRPC.Port))

	a.gRPCServer.GracefulStop()
}
