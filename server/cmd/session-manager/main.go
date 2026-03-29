package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/qmish/FProto/server/internal/observability"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"github.com/qmish/FProto/server/internal/session"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC listen address")
	metricsAddr := flag.String("metrics", ":9091", "Prometheus metrics address")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("session-manager")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "session-manager", "0.5.0", *otlpEndpoint)
	if err != nil {
		logger.Warn("Tracing init failed, continuing without traces", zap.Error(err))
	} else {
		defer func() { _ = shutdownTracer(ctx) }()
	}

	mp, metricsHandler, err := observability.InitMeter()
	if err != nil {
		logger.Fatal("Metrics init failed", zap.Error(err))
	}
	defer func() { _ = mp.Shutdown(ctx) }()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", metricsHandler)
		logger.Info("Metrics endpoint", zap.String("addr", *metricsAddr))
		if err := http.ListenAndServe(*metricsAddr, mux); err != nil {
			logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Redis connection failed", zap.Error(err))
	}
	logger.Info("Подключено к Redis", zap.String("addr", *redisAddr))

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
	}
	defer pool.Close()
	logger.Info("Подключено к PostgreSQL")

	pgStore := session.NewPGStore(pool)
	if err := pgStore.Migrate(ctx); err != nil {
		logger.Fatal("Migration failed", zap.Error(err))
	}

	redisStore := session.NewRedisStore(rdb, 0)
	mgr := session.NewManager(redisStore, pgStore, session.DefaultConfig())

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Fatal("Listen failed", zap.Error(err))
	}

	srv := grpc.NewServer(observability.GRPCServerOptions(logger)...)
	pb.RegisterSessionServiceServer(srv, session.NewGRPCServer(mgr))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение работы...")
		srv.GracefulStop()
		cancel()
	}()

	logger.Info("Session Manager запущен", zap.String("addr", *addr))
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("gRPC serve error", zap.Error(err))
	}
}
