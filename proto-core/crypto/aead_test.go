package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func randomKey() []byte {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return key
}

func TestAEAD_EncryptDecrypt(t *testing.T) {
	key := randomKey()
	aead, err := NewAEAD(key)
	if err != nil {
		t.Fatalf("new aead: %v", err)
	}

	sessionID := make([]byte, 16)
	_, _ = rand.Read(sessionID)

	plaintext := []byte("test message for aead")
	seq := uint64(42)

	ciphertext := EncryptAEAD(aead, sessionID, seq, plaintext)

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := DecryptAEAD(aead, sessionID, seq, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("plaintext mismatch")
	}
}

func TestAEAD_WrongSeqFails(t *testing.T) {
	key := randomKey()
	aead, _ := NewAEAD(key)

	sessionID := make([]byte, 16)
	plaintext := []byte("test")

	ciphertext := EncryptAEAD(aead, sessionID, 1, plaintext)

	_, err := DecryptAEAD(aead, sessionID, 2, ciphertext)
	if err == nil {
		t.Fatal("expected decryption error with wrong seq")
	}
}

func TestAEAD_WrongSessionIDFails(t *testing.T) {
	key := randomKey()
	aead, _ := NewAEAD(key)

	sid1 := make([]byte, 16)
	sid2 := make([]byte, 16)
	_, _ = rand.Read(sid1)
	_, _ = rand.Read(sid2)

	plaintext := []byte("test")
	ciphertext := EncryptAEAD(aead, sid1, 1, plaintext)

	_, err := DecryptAEAD(aead, sid2, 1, ciphertext)
	if err == nil {
		t.Fatal("expected decryption error with wrong session ID")
	}
}

func TestAEAD_TamperedCiphertextFails(t *testing.T) {
	key := randomKey()
	aead, _ := NewAEAD(key)

	sessionID := make([]byte, 16)
	plaintext := []byte("test integrity")
	ciphertext := EncryptAEAD(aead, sessionID, 1, plaintext)

	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[0] ^= 0xFF

	_, err := DecryptAEAD(aead, sessionID, 1, tampered)
	if err == nil {
		t.Fatal("expected error with tampered ciphertext")
	}
}

func TestBuildNonce(t *testing.T) {
	nonce := BuildNonce(0)
	if len(nonce) != 12 {
		t.Fatalf("nonce length: expected 12, got %d", len(nonce))
	}
	for _, b := range nonce {
		if b != 0 {
			t.Fatal("nonce for seq=0 should be all zeros")
		}
	}

	nonce = BuildNonce(1)
	if nonce[4] != 1 {
		t.Fatalf("expected nonce[4]=1 for seq=1, got %d", nonce[4])
	}
	for i := 0; i < 4; i++ {
		if nonce[i] != 0 {
			t.Fatalf("first 4 bytes should be zero, byte %d = %d", i, nonce[i])
		}
	}
}

func TestBuildAAD(t *testing.T) {
	sessionID := []byte{1, 2, 3, 4}
	aad := BuildAAD(sessionID, 42)

	if len(aad) != 12 {
		t.Fatalf("AAD length: expected 12, got %d", len(aad))
	}
	if !bytes.Equal(aad[:4], sessionID) {
		t.Fatal("AAD should start with session_id")
	}
}

func TestAEAD_InvalidKeySize(t *testing.T) {
	_, err := NewAEAD([]byte("short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestAEAD_SequentialSeqNumbers(t *testing.T) {
	key := randomKey()
	aead, _ := NewAEAD(key)
	sessionID := make([]byte, 16)

	for seq := uint64(0); seq < 100; seq++ {
		plaintext := []byte("msg")
		ct := EncryptAEAD(aead, sessionID, seq, plaintext)
		pt, err := DecryptAEAD(aead, sessionID, seq, ct)
		if err != nil {
			t.Fatalf("seq %d: decrypt error: %v", seq, err)
		}
		if !bytes.Equal(pt, plaintext) {
			t.Fatalf("seq %d: mismatch", seq)
		}
	}
}
