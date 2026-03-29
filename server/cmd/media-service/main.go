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
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/qmish/FProto/server/internal/media"
	"github.com/qmish/FProto/server/internal/observability"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50054", "gRPC listen address")
	metricsAddr := flag.String("metrics", ":9094", "Prometheus metrics address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	minioEndpoint := flag.String("minio", "localhost:9000", "MinIO endpoint")
	minioAccess := flag.String("minio-access", "minioadmin", "MinIO access key")
	minioSecret := flag.String("minio-secret", "minioadmin", "MinIO secret key")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("media-service")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "media-service", "0.5.0", *otlpEndpoint)
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

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		logger.Fatal("PostgreSQL connection failed", zap.Error(err))
	}
	defer pool.Close()

	store := media.NewStore(pool)
	if err := store.Migrate(ctx); err != nil {
		logger.Fatal("Migration failed", zap.Error(err))
	}

	s3Client, err := media.NewS3Client(*minioEndpoint, *minioAccess, *minioSecret, false)
	if err != nil {
		logger.Fatal("MinIO connection failed", zap.Error(err))
	}
	if err := s3Client.EnsureBucket(ctx); err != nil {
		logger.Fatal("Bucket creation failed", zap.Error(err))
	}

	svc := media.NewService(store, s3Client)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Fatal("Listen failed", zap.Error(err))
	}

	srv := grpc.NewServer(observability.GRPCServerOptions(logger)...)
	pb.RegisterMediaServiceServer(srv, media.NewGRPCServer(svc))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение MediaService...")
		srv.GracefulStop()
		cancel()
	}()

	logger.Info("MediaService запущен", zap.String("addr", *addr))
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("gRPC serve error", zap.Error(err))
	}
}
