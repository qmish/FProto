package crypto

import (
	"fmt"

	"github.com/flynn/noise"
)

// CipherSuite используемый в протоколе FProto:
// Noise_XX_25519_ChaChaPoly_BLAKE2s
var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

// NoiseSession оборачивает пару CipherState после завершения handshake.
// Send используется для шифрования исходящих, Recv -- для расшифровки входящих.
type NoiseSession struct {
	Send *noise.CipherState
	Recv *noise.CipherState
}

// Encrypt шифрует plaintext с использованием исходящего ключа сессии.
// Возвращает ciphertext (включая Poly1305-тег).
func (s *NoiseSession) Encrypt(plaintext []byte) ([]byte, error) {
	return s.Send.Encrypt(nil, nil, plaintext)
}

// Decrypt расшифровывает ciphertext с использованием входящего ключа сессии.
func (s *NoiseSession) Decrypt(ciphertext []byte) ([]byte, error) {
	plaintext, err := s.Recv.Decrypt(nil, nil, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("noise decrypt: %w", err)
	}
	return plaintext, nil
}

// ServerHandshake выполняет серверную сторону Noise XX handshake.
// Принимает статическую ключевую пару сервера и три сообщения handshake
// через функции read/write. Возвращает NoiseSession для шифрования данных.
func ServerHandshake(serverKey *KeyPair, read func() ([]byte, error), write func([]byte) error) (*NoiseSession, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   cipherSuite,
		Pattern:        noise.HandshakeXX,
		Initiator:      false,
		StaticKeypair:  noise.DHKey{Private: serverKey.Private, Public: serverKey.Public},
	})
	if err != nil {
		return nil, fmt.Errorf("new handshake state: %w", err)
	}

	// Шаг 1: <- e (читаем от клиента)
	msg1, err := read()
	if err != nil {
		return nil, fmt.Errorf("read init: %w", err)
	}
	_, _, _, err = hs.ReadMessage(nil, msg1)
	if err != nil {
		return nil, fmt.Errorf("process init: %w", err)
	}

	// Шаг 2: -> e, ee, s, es (отправляем клиенту)
	msg2, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("write response: %w", err)
	}
	if err := write(msg2); err != nil {
		return nil, fmt.Errorf("send response: %w", err)
	}

	// Шаг 3: <- s, se (читаем от клиента)
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

// ClientHandshake выполняет клиентскую сторону Noise XX handshake.
func ClientHandshake(clientKey *KeyPair, read func() ([]byte, error), write func([]byte) error) (*NoiseSession, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   cipherSuite,
		Pattern:        noise.HandshakeXX,
		Initiator:      true,
		StaticKeypair:  noise.DHKey{Private: clientKey.Private, Public: clientKey.Public},
	})
	if err != nil {
		return nil, fmt.Errorf("new handshake state: %w", err)
	}

	// Шаг 1: -> e (отправляем серверу)
	msg1, _, _, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("write init: %w", err)
	}
	if err := write(msg1); err != nil {
		return nil, fmt.Errorf("send init: %w", err)
	}

	// Шаг 2: <- e, ee, s, es (читаем от сервера)
	msg2, err := read()
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	_, _, _, err = hs.ReadMessage(nil, msg2)
	if err != nil {
		return nil, fmt.Errorf("process response: %w", err)
	}

	// Шаг 3: -> s, se (отправляем серверу)
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
