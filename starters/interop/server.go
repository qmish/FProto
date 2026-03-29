package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func main() {
	addr := flag.String("addr", "localhost:9500", "WebSocket listen address")
	flag.Parse()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			log.Printf("handshake: %v", err)
			return
		}
		log.Printf("[interop] client connected: %s (%s)", conn.RemoteAddr(), conn.Type())

		for {
			ct, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[interop] client disconnected: %s", conn.RemoteAddr())
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				log.Printf("[interop] decrypt: %v", err)
				return
			}
			log.Printf("[interop] echo: %d bytes from %s", len(pt), conn.RemoteAddr())

			enc, err := session.Encrypt(pt)
			if err != nil {
				log.Printf("[interop] encrypt: %v", err)
				return
			}
			if err := conn.WriteMessage(enc); err != nil {
				log.Printf("[interop] write: %v", err)
				return
			}
		}
	})

	log.Printf("[interop] echo server listening on ws://%s/ws", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
