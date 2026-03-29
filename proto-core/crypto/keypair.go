package crypto

import (
	"crypto/rand"
	"fmt"

	"github.com/flynn/noise"
)

// KeyPair holds an X25519 static keypair.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// GenerateKeyPair generates a new X25519 keypair using a cryptographically
// secure random source.
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
