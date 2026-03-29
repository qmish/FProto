package transport_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func startTestServer(t *testing.T) (*httptest.Server, *crypto.KeyPair) {
	t.Helper()

	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer ws.Close()

		session, err := crypto.ServerHandshake(serverKey, ws.ReadMessage, ws.WriteMessage)
		if err != nil {
			t.Errorf("server handshake: %v", err)
			return
		}

		for {
			ciphertext, err := ws.ReadMessage()
			if err != nil {
				return
			}

			plaintext, err := session.Decrypt(ciphertext)
			if err != nil {
				t.Errorf("decrypt: %v", err)
				return
			}

			response := fmt.Appendf(nil, "echo: %s", plaintext)
			encrypted, err := session.Encrypt(response)
			if err != nil {
				t.Errorf("encrypt: %v", err)
				return
			}

			if err := ws.WriteMessage(encrypted); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	return server, serverKey
}

func TestEchoServerRoundTrip(t *testing.T) {
	server, _ := startTestServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, err := transport.DialWS(wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	session, err := crypto.ClientHandshake(clientKey, ws.ReadMessage, ws.WriteMessage)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}

	message := "Привет, FProto!"
	ciphertext, err := session.Encrypt([]byte(message))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if err := ws.WriteMessage(ciphertext); err != nil {
		t.Fatalf("write: %v", err)
	}

	responseCT, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	responsePT, err := session.Decrypt(responseCT)
	if err != nil {
		t.Fatalf("decrypt response: %v", err)
	}

	expected := "echo: " + message
	if string(responsePT) != expected {
		t.Fatalf("unexpected response:\n  expected: %q\n  got:      %q", expected, string(responsePT))
	}
}

func TestMultipleMessages(t *testing.T) {
	server, _ := startTestServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ws, err := transport.DialWS(wsURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	session, err := crypto.ClientHandshake(clientKey, ws.ReadMessage, ws.WriteMessage)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}

	messages := []string{
		"Сообщение 1",
		"Сообщение 2",
		"Сообщение 3",
		"Тест Unicode: 你好世界 🌍",
		"Длинное сообщение: " + strings.Repeat("абвгд", 1000),
	}

	for i, msg := range messages {
		ct, err := session.Encrypt([]byte(msg))
		if err != nil {
			t.Fatalf("msg %d encrypt: %v", i, err)
		}

		if err := ws.WriteMessage(ct); err != nil {
			t.Fatalf("msg %d write: %v", i, err)
		}

		respCT, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("msg %d read: %v", i, err)
		}

		respPT, err := session.Decrypt(respCT)
		if err != nil {
			t.Fatalf("msg %d decrypt: %v", i, err)
		}

		expected := "echo: " + msg
		if string(respPT) != expected {
			t.Fatalf("msg %d: expected %q, got %q", i, expected, string(respPT))
		}
	}
}

func TestMultipleClients(t *testing.T) {
	server, _ := startTestServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	for c := 0; c < 5; c++ {
		ws, err := transport.DialWS(wsURL)
		if err != nil {
			t.Fatalf("client %d dial: %v", c, err)
		}

		clientKey, err := crypto.GenerateKeyPair()
		if err != nil {
			t.Fatalf("client %d key: %v", c, err)
		}

		session, err := crypto.ClientHandshake(clientKey, ws.ReadMessage, ws.WriteMessage)
		if err != nil {
			t.Fatalf("client %d handshake: %v", c, err)
		}

		msg := fmt.Sprintf("клиент %d", c)
		ct, err := session.Encrypt([]byte(msg))
		if err != nil {
			t.Fatalf("client %d encrypt: %v", c, err)
		}

		if err := ws.WriteMessage(ct); err != nil {
			t.Fatalf("client %d write: %v", c, err)
		}

		respCT, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("client %d read: %v", c, err)
		}

		respPT, err := session.Decrypt(respCT)
		if err != nil {
			t.Fatalf("client %d decrypt: %v", c, err)
		}

		expected := "echo: " + msg
		if string(respPT) != expected {
			t.Fatalf("client %d: expected %q, got %q", c, expected, string(respPT))
		}

		ws.Close()
	}
}
