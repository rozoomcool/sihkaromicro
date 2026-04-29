package interceptor

import (
	"log/slog"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewUnaryRecoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p any) error {
			log.Info("Recovered from panic", slog.Any("panic", p))
			if st, ok := p.(interface{ GRPCStatus() *status.Status }); ok {
				return st.GRPCStatus().Err()
			}

			return status.Errorf(codes.Internal, "internal error")
		}),
	}
	return recovery.UnaryServerInterceptor(recoveryOpts...)
}
