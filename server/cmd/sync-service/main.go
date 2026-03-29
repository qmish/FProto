package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qmish/FProto/server/internal/inbox"
	syncpkg "github.com/qmish/FProto/server/internal/sync"
)

func main() {
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	pushAddr := flag.String("push", "session-manager:50051", "PushService gRPC address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL: %v", err)
	}
	defer pool.Close()

	inboxStore := inbox.NewStore(pool)
	if err := inboxStore.Migrate(ctx); err != nil {
		log.Fatalf("Inbox migration: %v", err)
	}

	cfg := syncpkg.Config{
		KafkaBrokers:    strings.Split(*kafkaBrokers, ","),
		GroupID:         "sync-service",
		RedisAddr:       *redisAddr,
		PushServiceAddr: *pushAddr,
	}

	svc, err := syncpkg.New(ctx, cfg, inboxStore)
	if err != nil {
		log.Fatalf("SyncService init: %v", err)
	}
	defer svc.Close()

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version = sarama.V3_6_0_0
	admin, err := sarama.NewClusterAdmin(strings.Split(*kafkaBrokers, ","), kafkaCfg)
	if err != nil {
		log.Fatalf("Kafka admin: %v", err)
	}
	defer admin.Close()

	if err := svc.DiscoverTopics(admin); err != nil {
		log.Printf("Topic discovery: %v (will retry)", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение SyncService...")
		cancel()
	}()

	log.Println("SyncService запущен")
	if err := svc.Run(ctx); err != nil {
		log.Fatalf("SyncService run: %v", err)
	}
}
