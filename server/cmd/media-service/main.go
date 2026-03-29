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

	"github.com/qmish/FProto/server/internal/media"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50054", "gRPC listen address")
	pgDSN := flag.String("pg", "postgres://fproto:fproto_dev@localhost:5432/fproto?sslmode=disable", "PostgreSQL DSN")
	minioEndpoint := flag.String("minio", "localhost:9000", "MinIO endpoint")
	minioAccess := flag.String("minio-access", "minioadmin", "MinIO access key")
	minioSecret := flag.String("minio-secret", "minioadmin", "MinIO secret key")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL: %v", err)
	}
	defer pool.Close()

	store := media.NewStore(pool)
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("Migration: %v", err)
	}

	s3Client, err := media.NewS3Client(*minioEndpoint, *minioAccess, *minioSecret, false)
	if err != nil {
		log.Fatalf("MinIO: %v", err)
	}
	if err := s3Client.EnsureBucket(ctx); err != nil {
		log.Fatalf("Bucket: %v", err)
	}

	svc := media.NewService(store, s3Client)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Listen: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterMediaServiceServer(srv, media.NewGRPCServer(svc))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение MediaService...")
		srv.GracefulStop()
		cancel()
	}()

	log.Printf("MediaService запущен на %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve: %v", err)
	}
}
