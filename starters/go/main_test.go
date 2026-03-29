package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/session"
	"github.com/qmish/FProto/proto-core/transport"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func startTestServer(t *testing.T, addr string) {
	t.Helper()
	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		sess, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			return
		}

		for {
			ct, err := conn.ReadMessage()
			if err != nil {
				return
			}
			pt, err := sess.Decrypt(ct)
			if err != nil {
				return
			}
			enc, err := sess.Encrypt(pt)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(enc); err != nil {
				return
			}
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	time.Sleep(50 * time.Millisecond)
}

func TestEchoRoundTrip(t *testing.T) {
	addr := freePort(t)
	startTestServer(t, addr)

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sess, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatal(err)
	}

	messages := []string{"hello", "world", "fproto"}
	for _, msg := range messages {
		ct, err := sess.Encrypt([]byte(msg))
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(ct); err != nil {
			t.Fatal(err)
		}

		replyCT, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		replyPT, err := sess.Decrypt(replyCT)
		if err != nil {
			t.Fatal(err)
		}

		if string(replyPT) != msg {
			t.Errorf("expected %q, got %q", msg, string(replyPT))
		}
	}
}

func TestNoiseHandshake(t *testing.T) {
	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	if len(serverKey.Public) != 32 {
		t.Errorf("expected 32 byte public key, got %d", len(serverKey.Public))
	}
	if len(clientKey.Private) != 32 {
		t.Errorf("expected 32 byte private key, got %d", len(clientKey.Private))
	}
}

func TestAEADEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	aead, err := crypto.NewAEAD(key)
	if err != nil {
		t.Fatal(err)
	}

	sid := []byte("test-session")
	plaintext := []byte("secret data")

	ct := crypto.EncryptAEAD(aead, sid, 1, plaintext)
	pt, err := crypto.DecryptAEAD(aead, sid, 1, ct)
	if err != nil {
		t.Fatal(err)
	}

	if string(pt) != string(plaintext) {
		t.Errorf("expected %q, got %q", plaintext, pt)
	}
}

func TestRekeyDeterministic(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	k1, err := crypto.Rekey(key)
	if err != nil {
		t.Fatal(err)
	}
	k2, err := crypto.Rekey(key)
	if err != nil {
		t.Fatal(err)
	}

	if string(k1) != string(k2) {
		t.Error("rekey is not deterministic")
	}
	if string(k1) == string(key) {
		t.Error("rekey returned same key")
	}
}

func TestZeroize(t *testing.T) {
	buf := []byte{1, 2, 3, 4, 5}
	crypto.Zeroize(buf)
	for i, b := range buf {
		if b != 0 {
			t.Errorf("byte %d not zeroed: %d", i, b)
		}
	}
}

func TestSessionStateMachine(t *testing.T) {
	next, destroyed, err := session.Transition(session.StatusConnecting, session.EventHandshakeOK)
	if err != nil {
		t.Fatal(err)
	}
	if next != session.StatusActive || destroyed {
		t.Errorf("expected active/false, got %v/%v", next, destroyed)
	}

	next, destroyed, err = session.Transition(session.StatusActive, session.EventIdleTimeout)
	if err != nil {
		t.Fatal(err)
	}
	if next != session.StatusSleeping || destroyed {
		t.Errorf("expected sleeping/false, got %v/%v", next, destroyed)
	}

	next, destroyed, err = session.Transition(session.StatusSleeping, session.EventPacketReceived)
	if err != nil {
		t.Fatal(err)
	}
	if next != session.StatusActive || destroyed {
		t.Errorf("expected active/false, got %v/%v", next, destroyed)
	}

	_, destroyed, err = session.Transition(session.StatusConnecting, session.EventHandshakeFail)
	if err != nil {
		t.Fatal(err)
	}
	if !destroyed {
		t.Error("expected destroyed on handshake fail")
	}
}

func TestNonceUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := uint64(0); i < 1000; i++ {
		n := crypto.BuildNonce(i)
		key := string(n)
		if seen[key] {
			t.Fatalf("duplicate nonce at seq %d", i)
		}
		seen[key] = true
	}
}

func TestLargePayloadEcho(t *testing.T) {
	addr := freePort(t)
	startTestServer(t, addr)

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	sess, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatal(err)
	}

	payload := make([]byte, 32*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	ct, err := sess.Encrypt(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(ct); err != nil {
		t.Fatal(err)
	}

	replyCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	replyPT, err := sess.Decrypt(replyCT)
	if err != nil {
		t.Fatal(err)
	}

	if len(replyPT) != len(payload) {
		t.Errorf("expected %d bytes, got %d", len(payload), len(replyPT))
	}
}

func TestMultipleClients(t *testing.T) {
	addr := freePort(t)
	startTestServer(t, addr)

	for c := 0; c < 3; c++ {
		clientKey, _ := crypto.GenerateKeyPair()
		conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
		if err != nil {
			t.Fatal(err)
		}

		sess, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			t.Fatal(err)
		}

		msg := fmt.Sprintf("client-%d", c)
		ct, _ := sess.Encrypt([]byte(msg))
		conn.WriteMessage(ct)

		rCT, _ := conn.ReadMessage()
		rPT, _ := sess.Decrypt(rCT)
		if string(rPT) != msg {
			t.Errorf("client %d: expected %q, got %q", c, msg, string(rPT))
		}
		conn.Close()
	}
}

func TestJSONEcho(t *testing.T) {
	addr := freePort(t)
	startTestServer(t, addr)

	clientKey, _ := crypto.GenerateKeyPair()
	conn, _ := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
	defer conn.Close()

	sess, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)

	data := map[string]interface{}{"type": "ping", "seq": float64(42)}
	payload, _ := json.Marshal(data)

	ct, _ := sess.Encrypt(payload)
	conn.WriteMessage(ct)

	rCT, _ := conn.ReadMessage()
	rPT, _ := sess.Decrypt(rCT)

	var result map[string]interface{}
	if err := json.Unmarshal(rPT, &result); err != nil {
		t.Fatal(err)
	}
	if result["type"] != "ping" {
		t.Errorf("expected type=ping, got %v", result["type"])
	}
}

func TestBidirectionalExchange(t *testing.T) {
	addr := freePort(t)

	serverKey, _ := crypto.GenerateKeyPair()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _ := transport.UpgradeHTTP(w, r)
		defer conn.Close()
		sess, _ := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)

		serverMsg := []byte("server-push")
		ct, _ := sess.Encrypt(serverMsg)
		conn.WriteMessage(ct)

		clientCT, _ := conn.ReadMessage()
		clientPT, _ := sess.Decrypt(clientCT)

		reply, _ := sess.Encrypt(clientPT)
		conn.WriteMessage(reply)
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	time.Sleep(50 * time.Millisecond)

	clientKey, _ := crypto.GenerateKeyPair()
	conn, _ := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
	defer conn.Close()
	sess, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)

	pushCT, _ := conn.ReadMessage()
	pushPT, _ := sess.Decrypt(pushCT)
	if string(pushPT) != "server-push" {
		t.Errorf("expected 'server-push', got %q", string(pushPT))
	}

	ct, _ := sess.Encrypt([]byte("client-reply"))
	conn.WriteMessage(ct)

	echoCT, _ := conn.ReadMessage()
	echoPT, _ := sess.Decrypt(echoCT)
	if string(echoPT) != "client-reply" {
		t.Errorf("expected 'client-reply', got %q", string(echoPT))
	}
}
