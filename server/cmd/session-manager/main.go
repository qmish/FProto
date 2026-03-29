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
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"github.com/qmish/FProto/server/internal/session"
)

func main() {
	addr := flag.String("addr", ":50051", "gRPC listen address")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: *redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	log.Printf("Подключено к Redis: %s", *redisAddr)

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer pool.Close()
	log.Printf("Подключено к PostgreSQL")

	pgStore := session.NewPGStore(pool)
	if err := pgStore.Migrate(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	redisStore := session.NewRedisStore(rdb, 0)
	mgr := session.NewManager(redisStore, pgStore, session.DefaultConfig())

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterSessionServiceServer(srv, session.NewGRPCServer(mgr))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение работы...")
		srv.GracefulStop()
		cancel()
	}()

	log.Printf("Session Manager запущен на %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve: %v", err)
	}
}
