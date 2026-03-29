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

func BenchmarkNoiseHandshake(b *testing.B) {
	serverKey, _ := crypto.GenerateKeyPair()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			return
		}
		defer ws.Close()

		_, err = crypto.ServerHandshake(serverKey, ws.ReadMessage, ws.WriteMessage)
		if err != nil {
			return
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ws, err := transport.DialWS(wsURL)
		if err != nil {
			b.Fatalf("dial: %v", err)
		}

		clientKey, _ := crypto.GenerateKeyPair()
		_, err = crypto.ClientHandshake(clientKey, ws.ReadMessage, ws.WriteMessage)
		if err != nil {
			b.Fatalf("handshake: %v", err)
		}
		ws.Close()
	}
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	ch := make(chan []byte, 10)
	clientSession, _ := benchHandshake(serverKey, clientKey, ch)

	payload := make([]byte, 256)
	for i := range payload {
		payload[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct, err := clientSession.Encrypt(payload)
		if err != nil {
			b.Fatalf("encrypt: %v", err)
		}
		_ = ct
	}
}

func benchHandshake(serverKey, clientKey *crypto.KeyPair, _ chan []byte) (*crypto.NoiseSession, *crypto.NoiseSession) {
	c2s := make(chan []byte, 3)
	s2c := make(chan []byte, 3)

	var cs, ss *crypto.NoiseSession
	done := make(chan struct{})

	go func() {
		ss, _ = crypto.ServerHandshake(serverKey,
			func() ([]byte, error) { return <-c2s, nil },
			func(b []byte) error { s2c <- b; return nil },
		)
		close(done)
	}()

	cs, _ = crypto.ClientHandshake(clientKey,
		func() ([]byte, error) { return <-s2c, nil },
		func(b []byte) error { c2s <- b; return nil },
	)
	<-done

	return cs, ss
}

func BenchmarkEchoRoundTrip(b *testing.B) {
	serverKey, _ := crypto.GenerateKeyPair()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			resp := fmt.Appendf(nil, "echo: %s", pt)
			enc, err := session.Encrypt(resp)
			if err != nil {
				return
			}
			if err := ws.WriteMessage(enc); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	ws, _ := transport.DialWS(wsURL)
	defer ws.Close()

	clientKey, _ := crypto.GenerateKeyPair()
	session, _ := crypto.ClientHandshake(clientKey, ws.ReadMessage, ws.WriteMessage)

	payload := []byte("benchmark-payload-data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct, _ := session.Encrypt(payload)
		_ = ws.WriteMessage(ct)
		respCT, _ := ws.ReadMessage()
		_, _ = session.Decrypt(respCT)
	}
}
