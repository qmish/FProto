package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestNotify_SubscribeAndReceive(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	hub := newHub()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		handleClient(conn, serverKey, hub)
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

	sub := &Subscription{Channels: []string{"alerts"}, UserID: "test-user"}
	subMsg, _ := encodeSubscription(sub)
	ct, _ := session.Encrypt(subMsg)
	conn.WriteMessage(ct)

	time.Sleep(100 * time.Millisecond)

	notif := &Notification{
		ID:          "n-001",
		Channel:     "alerts",
		Title:       "Test Alert",
		Body:        "Hello from test",
		Priority:    "high",
		TimestampMs: nowMs(),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var received Notification

	go func() {
		defer wg.Done()
		respCT, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read: %v", err)
			return
		}
		respPT, _ := session.Decrypt(respCT)
		msgType, payload, _ := decodeFrame(respPT)
		if msgType != MsgNotification {
			t.Errorf("expected notification, got 0x%02x", msgType)
			return
		}
		json.Unmarshal(payload, &received)
	}()

	sent := hub.broadcast(notif)
	if sent != 1 {
		t.Fatalf("expected 1 sent, got %d", sent)
	}

	wg.Wait()

	if received.ID != "n-001" {
		t.Fatalf("expected n-001, got %s", received.ID)
	}
	if received.Title != "Test Alert" {
		t.Fatalf("expected Test Alert, got %s", received.Title)
	}
}

func TestNotify_ChannelFiltering(t *testing.T) {
	hub := newHub()

	serverKey, _ := crypto.GenerateKeyPair()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		handleClient(conn, serverKey, hub)
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

	sub := &Subscription{Channels: []string{"updates"}, UserID: "filter-user"}
	subMsg, _ := encodeSubscription(sub)
	ct, _ := session.Encrypt(subMsg)
	conn.WriteMessage(ct)

	time.Sleep(100 * time.Millisecond)

	notifAlerts := &Notification{ID: "n-a", Channel: "alerts", Title: "Should not arrive", TimestampMs: nowMs()}
	sent := hub.broadcast(notifAlerts)
	if sent != 0 {
		t.Fatalf("alert should not be sent to updates-only subscriber, got %d", sent)
	}

	notifUpdates := &Notification{ID: "n-u", Channel: "updates", Title: "Should arrive", TimestampMs: nowMs()}
	sent = hub.broadcast(notifUpdates)
	if sent != 1 {
		t.Fatalf("expected 1 sent for updates, got %d", sent)
	}
}

func TestNotify_EncodeDecodeMessages(t *testing.T) {
	n := &Notification{ID: "x", Channel: "ch", Title: "T", Body: "B", Priority: "low", TimestampMs: 1}
	encoded, _ := encodeNotification(n)
	msgType, payload, err := decodeFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if msgType != MsgNotification {
		t.Fatal("wrong type")
	}

	var decoded Notification
	json.Unmarshal(payload, &decoded)
	if decoded.ID != "x" || decoded.Channel != "ch" {
		t.Fatal("decode mismatch")
	}
}
