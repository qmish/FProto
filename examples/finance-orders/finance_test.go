package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestFinance_SignedOrderRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	book := &orderBook{}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		handleTrader(conn, serverKey, book)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	clientKey, _ := crypto.GenerateKeyPair()
	sigKey, _ := GenerateSigningKey()

	url := fmt.Sprintf("ws://%s/ws", ln.Addr().String())
	conn, err := transport.DialWS(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	order := &Order{
		OrderID:     "TEST-001",
		Instrument:  "BTC/USD",
		Side:        "buy",
		Quantity:    1.0,
		Price:       50000.0,
		TimestampMs: nowMs(),
	}

	msg, err := signOrder(order, sigKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	ct, _ := session.Encrypt(msg)
	if err := conn.WriteMessage(ct); err != nil {
		t.Fatalf("write: %v", err)
	}

	respCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	respPT, _ := session.Decrypt(respCT)
	_, payload, _ := decodeFrame(respPT)

	var resp OrderResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Status != "filled" {
		t.Fatalf("expected filled, got %s", resp.Status)
	}
	if resp.OrderID != "TEST-001" {
		t.Fatalf("order id mismatch")
	}
}

func TestFinance_InvalidSignature(t *testing.T) {
	sigKey, _ := GenerateSigningKey()

	order := &Order{OrderID: "BAD-001", Instrument: "ETH/USD", Side: "sell", Quantity: 5.0, Price: 3000.0, TimestampMs: nowMs()}
	msg, _ := signOrder(order, sigKey)

	_, payload, _ := decodeFrame(msg)

	var signed SignedOrder
	json.Unmarshal(payload, &signed)

	signed.Signature[0] ^= 0xFF
	tampered, _ := json.Marshal(signed)

	_, err := verifyOrder(tampered)
	if err == nil {
		t.Fatal("expected signature verification to fail")
	}
}

func TestFinance_EncodeDecodeOrder(t *testing.T) {
	sigKey, _ := GenerateSigningKey()
	order := &Order{OrderID: "T-001", Instrument: "XAU/USD", Side: "buy", Quantity: 100.0, Price: 2000.0, TimestampMs: 1234567890}

	msg, err := signOrder(order, sigKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	msgType, payload, err := decodeFrame(msg)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msgType != MsgOrder {
		t.Fatalf("expected order type")
	}

	decoded, err := verifyOrder(payload)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if decoded.OrderID != "T-001" || decoded.Instrument != "XAU/USD" {
		t.Fatalf("decode mismatch")
	}
}
