package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestIoT_TelemetryRoundTrip(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	tlsConf := transport.GenerateSelfSignedTLS()

	listener, err := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	store := newStore()

	go func() {
		ctx := context.Background()
		qconn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			return
		}
		handleDevice(qconn, serverKey, store)
	}()

	clientKey, _ := crypto.GenerateKeyPair()
	clientTLS := &tls.Config{InsecureSkipVerify: true}
	ctx := context.Background()

	conn, err := transport.DialQUIC(ctx, listener.Addr().String(), clientTLS)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}

	tel := &Telemetry{
		DeviceID:    "test-device",
		SensorData:  map[string]float64{"temp": 25.0},
		TimestampMs: nowMs(),
	}
	msg, _ := encodeTelemetry(tel)
	ct, _ := session.Encrypt(msg)
	if err := conn.WriteMessage(ct); err != nil {
		t.Fatalf("write: %v", err)
	}

	respCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	respPT, err := session.Decrypt(respCT)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	msgType, payload, _ := decodeMessage(respPT)
	if msgType != MsgControlCommand {
		t.Fatalf("expected command, got 0x%02x", msgType)
	}

	var cmd ControlCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cmd.Action != "ack" {
		t.Fatalf("expected ack, got %s", cmd.Action)
	}
	if cmd.DeviceID != "test-device" {
		t.Fatalf("expected test-device, got %s", cmd.DeviceID)
	}
}

func TestEncodeDecode(t *testing.T) {
	tel := &Telemetry{
		DeviceID:    "dev-1",
		SensorData:  map[string]float64{"x": 1.0, "y": 2.0},
		TimestampMs: 12345,
	}
	encoded, err := encodeTelemetry(tel)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	msgType, payload, err := decodeMessage(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msgType != MsgTelemetry {
		t.Fatalf("expected telemetry type")
	}

	var decoded Telemetry
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.DeviceID != "dev-1" {
		t.Fatalf("device id mismatch")
	}
}
