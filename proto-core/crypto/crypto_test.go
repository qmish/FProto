package crypto

import (
	"bytes"
	"sync"
	"testing"
)

func performHandshake(t *testing.T) (*NoiseSession, *NoiseSession) {
	t.Helper()

	serverKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}

	clientKey, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	clientToServer := make(chan []byte, 3)
	serverToClient := make(chan []byte, 3)

	var serverSession *NoiseSession
	var clientSession *NoiseSession
	var serverErr, clientErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		serverSession, serverErr = ServerHandshake(serverKey,
			func() ([]byte, error) { return <-clientToServer, nil },
			func(b []byte) error { serverToClient <- b; return nil },
		)
	}()

	go func() {
		defer wg.Done()
		clientSession, clientErr = ClientHandshake(clientKey,
			func() ([]byte, error) { return <-serverToClient, nil },
			func(b []byte) error { clientToServer <- b; return nil },
		)
	}()

	wg.Wait()

	if serverErr != nil {
		t.Fatalf("server handshake: %v", serverErr)
	}
	if clientErr != nil {
		t.Fatalf("client handshake: %v", clientErr)
	}

	return clientSession, serverSession
}

func TestHandshakeSuccess(t *testing.T) {
	client, server := performHandshake(t)

	if client.Send == nil || client.Recv == nil {
		t.Fatal("client session has nil cipher state")
	}
	if server.Send == nil || server.Recv == nil {
		t.Fatal("server session has nil cipher state")
	}
}

func TestEncryptDecryptClientToServer(t *testing.T) {
	client, server := performHandshake(t)

	plaintext := []byte("Hello from client!")
	ciphertext, err := client.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := server.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch:\n  expected: %q\n  got:      %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptServerToClient(t *testing.T) {
	client, server := performHandshake(t)

	plaintext := []byte("Hello from server!")
	ciphertext, err := server.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := client.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch")
	}
}

func TestBidirectionalCommunication(t *testing.T) {
	client, server := performHandshake(t)

	messages := []struct {
		from string
		text string
	}{
		{"client", "Message 1 from client"},
		{"server", "Response 1 from server"},
		{"client", "Message 2 from client"},
		{"server", "Response 2 from server"},
		{"client", "Message 3 from client"},
	}

	for i, msg := range messages {
		var ct []byte
		var pt []byte
		var err error

		var encErr error
		if msg.from == "client" {
			ct, encErr = client.Encrypt([]byte(msg.text))
			if encErr != nil {
				t.Fatalf("msg %d encrypt: %v", i, encErr)
			}
			pt, err = server.Decrypt(ct)
		} else {
			ct, encErr = server.Encrypt([]byte(msg.text))
			if encErr != nil {
				t.Fatalf("msg %d encrypt: %v", i, encErr)
			}
			pt, err = client.Decrypt(ct)
		}

		if err != nil {
			t.Fatalf("msg %d decrypt: %v", i, err)
		}

		if string(pt) != msg.text {
			t.Fatalf("msg %d: expected %q, got %q", i, msg.text, string(pt))
		}
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	client, _ := performHandshake(t)
	_, server2 := performHandshake(t)

	plaintext := []byte("secret message")
	ciphertext, err := client.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = server2.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("expected decryption error with wrong key")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	client, server := performHandshake(t)

	plaintext := []byte("data integrity")
	ciphertext, err := client.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)/2] ^= 0xFF

	_, err = server.Decrypt(tampered)
	if err == nil {
		t.Fatal("expected decryption error with tampered ciphertext")
	}
}

func TestLargePayload(t *testing.T) {
	client, server := performHandshake(t)

	plaintext := make([]byte, 64*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	ciphertext, err := client.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt large payload: %v", err)
	}

	decrypted, err := server.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt large payload: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatal("large payload round-trip mismatch")
	}
}

func TestGenerateKeyPair(t *testing.T) {
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(kp1.Private) != 32 || len(kp1.Public) != 32 {
		t.Fatalf("unexpected key sizes: private=%d, public=%d", len(kp1.Private), len(kp1.Public))
	}

	if bytes.Equal(kp1.Private, kp2.Private) {
		t.Fatal("two generated key pairs should not have identical private keys")
	}

	if bytes.Equal(kp1.Public, kp2.Public) {
		t.Fatal("two generated key pairs should not have identical public keys")
	}
}
