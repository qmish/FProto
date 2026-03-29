package sdk

import (
	"crypto/rand"
	"fmt"

	"github.com/flynn/noise"
)

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

// KeyPair holds a static X25519 key pair.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// GenerateKeyPair generates a new X25519 key pair.
func GenerateKeyPair() (*KeyPair, error) {
	kp, err := noise.DH25519.GenerateKeypair(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate keypair: %w", err)
	}
	return &KeyPair{Private: kp.Private, Public: kp.Public}, nil
}

func clientHandshake(key *KeyPair, conn Transport) (*noiseSession, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:  cipherSuite,
		Pattern:       noise.HandshakeXX,
		Initiator:     true,
		StaticKeypair: noise.DHKey{Private: key.Private, Public: key.Public},
	})
	if err != nil {
		return nil, err
	}

	// -> e
	msg1, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, err
	}
	if err := conn.WriteMessage(msg1); err != nil {
		return nil, err
	}

	// <- e, ee, s, es
	msg2, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	_, _, _, err = hs.ReadMessage(nil, msg2)
	if err != nil {
		return nil, err
	}

	// -> s, se
	msg3, csSend, csRecv, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, err
	}
	if err := conn.WriteMessage(msg3); err != nil {
		return nil, err
	}

	if csSend == nil || csRecv == nil {
		return nil, fmt.Errorf("handshake incomplete")
	}

	return &noiseSession{send: csSend, recv: csRecv}, nil
}
