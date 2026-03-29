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

	"github.com/qmish/FProto/server/internal/dispatcher"
	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
	"github.com/qmish/FProto/proto-core/observability"
	"github.com/qmish/FProto/server/internal/outbox"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50053", "gRPC listen address")
	metricsAddr := flag.String("metrics", ":9093", "Prometheus metrics address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("dispatcher")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "dispatcher", "0.5.0", *otlpEndpoint)
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

	outboxStore := outbox.NewStore(pool)
	if err := outboxStore.Migrate(ctx); err != nil {
		logger.Fatal("Outbox migration failed", zap.Error(err))
	}

	producer, err := kafkapkg.NewProducer([]string{*kafkaBrokers}, nil)
	if err != nil {
		logger.Fatal("Kafka producer failed", zap.Error(err))
	}
	defer producer.Close()

	relay := outbox.NewRelay(outboxStore, producer)
	go relay.Run(ctx)

	d := dispatcher.New(outboxStore)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Fatal("Listen failed", zap.Error(err))
	}

	srv := grpc.NewServer(observability.GRPCServerOptions(logger)...)
	pb.RegisterDispatcherServiceServer(srv, dispatcher.NewGRPCServer(d))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение работы Dispatcher...")
		srv.GracefulStop()
		cancel()
	}()

	logger.Info("Dispatcher запущен", zap.String("addr", *addr))
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("gRPC serve error", zap.Error(err))
	}
}
