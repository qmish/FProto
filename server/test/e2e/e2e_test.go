package e2e_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qmish/FProto/server/internal/crypto"
	"github.com/qmish/FProto/server/internal/transport"
)

func startWSEchoServer(t *testing.T) (string, *crypto.KeyPair) {
	t.Helper()
	serverKey, _ := crypto.GenerateKeyPair()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		defer ws.Close()

		session, err := crypto.ServerHandshake(serverKey, ws.ReadMessage, ws.WriteMessage)
		if err != nil {
			return
		}

		for {
			ct, err := ws.ReadMessage()
			if err != nil {
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				return
			}
			resp := append([]byte("echo: "), pt...)
			enc, _ := session.Encrypt(resp)
			if ws.WriteMessage(enc) != nil {
				return
			}
		}
	})

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go http.Serve(ln, mux)
	t.Cleanup(func() { ln.Close() })

	return ln.Addr().String(), serverKey
}

func startQUICEchoServer(t *testing.T) (string, *crypto.KeyPair) {
	t.Helper()
	serverKey, _ := crypto.GenerateKeyPair()
	tlsConf := transport.GenerateSelfSignedTLS()

	listener, err := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatalf("quic listen: %v", err)
	}
	t.Cleanup(func() { listener.Close() })

	go func() {
		for {
			ctx := context.Background()
			qconn, err := transport.AcceptQUIC(ctx, listener)
			if err != nil {
				return
			}
			go func() {
				defer qconn.Close()
				session, err := crypto.ServerHandshake(serverKey, qconn.ReadMessage, qconn.WriteMessage)
				if err != nil {
					return
				}
				for {
					ct, err := qconn.ReadMessage()
					if err != nil {
						return
					}
					pt, err := session.Decrypt(ct)
					if err != nil {
						return
					}
					resp := append([]byte("echo: "), pt...)
					enc, _ := session.Encrypt(resp)
					if qconn.WriteMessage(enc) != nil {
						return
					}
				}
			}()
		}
	}()

	return listener.Addr().String(), serverKey
}

// --- E2E: Go SDK via WebSocket ---

func TestE2E_GoSDK_WebSocket(t *testing.T) {
	addr, _ := startWSEchoServer(t)
	wsURL := "ws://" + addr + "/ws"

	clientKey, _ := crypto.GenerateKeyPair()
	conn, err := transport.DialWS(wsURL)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("e2e-ws-%d", i)
		ct, _ := session.Encrypt([]byte(msg))
		if err := conn.WriteMessage(ct); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		respCT, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		pt, err := session.Decrypt(respCT)
		if err != nil {
			t.Fatalf("decrypt %d: %v", i, err)
		}
		if string(pt) != "echo: "+msg {
			t.Fatalf("msg %d: expected %q, got %q", i, "echo: "+msg, string(pt))
		}
	}
}

// --- E2E: Go SDK via QUIC ---

func TestE2E_GoSDK_QUIC(t *testing.T) {
	addr, _ := startQUICEchoServer(t)

	clientTLS := &tls.Config{InsecureSkipVerify: true}
	ctx := context.Background()
	conn, err := transport.DialQUIC(ctx, addr, clientTLS)
	if err != nil {
		t.Fatalf("dial quic: %v", err)
	}
	defer conn.Close()

	clientKey, _ := crypto.GenerateKeyPair()
	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("e2e-quic-%d", i)
		ct, _ := session.Encrypt([]byte(msg))
		if err := conn.WriteMessage(ct); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		respCT, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		pt, err := session.Decrypt(respCT)
		if err != nil {
			t.Fatalf("decrypt %d: %v", i, err)
		}
		if string(pt) != "echo: "+msg {
			t.Fatalf("msg %d: expected %q, got %q", i, "echo: "+msg, string(pt))
		}
	}
}

// --- E2E: Multiple concurrent clients (WS) ---

