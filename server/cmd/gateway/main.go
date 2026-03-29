package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/crypto"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"github.com/qmish/FProto/server/internal/transport"
)

func main() {
	wsAddr := flag.String("ws-addr", ":8080", "WebSocket listen address")
	quicAddr := flag.String("quic-addr", ":8443", "QUIC listen address")
	dispatcherAddr := flag.String("dispatcher", "localhost:50053", "Dispatcher gRPC address")
	flag.Parse()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Ошибка генерации ключей: %v", err)
	}
	log.Printf("Gateway публичный ключ: %x", serverKey.Public)

	var dispatcherClient pb.DispatcherServiceClient
	if *dispatcherAddr != "" {
		conn, err := grpc.NewClient(*dispatcherAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Dispatcher gRPC connect: %v (работаем в echo-режиме)", err)
		} else {
			dispatcherClient = pb.NewDispatcherServiceClient(conn)
			log.Printf("Gateway -> Dispatcher: %s", *dispatcherAddr)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("WS upgrade error: %v", err)
			return
		}
		defer ws.Close()
		handleConn(ctx, ws, serverKey, dispatcherClient)
	})

	go func() {
		log.Printf("WebSocket Gateway на %s", *wsAddr)
		if err := http.ListenAndServe(*wsAddr, nil); err != nil {
			log.Printf("WS server error: %v", err)
		}
	}()

	tlsConf := transport.GenerateSelfSignedTLS()
	go func() {
		listener, err := transport.ListenQUIC(*quicAddr, tlsConf)
		if err != nil {
			log.Printf("QUIC listen error: %v", err)
			return
		}
		log.Printf("QUIC Gateway на %s", *quicAddr)
		for {
			qconn, err := transport.AcceptQUIC(ctx, listener)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("QUIC accept error: %v", err)
				continue
			}
			go func() {
				defer qconn.Close()
				handleConn(ctx, qconn, serverKey, dispatcherClient)
			}()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Gateway завершает работу...")
	cancel()
}

func handleConn(ctx context.Context, conn transport.Conn, serverKey *crypto.KeyPair, dispatcher pb.DispatcherServiceClient) {
	session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Printf("[%s] Handshake error: %v", conn.Type(), err)
		return
	}
	log.Printf("[%s] Handshake OK: %s", conn.Type(), conn.RemoteAddr())

	for {
		ct, err := conn.ReadMessage()
		if err != nil {
			return
		}
		pt, err := session.Decrypt(ct)
		if err != nil {
			log.Printf("[%s] Decrypt error: %v", conn.Type(), err)
			return
		}

		if dispatcher != nil {
			var frame pb.Frame
			if err := proto.Unmarshal(pt, &frame); err == nil && len(frame.EncryptedPayload) > 0 {
				var appMsg pb.AppMessage
				if err := proto.Unmarshal(frame.EncryptedPayload, &appMsg); err == nil {
					resp, err := dispatcher.Dispatch(ctx, &pb.DispatchRequest{
						SessionId: []byte(conn.RemoteAddr()),
						SenderId:  frame.SessionId,
						Message:   &appMsg,
					})
					if err != nil {
						log.Printf("[%s] Dispatch error: %v", conn.Type(), err)
					} else {
						ack := &pb.AppMessage{
							MessageId: resp.AckMessageId,
							Body: &pb.AppMessage_Ack{
								Ack: &pb.Ack{
									AckMessageId: resp.AckMessageId,
									Status:       pb.Ack_RECEIVED,
								},
							},
						}
						ackBytes, _ := proto.Marshal(ack)
						enc, err := session.Encrypt(ackBytes)
						if err == nil {
							_ = conn.WriteMessage(enc)
						}
						continue
					}
				}
			}
		}

		// Fallback: echo mode
		resp := fmt.Appendf(nil, "echo: %s", pt)
		enc, err := session.Encrypt(resp)
		if err != nil {
			return
		}
		if err := conn.WriteMessage(enc); err != nil {
			return
		}
	}
}

func init() {
	_ = tls.Config{}
}
