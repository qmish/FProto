package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qmish/FProto/server/internal/commands"
	"github.com/qmish/FProto/server/internal/groups"
)

func main() {
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	kafkaBrokers := flag.String("kafka", "localhost:9092", "Kafka brokers (comma-separated)")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL: %v", err)
	}
	defer pool.Close()

	groupStore := groups.NewStore(pool)
	if err := groupStore.Migrate(ctx); err != nil {
		log.Fatalf("Groups migration: %v", err)
	}

	handler := commands.NewHandler(groupStore)
	svc, err := commands.NewService(strings.Split(*kafkaBrokers, ","), "command-service", handler)
	if err != nil {
		log.Fatalf("CommandService: %v", err)
	}
	defer svc.Close()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение CommandService...")
		cancel()
	}()

	log.Println("CommandService запущен")
	if err := svc.Run(ctx); err != nil {
		log.Fatalf("CommandService: %v", err)
	}
}