func TestE2E_ConcurrentClients_WS(t *testing.T) {
	addr, _ := startWSEchoServer(t)
	wsURL := "ws://" + addr + "/ws"

	const numClients = 5
	done := make(chan error, numClients)

	for c := 0; c < numClients; c++ {
		go func(clientIdx int) {
			clientKey, _ := crypto.GenerateKeyPair()
			conn, err := transport.DialWS(wsURL)
			if err != nil {
				done <- err
				return
			}
			defer conn.Close()

			session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
			if err != nil {
				done <- err
				return
			}

			for i := 0; i < 3; i++ {
				msg := fmt.Sprintf("client%d-msg%d", clientIdx, i)
				ct, _ := session.Encrypt([]byte(msg))
				if err := conn.WriteMessage(ct); err != nil {
					done <- err
					return
				}
				respCT, err := conn.ReadMessage()
				if err != nil {
					done <- err
					return
				}
				pt, err := session.Decrypt(respCT)
				if err != nil {
					done <- err
					return
				}
				if string(pt) != "echo: "+msg {
					done <- fmt.Errorf("expected %q, got %q", "echo: "+msg, string(pt))
					return
				}
			}
			done <- nil
		}(c)
	}

	for i := 0; i < numClients; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("client error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("timeout waiting for clients")
		}
	}
}

// --- E2E: Rekey test ---

func TestE2E_Rekey(t *testing.T) {
	key1 := make([]byte, 32)
	copy(key1, []byte("test-key-for-rekey-1-AAAAAAAAA!"))

	newKey, err := crypto.Rekey(key1)
	if err != nil {
		t.Fatalf("rekey: %v", err)
	}

	if len(newKey) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(newKey))
	}

	aead1, _ := crypto.NewAEAD(key1)
	aead2, _ := crypto.NewAEAD(newKey)

	sessionID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	plaintext := []byte("message after rekey")

	ct := crypto.EncryptAEAD(aead2, sessionID, 0, plaintext)

	_, err = crypto.DecryptAEAD(aead1, sessionID, 0, ct)
	if err == nil {
		t.Fatal("old key should not decrypt new key's ciphertext")
	}

	pt, err := crypto.DecryptAEAD(aead2, sessionID, 0, ct)
	if err != nil {
		t.Fatalf("new key decrypt: %v", err)
	}
	if string(pt) != string(plaintext) {
		t.Fatalf("mismatch")
	}
}

// --- E2E: Session preservation (transport type identification) ---

func TestE2E_TransportTypes(t *testing.T) {
	addr, _ := startWSEchoServer(t)
	wsURL := "ws://" + addr + "/ws"

	conn, _ := transport.DialWS(wsURL)
	defer conn.Close()

	if conn.Type() != transport.TypeWebSocket {
		t.Fatalf("expected websocket, got %s", conn.Type())
	}
	if !strings.Contains(conn.RemoteAddr(), "127.0.0.1") {
		t.Fatalf("expected localhost addr, got %s", conn.RemoteAddr())
	}
}

// --- E2E: AEAD seq-nonce correctness across multiple messages ---

func TestE2E_AEAD_SeqNonce(t *testing.T) {
	key := make([]byte, 32)
	copy(key, []byte("e2e-test-key-for-aead-nonce!!!!"))

	aead, _ := crypto.NewAEAD(key)
	sessionID := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	for seq := uint64(0); seq < 50; seq++ {
		msg := fmt.Sprintf("msg-seq-%d", seq)
		ct := crypto.EncryptAEAD(aead, sessionID, seq, []byte(msg))
		pt, err := crypto.DecryptAEAD(aead, sessionID, seq, ct)
		if err != nil {
			t.Fatalf("seq %d: %v", seq, err)
		}
		if string(pt) != msg {
			t.Fatalf("seq %d: mismatch", seq)
		}

		_, err = crypto.DecryptAEAD(aead, sessionID, seq+1, ct)
		if err == nil {
			t.Fatalf("seq %d: should fail with wrong seq", seq)
		}
	}
}
