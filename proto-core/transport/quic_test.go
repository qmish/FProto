package transport_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestQUIC_EchoRoundTrip(t *testing.T) {
	serverTLS := transport.GenerateSelfSignedTLS()

	listener, err := transport.ListenQUIC("127.0.0.1:0", serverTLS)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	serverKey, _ := crypto.GenerateKeyPair()

	go func() {
		ctx := context.Background()
		qconn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			return
		}
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
			if err := qconn.WriteMessage(enc); err != nil {
				return
			}
		}
	}()

	clientTLS := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"proto/1"},
	}

	ctx := context.Background()
	qconn, err := transport.DialQUIC(ctx, listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer qconn.Close()

	if qconn.Type() != transport.TypeQUIC {
		t.Fatalf("expected QUIC type, got %s", qconn.Type())
	}

	clientKey, _ := crypto.GenerateKeyPair()
	session, err := crypto.ClientHandshake(clientKey, qconn.ReadMessage, qconn.WriteMessage)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}

	msg := "Hello QUIC!"
	ct, _ := session.Encrypt([]byte(msg))
	if err := qconn.WriteMessage(ct); err != nil {
		t.Fatalf("write: %v", err)
	}

	respCT, err := qconn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	respPT, err := session.Decrypt(respCT)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	expected := "echo: " + msg
	if string(respPT) != expected {
		t.Fatalf("expected %q, got %q", expected, string(respPT))
	}
}
