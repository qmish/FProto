package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/qmish/FProto/server/internal/inbox"
	"github.com/qmish/FProto/server/internal/observability"
	syncpkg "github.com/qmish/FProto/server/internal/sync"
)

func main() {
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	pushAddr := flag.String("push", "session-manager:50051", "PushService gRPC address")
	metricsAddr := flag.String("metrics", ":9095", "Prometheus metrics address")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("sync-service")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "sync-service", "0.5.0", *otlpEndpoint)
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

	inboxStore := inbox.NewStore(pool)
	if err := inboxStore.Migrate(ctx); err != nil {
		logger.Fatal("Inbox migration failed", zap.Error(err))
	}

	cfg := syncpkg.Config{
		KafkaBrokers:    strings.Split(*kafkaBrokers, ","),
		GroupID:         "sync-service",
		RedisAddr:       *redisAddr,
		PushServiceAddr: *pushAddr,
	}

	svc, err := syncpkg.New(ctx, cfg, inboxStore)
	if err != nil {
		logger.Fatal("SyncService init failed", zap.Error(err))
	}
	defer svc.Close()

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version = sarama.V3_6_0_0
	admin, err := sarama.NewClusterAdmin(strings.Split(*kafkaBrokers, ","), kafkaCfg)
	if err != nil {
		logger.Fatal("Kafka admin connection failed", zap.Error(err))
	}
	defer admin.Close()

	if err := svc.DiscoverTopics(admin); err != nil {
		logger.Warn("Topic discovery failed, will retry", zap.Error(err))
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение SyncService...")
		cancel()
	}()

	logger.Info("SyncService запущен")
	if err := svc.Run(ctx); err != nil {
		logger.Fatal("SyncService run error", zap.Error(err))
	}
}
