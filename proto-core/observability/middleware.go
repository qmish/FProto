package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// GRPCServerOptions returns gRPC server options with OTel stats handler
// and a unary logging interceptor.
func GRPCServerOptions(logger *zap.Logger) []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(loggingUnaryInterceptor(logger)),
	}
}

// GRPCDialOptions returns gRPC dial options with OTel stats handler for clients.
func GRPCDialOptions() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

func loggingUnaryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
		}

		sc := trace.SpanContextFromContext(ctx)
		if sc.IsValid() {
			fields = append(fields,
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
		}

		if err != nil {
			st, _ := status.FromError(err)
			fields = append(fields, zap.String("grpc_code", st.Code().String()))
			logger.Warn("gRPC call failed", fields...)
		} else {
			logger.Info("gRPC call", fields...)
		}

		return resp, err
	}
}
