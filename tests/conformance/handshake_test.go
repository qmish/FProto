package conformance

import (
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

func TestMinimal_HandshakeWebSocket(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	mux := http.NewServeMux()
	var serverSession *crypto.NoiseSession
	var serverErr error
	var wg sync.WaitGroup
	wg.Add(1)

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			serverErr = err
			return
		}
		defer conn.Close()
		serverSession, serverErr = crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
	})

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	defer ln.Close()
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	url := fmt.Sprintf("ws://%s/ws", ln.Addr().String())
	conn, err := transport.DialWS(url)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	clientSession, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	wg.Wait()

	if serverErr != nil {
		t.Fatalf("server handshake: %v", serverErr)
	}
	if clientSession == nil || serverSession == nil {
		t.Fatal("sessions must not be nil")
	}
	if clientSession.Send == nil || clientSession.Recv == nil {
		t.Fatal("client cipher states must not be nil")
	}
}

func TestMinimal_HandshakeQUIC(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, err := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	var serverSession *crypto.NoiseSession
	var serverErr error
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ctx := context.Background()
		conn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			serverErr = err
			return
		}
		defer conn.Close()
		serverSession, serverErr = crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
	}()

	clientTLS := &tls.Config{InsecureSkipVerify: true}
	ctx := context.Background()
	conn, err := transport.DialQUIC(ctx, listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	clientSession, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	wg.Wait()

	if serverErr != nil {
		t.Fatalf("server handshake: %v", serverErr)
	}
	if clientSession == nil || serverSession == nil {
		t.Fatal("sessions must not be nil")
	}
}

func TestMinimal_HandshakeProducesDistinctKeys(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	var srvSession *crypto.NoiseSession
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		conn, _ := transport.AcceptQUIC(context.Background(), listener)
		defer conn.Close()
		srvSession, _ = crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
	}()

	conn, _ := transport.DialQUIC(context.Background(), listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	defer conn.Close()

	cliSession, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	wg.Wait()

	if cliSession == nil || srvSession == nil {
		t.Fatal("sessions nil")
	}

	msg := []byte("conformance-test")
	ct, err := cliSession.Encrypt(msg)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := srvSession.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != string(msg) {
		t.Fatalf("round-trip mismatch: %q vs %q", pt, msg)
	}
}
