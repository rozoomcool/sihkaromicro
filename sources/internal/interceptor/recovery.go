package interceptor

import (
	"log/slog"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUnaryRecoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return recovery.UnaryServerInterceptor(recoveryOpts(log)...)
}

func NewStreamRecoveryInterceptor(log *slog.Logger) grpc.StreamServerInterceptor {
	return recovery.StreamServerInterceptor(recoveryOpts(log)...)
}

func recoveryOpts(log *slog.Logger) []recovery.Option {
	return []recovery.Option{
		recovery.WithRecoveryHandler(func(p any) error {
			log.Error("recovered from panic", slog.Any("panic", p))
			if st, ok := p.(interface{ GRPCStatus() *status.Status }); ok {
				return st.GRPCStatus().Err()
			}
			return status.Errorf(codes.Internal, "internal error")
		}),
	}
}
