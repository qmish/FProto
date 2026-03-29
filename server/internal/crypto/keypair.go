package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/flynn/noise"
)

// KeyPair содержит статическую ключевую пару X25519.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// GenerateKeyPair генерирует новую статическую ключевую пару X25519
// с использованием криптографически стойкого генератора.
func GenerateKeyPair() (*KeyPair, error) {
	dh := noise.DH25519
	kp, err := dh.GenerateKeypair(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate X25519 keypair: %w", err)
	}
	return &KeyPair{
		Private: kp.Private,
		Public:  kp.Public,
	}, nil
}
