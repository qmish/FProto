package conformance

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestStandard_WebSocketRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	var received []byte
	var serverErr error
	var wg sync.WaitGroup
	wg.Add(1)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			serverErr = err
			return
		}
		defer conn.Close()

		if conn.Type() != transport.TypeWebSocket {
			serverErr = fmt.Errorf("expected WebSocket type")
			return
		}

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			serverErr = err
			return
		}

		ct, err := conn.ReadMessage()
		if err != nil {
			serverErr = err
			return
		}
		received, err = session.Decrypt(ct)
		if err != nil {
			serverErr = err
			return
		}

		reply, err := session.Encrypt([]byte("server-echo"))
		if err != nil {
			serverErr = err
			return
		}
		conn.WriteMessage(reply)
	})

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	conn, err := transport.DialWS(fmt.Sprintf("ws://%s/ws", ln.Addr().String()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	msg := []byte("client-hello")
	ct, _ := session.Encrypt(msg)
	conn.WriteMessage(ct)

	replyCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}
	reply, err := session.Decrypt(replyCT)
	if err != nil {
		t.Fatalf("decrypt reply: %v", err)
	}

	wg.Wait()
	if serverErr != nil {
		t.Fatalf("server: %v", serverErr)
	}

	if !bytes.Equal(received, msg) {
		t.Fatalf("server received %q, expected %q", received, msg)
	}
	if string(reply) != "server-echo" {
		t.Fatalf("client received %q, expected server-echo", reply)
	}
}

func TestStandard_QUICRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	var received []byte
	var serverErr error
	replySent := make(chan struct{})
	clientDone := make(chan struct{})

	go func() {
		ctx := context.Background()
		conn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		defer func() {
			<-clientDone
			conn.Close()
		}()

		if conn.Type() != transport.TypeQUIC {
			serverErr = fmt.Errorf("expected QUIC type")
			close(replySent)
			return
		}

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		ct, err := conn.ReadMessage()
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		received, err = session.Decrypt(ct)
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}

		reply, err := session.Encrypt([]byte("quic-echo"))
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		if err := conn.WriteMessage(reply); err != nil {
			serverErr = err
			close(replySent)
			return
		}
		close(replySent)
	}()

	ctx := context.Background()
	conn, err := transport.DialQUIC(ctx, listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() {
		close(clientDone)
		conn.Close()
	}()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	msg := []byte("quic-hello")
	ct, _ := session.Encrypt(msg)
	if err := conn.WriteMessage(ct); err != nil {
		t.Fatalf("write: %v", err)
	}

	replyCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	reply, err := session.Decrypt(replyCT)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	<-replySent
	if serverErr != nil {
		t.Fatalf("server: %v", serverErr)
	}

	if !bytes.Equal(received, msg) {
		t.Fatalf("received %q, expected %q", received, msg)
	}
	if string(reply) != "quic-echo" {
		t.Fatalf("reply %q, expected quic-echo", reply)
	}
}

func TestStandard_TransportTypeIdentifiers(t *testing.T) {
	types := map[transport.Type]string{
		transport.TypeWebSocket: "websocket",
		transport.TypeQUIC:      "quic",
		transport.TypeGRPC:      "grpc",
	}

	for typ, expected := range types {
		if string(typ) != expected {
			t.Fatalf("type %v should be %q", typ, expected)
		}
	}
}

func TestStandard_LargePayloadRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	payload := make([]byte, 32768)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	var received []byte
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, _ := transport.AcceptQUIC(context.Background(), listener)
		defer conn.Close()
		session, _ := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		ct, _ := conn.ReadMessage()
		received, _ = session.Decrypt(ct)
	}()

	conn, _ := transport.DialQUIC(context.Background(), listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	defer conn.Close()

	session, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	ct, _ := session.Encrypt(payload)
	conn.WriteMessage(ct)

	<-done

	if !bytes.Equal(received, payload) {
		t.Fatalf("large payload mismatch: sent %d bytes, received %d bytes", len(payload), len(received))
	}
}
