package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", "localhost:9100", "WebSocket address")
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*addr)
	case "client":
		runClient(*addr)
	default:
		log.Fatalf("unknown mode: %s (use server or client)", *mode)
	}
}

func runServer(addr string) {
	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("accept: %v", err)
			return
		}
		defer conn.Close()

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			log.Printf("handshake: %v", err)
			return
		}
		log.Printf("[server] client connected: %s", conn.RemoteAddr())

		for {
			ct, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[server] read: %v", err)
				return
			}

			pt, err := session.Decrypt(ct)
			if err != nil {
				log.Printf("[server] decrypt: %v", err)
				return
			}
			log.Printf("[server] received: %s", string(pt))

			reply, _ := json.Marshal(map[string]interface{}{
				"echo":      string(pt),
				"timestamp": time.Now().UnixMilli(),
			})

			enc, err := session.Encrypt(reply)
			if err != nil {
				log.Printf("[server] encrypt: %v", err)
				return
			}
			if err := conn.WriteMessage(enc); err != nil {
				log.Printf("[server] write: %v", err)
				return
			}
		}
	})

	log.Printf("[server] listening on ws://%s/ws", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func runClient(addr string) {
	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	url := fmt.Sprintf("ws://%s/ws", addr)
	conn, err := transport.DialWS(url)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Fatalf("handshake: %v", err)
	}
	log.Println("[client] handshake complete")

	messages := []string{"Hello FProto!", "Encrypted message", "Goodbye!"}
	for _, msg := range messages {
		ct, err := session.Encrypt([]byte(msg))
		if err != nil {
			log.Fatalf("encrypt: %v", err)
		}
		if err := conn.WriteMessage(ct); err != nil {
			log.Fatalf("write: %v", err)
		}

		replyCT, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		replyPT, err := session.Decrypt(replyCT)
		if err != nil {
			log.Fatalf("decrypt: %v", err)
		}
		log.Printf("[client] echo: %s", string(replyPT))
	}
	log.Println("[client] session complete")
}
