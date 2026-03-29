package senderkeys

import (
	"crypto/ed25519"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// Decrypt verifies the Ed25519 signature and decrypts the ciphertext
// using the message_key derived from the sender's chain_key.
func Decrypt(chainKey []byte, publicKey ed25519.PublicKey, msg *EncryptedMessage) (plaintext, nextChain []byte, err error) {
	if !ed25519.Verify(publicKey, msg.Ciphertext, msg.Signature) {
		return nil, nil, fmt.Errorf("invalid signature")
	}

	nextChain, messageKey := Ratchet(chainKey)

	aead, err := chacha20poly1305.New(messageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("chacha20poly1305: %w", err)
	}

	plaintext, err = aead.Open(nil, msg.Nonce, msg.Ciphertext, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nextChain, nil
}
