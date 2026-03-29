package conformance

import (
	"bytes"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
)

func TestMinimal_KeyPairGeneration(t *testing.T) {
	kp1, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	kp2, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	if len(kp1.Public) != 32 {
		t.Fatalf("public key must be 32 bytes, got %d", len(kp1.Public))
	}
	if len(kp1.Private) != 32 {
		t.Fatalf("private key must be 32 bytes, got %d", len(kp1.Private))
	}
	if bytes.Equal(kp1.Public, kp2.Public) {
		t.Fatal("two generated key pairs must have different public keys")
	}
	if bytes.Equal(kp1.Private, kp2.Private) {
		t.Fatal("two generated key pairs must have different private keys")
	}
}

func TestMinimal_AEADEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	aead, err := crypto.NewAEAD(key)
	if err != nil {
		t.Fatalf("new aead: %v", err)
	}

	plaintext := []byte("hello conformance")
	sessionID := make([]byte, 16)
	var seq uint64 = 42

	ct := crypto.EncryptAEAD(aead, sessionID, seq, plaintext)

	pt, err := crypto.DecryptAEAD(aead, sessionID, seq, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(pt, plaintext) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestMinimal_AEADTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	aead, _ := crypto.NewAEAD(key)

	plaintext := []byte("tamper test")
	sessionID := make([]byte, 16)

	ct := crypto.EncryptAEAD(aead, sessionID, 1, plaintext)
	ct[len(ct)-1] ^= 0xFF

	_, err := crypto.DecryptAEAD(aead, sessionID, 1, ct)
	if err == nil {
		t.Fatal("decryption of tampered ciphertext must fail")
	}
}

func TestMinimal_AEADWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	aead1, _ := crypto.NewAEAD(key1)
	aead2, _ := crypto.NewAEAD(key2)

	plaintext := []byte("wrong key test")
	sessionID := make([]byte, 16)

	ct := crypto.EncryptAEAD(aead1, sessionID, 1, plaintext)

	_, err := crypto.DecryptAEAD(aead2, sessionID, 1, ct)
	if err == nil {
		t.Fatal("decryption with wrong key must fail")
	}
}

func TestMinimal_NonceUniqueness(t *testing.T) {
	n1 := crypto.BuildNonce(1)
	n2 := crypto.BuildNonce(2)
	n3 := crypto.BuildNonce(1)

	if bytes.Equal(n1, n2) {
		t.Fatal("different seq must produce different nonces")
	}
	if !bytes.Equal(n1, n3) {
		t.Fatal("same seq must produce same nonce")
	}
}

func TestMinimal_AADIncludesSessionAndSeq(t *testing.T) {
	sid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	aad1 := crypto.BuildAAD(sid, 1)
	aad2 := crypto.BuildAAD(sid, 2)

	if bytes.Equal(aad1, aad2) {
		t.Fatal("different seq must produce different AAD")
	}
	if len(aad1) != 24 {
		t.Fatalf("AAD should be sessionID(16) + seq(8) = 24 bytes, got %d", len(aad1))
	}
}

func TestMinimal_Rekey(t *testing.T) {
	original := make([]byte, 32)
	for i := range original {
		original[i] = byte(i)
	}

	rekeyed, err := crypto.Rekey(original)
	if err != nil {
		t.Fatalf("rekey: %v", err)
	}

	if len(rekeyed) != 32 {
		t.Fatalf("rekeyed key must be 32 bytes, got %d", len(rekeyed))
	}
	if bytes.Equal(original, rekeyed) {
		t.Fatal("rekeyed key must differ from original")
	}

	rekeyed2, _ := crypto.Rekey(original)
	if !bytes.Equal(rekeyed, rekeyed2) {
		t.Fatal("rekey must be deterministic")
	}
}

func TestMinimal_Zeroize(t *testing.T) {
	data := make([]byte, 32)
	for i := range data {
		data[i] = 0xFF
	}

	crypto.Zeroize(data)

	for i, b := range data {
		if b != 0 {
			t.Fatalf("byte %d not zeroed: 0x%02x", i, b)
		}
	}
}

func TestMinimal_RekeyThenEncrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 10)
	}

	newKey, _ := crypto.Rekey(key)

	aead1, _ := crypto.NewAEAD(newKey)
	aead2, _ := crypto.NewAEAD(key)

	plaintext := []byte("post-rekey message")
	sessionID := make([]byte, 16)

	ct := crypto.EncryptAEAD(aead1, sessionID, 100, plaintext)

	pt, err := crypto.DecryptAEAD(aead1, sessionID, 100, ct)
	if err != nil {
		t.Fatalf("decrypt with new key: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Fatal("round-trip failed after rekey")
	}

	_, err = crypto.DecryptAEAD(aead2, sessionID, 100, ct)
	if err == nil {
		t.Fatal("old key must not decrypt post-rekey ciphertext")
	}
}
