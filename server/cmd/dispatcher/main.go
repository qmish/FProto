package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/qmish/FProto/server/internal/dispatcher"
	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
	"github.com/qmish/FProto/server/internal/outbox"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50053", "gRPC listen address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer pool.Close()

	outboxStore := outbox.NewStore(pool)
	if err := outboxStore.Migrate(ctx); err != nil {
		log.Fatalf("Outbox migration failed: %v", err)
	}

	producer, err := kafkapkg.NewProducer([]string{*kafkaBrokers}, nil)
	if err != nil {
		log.Fatalf("Kafka producer failed: %v", err)
	}
	defer producer.Close()

	relay := outbox.NewRelay(outboxStore, producer)
	go relay.Run(ctx)

	d := dispatcher.New(outboxStore)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterDispatcherServiceServer(srv, dispatcher.NewGRPCServer(d))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение работы Dispatcher...")
		srv.GracefulStop()
		cancel()
	}()

	log.Printf("Dispatcher запущен на %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve: %v", err)
	}
}
