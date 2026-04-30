package interceptor

import (
	"context"
	"log/slog"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

func NewUnaryLoggerInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	loggingOpts := []logging.Option{
		logging.WithLogOnEvents(
			logging.StartCall, logging.FinishCall,
			// logging.PayloadReceived, logging.PayloadSent,
		),
	}
	return logging.UnaryServerInterceptor(InterceptorLogger(log), loggingOpts...)
}

// InterceptorLogger adapts slog logger to interceptor logger.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}
