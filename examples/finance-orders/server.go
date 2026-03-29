package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

type orderBook struct {
	mu     sync.Mutex
	orders []Order
	seq    atomic.Int64
}

func (ob *orderBook) accept(o Order) string {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.orders = append(ob.orders, o)
	return fmt.Sprintf("accepted-%d", ob.seq.Add(1))
}

func (ob *orderBook) count() int {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	return len(ob.orders)
}

func runServer(addr string) {
	serverKey, _ := crypto.GenerateKeyPair()
	book := &orderBook{}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		handleTrader(conn, serverKey, book)
	})

	log.Printf("Finance WS server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleTrader(conn transport.Conn, key *crypto.KeyPair, book *orderBook) {
	defer conn.Close()

	session, err := crypto.ServerHandshake(key, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Printf("handshake: %v", err)
		return
	}
	log.Printf("trader connected: %s", conn.RemoteAddr())

	for {
		ct, err := conn.ReadMessage()
		if err != nil {
			return
		}
		pt, err := session.Decrypt(ct)
		if err != nil {
			log.Printf("decrypt: %v", err)
			return
		}

		msgType, payload, err := decodeFrame(pt)
		if err != nil {
			continue
		}

		switch msgType {
		case MsgOrder:
			order, err := verifyOrder(payload)
			if err != nil {
				resp := &OrderResponse{Status: "rejected", Message: err.Error()}
				respBytes, _ := encodeResponse(resp)
				enc, _ := session.Encrypt(respBytes)
				conn.WriteMessage(enc)
				continue
			}

			execID := book.accept(*order)
			log.Printf("order %s: %s %s %.2f @ %.2f (total=%d)",
				order.OrderID, order.Side, order.Instrument,
				order.Quantity, order.Price, book.count())

			resp := &OrderResponse{
				OrderID: order.OrderID,
				Status:  "filled",
				Message: execID,
			}
			respBytes, _ := encodeResponse(resp)
			enc, _ := session.Encrypt(respBytes)
			conn.WriteMessage(enc)

		default:
			log.Printf("unknown msg type: 0x%02x", msgType)
		}
	}
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", ":8080", "WebSocket address")
	flag.Parse()

	if *mode == "server" {
		runServer(*addr)
	} else {
		runClient(*addr)
	}
}

func runClient(addr string) {
	clientKey, _ := crypto.GenerateKeyPair()
	sigKey, _ := GenerateSigningKey()

	url := fmt.Sprintf("ws://localhost%s/ws", addr)
	conn, err := transport.DialWS(url)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Fatalf("handshake: %v", err)
	}
	log.Println("connected to trading server")

	orders := []Order{
		{OrderID: "ORD-001", Instrument: "BTC/USD", Side: "buy", Quantity: 0.5, Price: 65000.0, TimestampMs: nowMs()},
		{OrderID: "ORD-002", Instrument: "ETH/USD", Side: "sell", Quantity: 10.0, Price: 3200.0, TimestampMs: nowMs()},
		{OrderID: "ORD-003", Instrument: "BTC/USD", Side: "buy", Quantity: 1.0, Price: 64999.0, TimestampMs: nowMs()},
	}

	for _, order := range orders {
		msg, _ := signOrder(&order, sigKey)
		ct, _ := session.Encrypt(msg)
		if err := conn.WriteMessage(ct); err != nil {
			log.Fatalf("write: %v", err)
		}

		respCT, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		respPT, _ := session.Decrypt(respCT)
		_, respPayload, _ := decodeFrame(respPT)

		var resp OrderResponse
		json.Unmarshal(respPayload, &resp)
		log.Printf("response: %s %s %s", resp.OrderID, resp.Status, resp.Message)
	}
}
