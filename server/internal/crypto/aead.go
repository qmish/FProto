package crypto

import (
	"crypto/cipher"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// NewAEAD creates a ChaCha20-Poly1305 AEAD cipher from a 32-byte key.
func NewAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", chacha20poly1305.KeySize, len(key))
	}
	return chacha20poly1305.New(key)
}

// BuildNonce constructs a 96-bit nonce from a seq counter: 4 zero bytes + LE64(seq).
func BuildNonce(seq uint64) []byte {
	nonce := make([]byte, chacha20poly1305.NonceSize) // 12 bytes
	binary.LittleEndian.PutUint64(nonce[4:], seq)
	return nonce
}

// BuildAAD constructs additional authenticated data: session_id || LE64(seq).
func BuildAAD(sessionID []byte, seq uint64) []byte {
	aad := make([]byte, len(sessionID)+8)
	copy(aad, sessionID)
	binary.LittleEndian.PutUint64(aad[len(sessionID):], seq)
	return aad
}

// EncryptAEAD encrypts plaintext using ChaCha20-Poly1305 with seq-based nonce and AAD.
func EncryptAEAD(aead cipher.AEAD, sessionID []byte, seq uint64, plaintext []byte) []byte {
	nonce := BuildNonce(seq)
	aad := BuildAAD(sessionID, seq)
	return aead.Seal(nil, nonce, plaintext, aad)
}

// DecryptAEAD decrypts ciphertext using ChaCha20-Poly1305 with seq-based nonce and AAD.
func DecryptAEAD(aead cipher.AEAD, sessionID []byte, seq uint64, ciphertext []byte) ([]byte, error) {
	nonce := BuildNonce(seq)
	aad := BuildAAD(sessionID, seq)
	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("aead decrypt: %w", err)
	}
	return plaintext, nil
}
