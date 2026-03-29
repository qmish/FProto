package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

const (
	MsgOrder         byte = 0x01
	MsgOrderResponse byte = 0x02
)

type Order struct {
	OrderID    string  `json:"order_id"`
	Instrument string  `json:"instrument"`
	Side       string  `json:"side"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	TimestampMs int64  `json:"timestamp_ms"`
}

type SignedOrder struct {
	Order     []byte `json:"order"`
	Signature []byte `json:"signature"`
	PublicKey []byte `json:"public_key"`
}

type OrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SigningKey struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

func GenerateSigningKey() (*SigningKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519 keygen: %w", err)
	}
	return &SigningKey{Public: pub, Private: priv}, nil
}

func signOrder(order *Order, key *SigningKey) ([]byte, error) {
	orderBytes, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(key.Private, orderBytes)

	signed := SignedOrder{
		Order:     orderBytes,
		Signature: sig,
		PublicKey: key.Public,
	}
	payload, err := json.Marshal(signed)
	if err != nil {
		return nil, err
	}

	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgOrder
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func verifyOrder(signedBytes []byte) (*Order, error) {
	var signed SignedOrder
	if err := json.Unmarshal(signedBytes, &signed); err != nil {
		return nil, fmt.Errorf("unmarshal signed: %w", err)
	}

	if !ed25519.Verify(signed.PublicKey, signed.Order, signed.Signature) {
		return nil, fmt.Errorf("invalid Ed25519 signature")
	}

	var order Order
	if err := json.Unmarshal(signed.Order, &order); err != nil {
		return nil, fmt.Errorf("unmarshal order: %w", err)
	}
	return &order, nil
}

func encodeResponse(resp *OrderResponse) ([]byte, error) {
	payload, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgOrderResponse
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func decodeFrame(data []byte) (byte, []byte, error) {
	if len(data) < 5 {
		return 0, nil, fmt.Errorf("frame too short")
	}
	msgType := data[0]
	payloadLen := binary.BigEndian.Uint32(data[1:5])
	if int(payloadLen) > len(data)-5 {
		return 0, nil, fmt.Errorf("payload overflow")
	}
	return msgType, data[5 : 5+payloadLen], nil
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
