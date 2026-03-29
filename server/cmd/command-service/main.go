package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/qmish/FProto/server/internal/commands"
	"github.com/qmish/FProto/server/internal/groups"
	"github.com/qmish/FProto/server/internal/observability"
)

func main() {
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	metricsAddr := flag.String("metrics", ":9096", "Prometheus metrics address")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("command-service")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "command-service", "0.5.0", *otlpEndpoint)
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

	groupStore := groups.NewStore(pool)
	if err := groupStore.Migrate(ctx); err != nil {
		logger.Fatal("Groups migration failed", zap.Error(err))
	}

	handler := commands.NewHandler(groupStore)
	svc, err := commands.NewService(strings.Split(*kafkaBrokers, ","), "command-service", handler)
	if err != nil {
		logger.Fatal("CommandService init failed", zap.Error(err))
	}
	defer svc.Close()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение CommandService...")
		cancel()
	}()

	logger.Info("CommandService запущен")
	if err := svc.Run(ctx); err != nil {
		logger.Fatal("CommandService run error", zap.Error(err))
	}
}
