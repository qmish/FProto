package senderkeys

import (
	"bytes"
	"testing"
)

func TestGenerateSenderKey(t *testing.T) {
	sk, err := GenerateSenderKey(1)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if sk.KeyID != 1 {
		t.Fatal("key_id mismatch")
	}
	if len(sk.ChainKey) != 32 {
		t.Fatalf("chain_key len: %d", len(sk.ChainKey))
	}
	if len(sk.PublicKey) != 32 {
		t.Fatalf("public_key len: %d", len(sk.PublicKey))
	}
	if len(sk.SigningKey) != 64 {
		t.Fatalf("signing_key len: %d", len(sk.SigningKey))
	}
}

func TestGenerateUniqueness(t *testing.T) {
	sk1, _ := GenerateSenderKey(1)
	sk2, _ := GenerateSenderKey(2)
	if bytes.Equal(sk1.ChainKey, sk2.ChainKey) {
		t.Fatal("chain keys should be unique")
	}
	if bytes.Equal(sk1.PublicKey, sk2.PublicKey) {
		t.Fatal("public keys should be unique")
	}
}

func TestRatchet(t *testing.T) {
	chainKey := make([]byte, 32)
	chainKey[0] = 0x42

	nextChain, msgKey := Ratchet(chainKey)
	if len(nextChain) != 32 {
		t.Fatalf("next chain len: %d", len(nextChain))
	}
	if len(msgKey) != 32 {
		t.Fatalf("msg key len: %d", len(msgKey))
	}
	if bytes.Equal(nextChain, chainKey) {
		t.Fatal("next chain should differ from current")
	}
	if bytes.Equal(nextChain, msgKey) {
		t.Fatal("chain key and message key should differ")
	}
}

func TestRatchetDeterministic(t *testing.T) {
	chainKey := []byte("12345678901234567890123456789012")
	next1, mk1 := Ratchet(chainKey)
	next2, mk2 := Ratchet(chainKey)
	if !bytes.Equal(next1, next2) {
		t.Fatal("ratchet should be deterministic")
	}
	if !bytes.Equal(mk1, mk2) {
		t.Fatal("message key should be deterministic")
	}
}

func TestRatchetN(t *testing.T) {
	chainKey := make([]byte, 32)
	final, mks := RatchetN(chainKey, 10)
	if len(mks) != 10 {
		t.Fatalf("expected 10 message keys, got %d", len(mks))
	}
	if len(final) != 32 {
		t.Fatal("final chain key wrong length")
	}
	// Verify all message keys are different
	seen := map[string]bool{}
	for _, mk := range mks {
		s := string(mk)
		if seen[s] {
			t.Fatal("duplicate message key")
		}
		seen[s] = true
	}
}

