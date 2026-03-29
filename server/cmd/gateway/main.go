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

	"github.com/qmish/FProto/server/internal/crypto"
	"github.com/qmish/FProto/server/internal/transport"
)

func main() {
	wsAddr := flag.String("ws-addr", ":8080", "WebSocket listen address")
	quicAddr := flag.String("quic-addr", ":8443", "QUIC listen address")
	flag.Parse()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Ошибка генерации ключей: %v", err)
	}
	log.Printf("Gateway публичный ключ: %x", serverKey.Public)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("WS upgrade error: %v", err)
			return
		}
		defer ws.Close()
		handleConn(ws, serverKey)
	})

	go func() {
		log.Printf("WebSocket Gateway на %s", *wsAddr)
		if err := http.ListenAndServe(*wsAddr, nil); err != nil {
			log.Printf("WS server error: %v", err)
		}
	}()

	// QUIC listener
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
				handleConn(qconn, serverKey)
			}()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Gateway завершает работу...")
	cancel()
}

func handleConn(conn transport.Conn, serverKey *crypto.KeyPair) {
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
