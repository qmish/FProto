package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/qmish/FProto/server/internal/crypto"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func main() {
	addr := flag.String("addr", ":50052", "gRPC listen address")
	flag.Parse()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Ошибка генерации ключей: %v", err)
	}
	log.Printf("Серверный публичный ключ: %x", serverKey.Public)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	srv := grpc.NewServer()
	pb.RegisterCryptoServiceServer(srv, crypto.NewCryptoGRPCServer(serverKey))

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Завершение работы...")
		srv.GracefulStop()
	}()

	log.Printf("Crypto Handler запущен на %s", *addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("gRPC serve: %v", err)
	}
}
