package sdk_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/gorilla/websocket"
	sdk "github.com/qmish/FProto/sdk/go"
)

var (
	testCipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)
	testUpgrader    = websocket.Upgrader{
		ReadBufferSize:  65536,
		WriteBufferSize: 65536,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
)

func genKey() noise.DHKey {
	kp, _ := noise.DH25519.GenerateKeypair(rand.Reader)
	return kp
}

func startEchoServer(t *testing.T) *httptest.Server {
	t.Helper()
	serverStatic := genKey()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()

		hs, _ := noise.NewHandshakeState(noise.Config{
			CipherSuite:  testCipherSuite,
			Pattern:       noise.HandshakeXX,
			Initiator:     false,
			StaticKeypair: serverStatic,
		})

		// <- e
		_, msg1, _ := ws.ReadMessage()
		_, _, _, _ = hs.ReadMessage(nil, msg1)

		// -> e, ee, s, es
		msg2, _, _, _ := hs.WriteMessage(nil, nil)
		_ = ws.WriteMessage(websocket.BinaryMessage, msg2)

		// <- s, se
		_, msg3, _ := ws.ReadMessage()
		_, csRecv, csSend, _ := hs.ReadMessage(nil, msg3)

		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			pt, err := csRecv.Decrypt(nil, nil, data)
			if err != nil {
				return
			}
			resp := fmt.Appendf(nil, "echo: %s", pt)
			ct, _ := csSend.Encrypt(nil, nil, resp)
			if err := ws.WriteMessage(websocket.BinaryMessage, ct); err != nil {
				return
			}
		}
	}))
}

func TestSDK_WebSocketEcho(t *testing.T) {
	server := startEchoServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	clientKey, err := sdk.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	client := sdk.New(clientKey, sdk.Config{
		ServerAddr:    wsURL,
		TransportType: "websocket",
	})

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	if err := client.Send([]byte("Hello SDK!")); err != nil {
		t.Fatalf("send: %v", err)
	}

	resp, err := client.Receive()
	if err != nil {
		t.Fatalf("receive: %v", err)
	}

	if string(resp) != "echo: Hello SDK!" {
		t.Fatalf("unexpected: %q", string(resp))
	}
}

func TestSDK_MultipleMessages(t *testing.T) {
	server := startEchoServer(t)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	clientKey, _ := sdk.GenerateKeyPair()

	client := sdk.New(clientKey, sdk.Config{
		ServerAddr:    wsURL,
		TransportType: "websocket",
	})

	ctx := context.Background()
	_ = client.Connect(ctx)
	defer client.Close()

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("msg-%d", i)
		_ = client.Send([]byte(msg))
		resp, err := client.Receive()
		if err != nil {
			t.Fatalf("msg %d receive: %v", i, err)
		}
		if string(resp) != "echo: "+msg {
			t.Fatalf("msg %d: expected %q, got %q", i, "echo: "+msg, string(resp))
		}
	}
}

func TestSDK_ReconnectConfig(t *testing.T) {
	cfg := sdk.DefaultReconnectConfig()
	if cfg.InitialDelay != time.Second {
		t.Fatalf("expected 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Fatalf("expected 30s, got %v", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Fatalf("expected 2.0, got %f", cfg.Multiplier)
	}

	d := sdk.NextDelay(0, cfg)
	if d < 800*time.Millisecond || d > 1200*time.Millisecond {
		t.Fatalf("attempt 0: expected ~1s, got %v", d)
	}

	d = sdk.NextDelay(5, cfg)
	if d > 36*time.Second {
		t.Fatalf("attempt 5: expected <= 30s+jitter, got %v", d)
	}
}
