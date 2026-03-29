package senderkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// EncryptedMessage is the output of group message encryption.
type EncryptedMessage struct {
	KeyID      uint32
	Ciphertext []byte
	Nonce      []byte // 12 bytes
	Signature  []byte // Ed25519 signature over ciphertext
}

// Encrypt encrypts plaintext using the current message_key derived from chain ratchet,
// and signs the ciphertext with the sender's Ed25519 private key.
func Encrypt(chainKey []byte, signingKey ed25519.PrivateKey, keyID uint32, plaintext []byte) (*EncryptedMessage, []byte, error) {
	nextChain, messageKey := Ratchet(chainKey)

	aead, err := chacha20poly1305.New(messageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("chacha20poly1305: %w", err)
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	signature := ed25519.Sign(signingKey, ciphertext)

	return &EncryptedMessage{
		KeyID:      keyID,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Signature:  signature,
	}, nextChain, nil
}
