package grpcapp

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/rozoomcool/sihkaromicro/sources/internal/config"
	"github.com/rozoomcool/sihkaromicro/sources/internal/handler"
	"github.com/rozoomcool/sihkaromicro/sources/internal/interceptor"
	"github.com/rozoomcool/sihkaromicro/sources/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	cfg        *config.Config
}

func New(
	sourceService service.SourceService,
	log *slog.Logger,
	cfg *config.Config,
) *App {
	authInterceptor, err := interceptor.NewAuthInterceptor(context.Background(), cfg.Keycloak)
	if err != nil {
		panic(fmt.Sprintf("failed to create auth interceptor: %v", err))
	}

	gRPCServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.NewUnaryRecoveryInterceptor(log),
			interceptor.NewUnaryLoggerInterceptor(log),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			interceptor.NewStreamRecoveryInterceptor(log),
			interceptor.NewStreamLoggerInterceptor(log),
			authInterceptor.Stream(),
		),
	)

	sourceHandler := handler.NewSourceHandler(sourceService, log)
	sourceHandler.Register(gRPCServer)

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(gRPCServer, healthcheck)
	healthcheck.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		cfg:        cfg,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

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

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping gRPC server", slog.String("port", a.cfg.GRPC.Port))

	a.gRPCServer.GracefulStop()
}
