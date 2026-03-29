package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/qmish/FProto/server/internal/crypto"
	"github.com/qmish/FProto/server/internal/transport"
)

func main() {
	addr := flag.String("addr", ":8080", "адрес для прослушивания")
	flag.Parse()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Ошибка генерации ключей: %v", err)
	}
	log.Printf("Серверный публичный ключ: %x", serverKey.Public)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnection(w, r, serverKey)
	})

	log.Printf("Эхо-сервер запущен на %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request, serverKey *crypto.KeyPair) {
	ws, err := transport.UpgradeHTTP(w, r)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer ws.Close()

	session, err := crypto.ServerHandshake(serverKey, ws.ReadMessage, ws.WriteMessage)
	if err != nil {
		log.Printf("Handshake error: %v", err)
		return
	}
	log.Printf("Handshake завершён для %s", r.RemoteAddr)

	for {
		ciphertext, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			return
		}

		plaintext, err := session.Decrypt(ciphertext)
		if err != nil {
			log.Printf("Decrypt error: %v", err)
			return
		}

		response := fmt.Appendf(nil, "echo: %s", plaintext)

		encrypted, err := session.Encrypt(response)
		if err != nil {
			log.Printf("Encrypt error: %v", err)
			return
		}

		if err := ws.WriteMessage(encrypted); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
}
