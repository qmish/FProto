package conformance

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"sync"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

type groupMessage struct {
	GroupID  string `json:"group_id"`
	SenderID string `json:"sender_id"`
	Text     string `json:"text"`
}

func TestFull_GroupEncryptedRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	var serverReceived []byte
	var serverErr error
	replySent := make(chan struct{})
	clientDone := make(chan struct{})

	go func() {
		conn, err := transport.AcceptQUIC(context.Background(), listener)
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		defer func() {
			<-clientDone
			conn.Close()
		}()
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
		serverReceived, err = session.Decrypt(ct)
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}

		reply, err := session.Encrypt([]byte(`{"status":"delivered"}`))
		if err != nil {
			serverErr = err
			close(replySent)
			return
		}
		conn.WriteMessage(reply)
		close(replySent)
	}()

	clientKey, _ := crypto.GenerateKeyPair()
	conn, err := transport.DialQUIC(context.Background(), listener.Addr().String(),
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

	msg := groupMessage{
		GroupID:  "group-123",
		SenderID: "user-456",
		Text:     "hello group!",
	}
	payload, _ := json.Marshal(msg)
	ct, _ := session.Encrypt(payload)
	conn.WriteMessage(ct)

	replyCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}
	replyPT, err := session.Decrypt(replyCT)
	if err != nil {
		t.Fatalf("decrypt reply: %v", err)
	}

	<-replySent
	if serverErr != nil {
		t.Fatalf("server: %v", serverErr)
	}

	var decoded groupMessage
	if err := json.Unmarshal(serverReceived, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.GroupID != "group-123" {
		t.Fatalf("group_id mismatch")
	}
	if decoded.Text != "hello group!" {
		t.Fatalf("text mismatch")
	}
	if string(replyPT) != `{"status":"delivered"}` {
		t.Fatalf("reply mismatch: %s", replyPT)
	}
}

func TestFull_MultipleClientsToServer(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	numClients := 3
	var mu sync.Mutex
	received := make([]string, 0, numClients)
	allClientsDone := make(chan struct{})

	var serverWg sync.WaitGroup
	serverWg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func() {
			defer serverWg.Done()
			conn, err := transport.AcceptQUIC(context.Background(), listener)
			if err != nil {
				return
			}
			defer func() {
				<-allClientsDone
				conn.Close()
			}()

			session, err := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)
			if err != nil {
				return
			}
			ct, err := conn.ReadMessage()
			if err != nil {
				return
			}
			pt, err := session.Decrypt(ct)
			if err != nil {
				return
			}

			mu.Lock()
			received = append(received, string(pt))
			mu.Unlock()

			reply, _ := session.Encrypt([]byte("ack"))
			conn.WriteMessage(reply)
		}()
	}

	var clientWg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		clientWg.Add(1)
		go func(id int) {
			defer clientWg.Done()
			clientKey, _ := crypto.GenerateKeyPair()
			conn, err := transport.DialQUIC(context.Background(), listener.Addr().String(),
				&tls.Config{InsecureSkipVerify: true})
			if err != nil {
				t.Errorf("client %d dial: %v", id, err)
				return
			}
			defer conn.Close()

			session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
			if err != nil {
				t.Errorf("client %d handshake: %v", id, err)
				return
			}

			msg := []byte("hello from client")
			ct, _ := session.Encrypt(msg)
			conn.WriteMessage(ct)

			replyCT, err := conn.ReadMessage()
			if err != nil {
				t.Errorf("client %d read: %v", id, err)
				return
			}
			reply, err := session.Decrypt(replyCT)
			if err != nil {
				t.Errorf("client %d decrypt: %v", id, err)
				return
			}
			if string(reply) != "ack" {
				t.Errorf("client %d: expected ack, got %s", id, reply)
			}
		}(i)
	}

	clientWg.Wait()
	close(allClientsDone)
	serverWg.Wait()

	if len(received) != numClients {
		t.Fatalf("expected %d messages, got %d", numClients, len(received))
	}
}

func TestFull_SequentialMessages(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	clientKey, _ := crypto.GenerateKeyPair()

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, _ := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	defer listener.Close()

	numMessages := 20
	serverMessages := make([]string, 0, numMessages)
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, _ := transport.AcceptQUIC(context.Background(), listener)
		defer conn.Close()
		session, _ := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)

		for i := 0; i < numMessages; i++ {
			ct, err := conn.ReadMessage()
			if err != nil {
				return
			}
			pt, _ := session.Decrypt(ct)
			serverMessages = append(serverMessages, string(pt))
		}
	}()

	conn, _ := transport.DialQUIC(context.Background(), listener.Addr().String(),
		&tls.Config{InsecureSkipVerify: true})
	defer conn.Close()

	session, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)

	for i := 0; i < numMessages; i++ {
		msg := []byte("msg-" + string(rune('A'+i)))
		ct, _ := session.Encrypt(msg)
		conn.WriteMessage(ct)
	}

	<-done

	if len(serverMessages) != numMessages {
		t.Fatalf("expected %d messages, got %d", numMessages, len(serverMessages))
	}

	for i := 0; i < numMessages; i++ {
		expected := "msg-" + string(rune('A'+i))
		if serverMessages[i] != expected {
			t.Fatalf("message %d: expected %q, got %q", i, expected, serverMessages[i])
		}
	}
}
