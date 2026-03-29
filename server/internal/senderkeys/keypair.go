package senderkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
)

// SenderKey contains the cryptographic material for group E2E encryption.
type SenderKey struct {
	KeyID      uint32
	ChainKey   []byte // 32 bytes, used for HMAC ratchet
	SigningKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// GenerateSenderKey creates a new Sender Key with random chain_key and Ed25519 keypair.
func GenerateSenderKey(keyID uint32) (*SenderKey, error) {
	chainKey := make([]byte, 32)
	if _, err := rand.Read(chainKey); err != nil {
		return nil, fmt.Errorf("generate chain_key: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519: %w", err)
	}

	return &SenderKey{
		KeyID:      keyID,
		ChainKey:   chainKey,
		SigningKey:  priv,
		PublicKey:   pub,
	}, nil
}

// PublicMaterial returns the public parts needed for distribution.
func (sk *SenderKey) PublicMaterial() (chainKey []byte, publicKey ed25519.PublicKey) {
	return sk.ChainKey, sk.PublicKey
}
