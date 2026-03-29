package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/qmish/FProto/server/internal/crypto"
	"github.com/qmish/FProto/server/internal/observability"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50052", "gRPC listen address")
	metricsAddr := flag.String("metrics", ":9092", "Prometheus metrics address")
	otlpEndpoint := flag.String("otlp", "localhost:4317", "OTLP gRPC endpoint")
	flag.Parse()

	logger, err := observability.NewLogger("crypto-handler")
	if err != nil {
		panic("logger init: " + err.Error())
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracer, err := observability.InitTracer(ctx, "crypto-handler", "0.5.0", *otlpEndpoint)
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

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		logger.Fatal("Ошибка генерации ключей", zap.Error(err))
	}
	logger.Info("Серверный публичный ключ сгенерирован")

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Fatal("Listen failed", zap.Error(err))
	}

	srv := grpc.NewServer(observability.GRPCServerOptions(logger)...)
	pb.RegisterCryptoServiceServer(srv, crypto.NewCryptoGRPCServer(serverKey))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		logger.Info("Завершение работы...")
		srv.GracefulStop()
		cancel()
	}()

	logger.Info("Crypto Handler запущен", zap.String("addr", *addr))
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("gRPC serve error", zap.Error(err))
	}
}