func TestEncryptDecrypt(t *testing.T) {
	sk, err := GenerateSenderKey(1)
	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("hello group!")
	msg, nextChainSender, err := Encrypt(sk.ChainKey, sk.SigningKey, sk.KeyID, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if msg.KeyID != 1 {
		t.Fatal("key_id mismatch")
	}
	if len(msg.Ciphertext) == 0 {
		t.Fatal("empty ciphertext")
	}
	if len(msg.Nonce) != 12 {
		t.Fatalf("nonce len: %d", len(msg.Nonce))
	}
	if len(msg.Signature) != 64 {
		t.Fatalf("signature len: %d", len(msg.Signature))
	}

	decrypted, nextChainRecv, err := Decrypt(sk.ChainKey, sk.PublicKey, msg)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != "hello group!" {
		t.Fatalf("decrypted mismatch: %s", decrypted)
	}
	if !bytes.Equal(nextChainSender, nextChainRecv) {
		t.Fatal("sender and receiver should advance to same chain state")
	}
}

func TestEncryptDecryptMultipleMessages(t *testing.T) {
	sk, _ := GenerateSenderKey(1)
	chainSender := sk.ChainKey
	chainRecv := make([]byte, len(sk.ChainKey))
	copy(chainRecv, sk.ChainKey)

	for i := 0; i < 20; i++ {
		msg, nextS, err := Encrypt(chainSender, sk.SigningKey, sk.KeyID, []byte("msg"))
		if err != nil {
			t.Fatalf("encrypt %d: %v", i, err)
		}
		_, nextR, err := Decrypt(chainRecv, sk.PublicKey, msg)
		if err != nil {
			t.Fatalf("decrypt %d: %v", i, err)
		}
		chainSender = nextS
		chainRecv = nextR
	}
}

func TestDecryptInvalidSignature(t *testing.T) {
	sk, _ := GenerateSenderKey(1)
	msg, _, _ := Encrypt(sk.ChainKey, sk.SigningKey, sk.KeyID, []byte("test"))

	msg.Signature[0] ^= 0xff // corrupt signature
	_, _, err := Decrypt(sk.ChainKey, sk.PublicKey, msg)
	if err == nil {
		t.Fatal("expected signature verification failure")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	sk, _ := GenerateSenderKey(1)
	msg, _, _ := Encrypt(sk.ChainKey, sk.SigningKey, sk.KeyID, []byte("test"))

	msg.Ciphertext[0] ^= 0xff
	_, _, err := Decrypt(sk.ChainKey, sk.PublicKey, msg)
	if err == nil {
		t.Fatal("expected decryption failure for tampered ciphertext")
	}
}

func TestGroupRoundTrip(t *testing.T) {
	participants := 10
	senderKeys := make([]*SenderKey, participants)
	for i := 0; i < participants; i++ {
		sk, err := GenerateSenderKey(uint32(i + 1))
		if err != nil {
			t.Fatal(err)
		}
		senderKeys[i] = sk
	}

	sender := senderKeys[0]
	msg, _, err := Encrypt(sender.ChainKey, sender.SigningKey, sender.KeyID, []byte("group message"))
	if err != nil {
		t.Fatal(err)
	}

	// All participants (including sender) can decrypt using sender's public material
	for i := 0; i < participants; i++ {
		pt, _, err := Decrypt(sender.ChainKey, sender.PublicKey, msg)
		if err != nil {
			t.Fatalf("participant %d decrypt: %v", i, err)
		}
		if string(pt) != "group message" {
			t.Fatalf("participant %d plaintext mismatch", i)
		}
	}
}

func TestDistribution(t *testing.T) {
	sk, _ := GenerateSenderKey(42)
	groupID := []byte{0x11, 0x22}
	senderID := []byte{0xaa, 0xbb}

	dist := ToDistribution(groupID, senderID, sk)
	if dist.KeyId != 42 {
		t.Fatal("key_id mismatch")
	}
	if !bytes.Equal(dist.GroupId, groupID) {
		t.Fatal("group_id mismatch")
	}
	if !bytes.Equal(dist.SenderId, senderID) {
		t.Fatal("sender_id mismatch")
	}

	keyID, chainKey, pubKey := FromDistribution(dist)
	if keyID != 42 {
		t.Fatal("from distribution key_id")
	}
	if !bytes.Equal(chainKey, sk.ChainKey) {
		t.Fatal("chain key mismatch")
	}
	if !bytes.Equal(pubKey, sk.PublicKey) {
		t.Fatal("public key mismatch")
	}
}

func TestRotationMemberRemoved(t *testing.T) {
	if !NeedsFullRotation(RotationMemberRemoved) {
		t.Fatal("member removed should need full rotation")
	}
	if NeedsDistribution(RotationMemberRemoved) {
		t.Fatal("member removed should not need distribution")
	}
}

func TestRotationMemberAdded(t *testing.T) {
	if NeedsFullRotation(RotationMemberAdded) {
		t.Fatal("member added should not need full rotation")
	}
	if !NeedsDistribution(RotationMemberAdded) {
		t.Fatal("member added should need distribution")
	}
}

func TestForwardSecrecy(t *testing.T) {
	sk, _ := GenerateSenderKey(1)
	chain := sk.ChainKey

	msg1, next1, _ := Encrypt(chain, sk.SigningKey, sk.KeyID, []byte("msg1"))
	msg2, _, _ := Encrypt(next1, sk.SigningKey, sk.KeyID, []byte("msg2"))

	// Even knowing chain after msg2, cannot decrypt msg1 with that chain
	_, _, err := Decrypt(next1, sk.PublicKey, msg1)
	if err == nil {
		t.Fatal("should not decrypt msg1 with advanced chain (forward secrecy)")
	}

	// msg2 decrypts with correct chain
	pt, _, err := Decrypt(next1, sk.PublicKey, msg2)
	if err != nil {
		t.Fatalf("decrypt msg2: %v", err)
	}
	if string(pt) != "msg2" {
		t.Fatal("msg2 mismatch")
	}
}

func TestCacheKeyFormat(t *testing.T) {
	key := cacheKey([]byte{0xaa, 0xbb})
	expected := "group:aabb:sender_keys"
	if key != expected {
		t.Fatalf("expected %s, got %s", expected, key)
	}
}
