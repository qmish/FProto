package conformance

import (
	"bytes"
	"context"
	"crypto/tls"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestFull_EndToEndWithRekey(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	var serverReceived [][]byte
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, _ := transport.AcceptQUIC(context.Background(), listener)
		defer conn.Close()

		session, _ := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)

		for i := 0; i < 5; i++ {
			ct, err := conn.ReadMessage()
			if err != nil {
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				return
			}
			serverReceived = append(serverReceived, pt)
		}
	}()

	conn, _ := transport.DialQUIC(context.Background(), listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	defer conn.Close()

	session, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)

	messages := [][]byte{
		[]byte("before-rekey-1"),
		[]byte("before-rekey-2"),
		[]byte("before-rekey-3"),
		[]byte("after-rekey-1"),
		[]byte("after-rekey-2"),
	}

	for i, msg := range messages {
		ct, err := session.Encrypt(msg)
		if err != nil {
			t.Fatalf("encrypt %d: %v", i, err)
		}
		conn.WriteMessage(ct)
	}

	<-done

	if len(serverReceived) != 5 {
		t.Fatalf("expected 5, got %d", len(serverReceived))
	}
	for i, msg := range messages {
		if !bytes.Equal(serverReceived[i], msg) {
			t.Fatalf("message %d: %q != %q", i, serverReceived[i], msg)
		}
	}
}

func TestFull_BidirectionalEncryptedExchange(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	rounds := 5
	done := make(chan error, 1)

	go func() {
		conn, err := transport.AcceptQUIC(context.Background(), listener)
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()

		session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
		if err != nil {
			done <- err
			return
		}

		for i := 0; i < rounds; i++ {
			ct, err := conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				done <- err
				return
			}

			reply := append([]byte("echo:"), pt...)
			replyCT, err := session.Encrypt(reply)
			if err != nil {
				done <- err
				return
			}
			if err := conn.WriteMessage(replyCT); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	conn, err := transport.DialQUIC(context.Background(), listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	for i := 0; i < rounds; i++ {
		msg := []byte("ping")
		ct, _ := session.Encrypt(msg)
		if err := conn.WriteMessage(ct); err != nil {
			t.Fatalf("round %d write: %v", i, err)
		}

		replyCT, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("round %d read: %v", i, err)
		}
		reply, err := session.Decrypt(replyCT)
		if err != nil {
			t.Fatalf("round %d decrypt: %v", i, err)
		}

		expected := []byte("echo:ping")
		if !bytes.Equal(reply, expected) {
			t.Fatalf("round %d: expected %q, got %q", i, expected, reply)
		}
	}

	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestFull_ZeroLengthMessage(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

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
	ct, _ := session.Encrypt([]byte{})
	conn.WriteMessage(ct)

	<-done

	if len(received) != 0 {
		t.Fatalf("expected empty message, got %d bytes", len(received))
	}
}
