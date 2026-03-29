package crypto

import (
	"fmt"

	"github.com/flynn/noise"
)

// CipherSuite: Noise_XX_25519_ChaChaPoly_BLAKE2s
var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

// NoiseSession wraps Send/Recv CipherStates after a completed handshake.
type NoiseSession struct {
	Send *noise.CipherState
	Recv *noise.CipherState
}

// Encrypt encrypts plaintext with the outgoing session key.
func (s *NoiseSession) Encrypt(plaintext []byte) ([]byte, error) {
	return s.Send.Encrypt(nil, nil, plaintext)
}

// Decrypt decrypts ciphertext with the incoming session key.
func (s *NoiseSession) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := s.Recv.Decrypt(nil, nil, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("noise decrypt: %w", err)
	}
	return plaintext, nil
}

// ServerHandshake performs the server side of Noise XX handshake.
func ServerHandshake(serverKey *KeyPair, read func() ([]byte, error), write func([]byte) error) (*NoiseSession, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:  cipherSuite,
		Pattern:      noise.HandshakeXX,
		Initiator:    false,
		StaticKeypair: noise.DHKey{Private: serverKey.Private, Public: serverKey.Public},
	})
	if err != nil {
		return nil, fmt.Errorf("new handshake state: %w", err)
	}

	msg1, err := read()
	if err != nil {
		return nil, fmt.Errorf("read init: %w", err)
	}
	_, _, _, err = hs.ReadMessage(nil, msg1)
	if err != nil {
		return nil, fmt.Errorf("process init: %w", err)
	}

	msg2, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	if err := write(msg2); err != nil {
		return nil, fmt.Errorf("send response: %w", err)
	}

	msg3, err := read()
	if err != nil {
		return nil, fmt.Errorf("read finish: %w", err)
	}
	_, csRecv, csSend, err := hs.ReadMessage(nil, msg3)
	if err != nil {
		return nil, fmt.Errorf("process finish: %w", err)
	}

	if csRecv == nil || csSend == nil {
		return nil, fmt.Errorf("handshake incomplete: cipher states are nil")
	}

	return &NoiseSession{Send: csSend, Recv: csRecv}, nil
}

// ClientHandshake performs the client side of Noise XX handshake.
func ClientHandshake(clientKey *KeyPair, read func() ([]byte, error), write func([]byte) error) (*NoiseSession, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:  cipherSuite,
		Pattern:      noise.HandshakeXX,
		Initiator:    true,
		StaticKeypair: noise.DHKey{Private: clientKey.Private, Public: clientKey.Public},
	})
	if err != nil {
		return nil, fmt.Errorf("new handshake state: %w", err)
	}

	msg1, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("write init: %w", err)
	}
	if err := write(msg1); err != nil {
		return nil, fmt.Errorf("send init: %w", err)
	}

	msg2, err := read()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	_, _, _, err = hs.ReadMessage(nil, msg2)
	if err != nil {
		return nil, fmt.Errorf("process response: %w", err)
	}

	msg3, csSend, csRecv, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("write finish: %w", err)
	}
	if err := write(msg3); err != nil {
		return nil, fmt.Errorf("send finish: %w", err)
	}

	if csSend == nil || csRecv == nil {
		return nil, fmt.Errorf("handshake incomplete: cipher states are nil")
	}

	return &NoiseSession{Send: csSend, Recv: csRecv}, nil
}
