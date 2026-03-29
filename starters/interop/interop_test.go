package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/qmish/FProto/proto-core/crypto"
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

func startInteropServer(t *testing.T, addr string) {
	t.Helper()
	serverKey, _ := crypto.GenerateKeyPair()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			return
		}

		for {
			ct, err := conn.ReadMessage()
			if err != nil {
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				return
			}
			enc, _ := session.Encrypt(pt)
			conn.WriteMessage(enc)
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go srv.ListenAndServe()
	t.Cleanup(func() { srv.Close() })
	time.Sleep(50 * time.Millisecond)
}

func TestInteropGoEchoRoundTrip(t *testing.T) {
	addr := freePort(t)
	startInteropServer(t, addr)

	clientKey, _ := crypto.GenerateKeyPair()
	conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatal(err)
	}

	messages := []string{"interop-hello", "interop-world", "cross-language-test"}
	for _, msg := range messages {
		ct, _ := session.Encrypt([]byte(msg))
		conn.WriteMessage(ct)

		replyCT, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		replyPT, err := session.Decrypt(replyCT)
		if err != nil {
			t.Fatal(err)
		}
		if string(replyPT) != msg {
			t.Errorf("expected %q, got %q", msg, string(replyPT))
		}
	}
}

func TestInteropAEADTestVectors(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	sid := []byte("interop-test")
	plaintext := []byte("hello cross-language")

	aead, err := crypto.NewAEAD(key)
	if err != nil {
		t.Fatal(err)
	}

	ct := crypto.EncryptAEAD(aead, sid, 42, plaintext)
	t.Logf("AEAD ciphertext (hex): %s", hex.EncodeToString(ct))

	pt, err := crypto.DecryptAEAD(aead, sid, 42, ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != string(plaintext) {
		t.Errorf("roundtrip failed")
	}

	newKey, err := crypto.Rekey(key)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Rekey result (hex): %s", hex.EncodeToString(newKey))
	if len(newKey) != 32 {
		t.Errorf("expected 32 byte key, got %d", len(newKey))
	}
}

func TestInteropNonceCompatibility(t *testing.T) {
	n0 := crypto.BuildNonce(0)
	expected := make([]byte, 12)
	if string(n0) != string(expected) {
		t.Errorf("nonce(0) mismatch: got %x", n0)
	}

	n1 := crypto.BuildNonce(1)
	if n1[4] != 1 {
		t.Errorf("nonce(1) byte[4] should be 1, got %d", n1[4])
	}
	for i := 5; i < 12; i++ {
		if n1[i] != 0 {
			t.Errorf("nonce(1) byte[%d] should be 0, got %d", i, n1[i])
		}
	}
}

func TestInteropMultipleClients(t *testing.T) {
	addr := freePort(t)
	startInteropServer(t, addr)

	for c := 0; c < 5; c++ {
		clientKey, _ := crypto.GenerateKeyPair()
		conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", addr))
		if err != nil {
			t.Fatal(err)
		}

		session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			t.Fatal(err)
		}

		msg := fmt.Sprintf("client-%d", c)
		ct, _ := session.Encrypt([]byte(msg))
		conn.WriteMessage(ct)

		rCT, _ := conn.ReadMessage()
		rPT, _ := session.Decrypt(rCT)
		if string(rPT) != msg {
			t.Errorf("client %d: expected %q, got %q", c, msg, string(rPT))
		}
		conn.Close()
	}
}
