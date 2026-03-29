package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"sync"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

type telemetryStore struct {
	mu      sync.Mutex
	records map[string][]Telemetry
}

func newStore() *telemetryStore {
	return &telemetryStore{records: make(map[string][]Telemetry)}
}

func (s *telemetryStore) add(t Telemetry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[t.DeviceID] = append(s.records[t.DeviceID], t)
	if len(s.records[t.DeviceID]) > 1000 {
		s.records[t.DeviceID] = s.records[t.DeviceID][len(s.records[t.DeviceID])-1000:]
	}
}

func (s *telemetryStore) count(deviceID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records[deviceID])
}

func runServer(quicAddr string) {
	serverKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	tlsConf := transport.GenerateSelfSignedTLS()
	listener, err := transport.ListenQUIC(quicAddr, tlsConf)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("IoT QUIC server on %s", quicAddr)

	store := newStore()
	ctx := context.Background()

	for {
		qconn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleDevice(qconn, serverKey, store)
	}
}

func handleDevice(conn transport.Conn, key *crypto.KeyPair, store *telemetryStore) {
	defer conn.Close()

	session, err := crypto.ServerHandshake(key, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Printf("handshake: %v", err)
		return
	}
	log.Printf("device connected: %s", conn.RemoteAddr())

	for {
		ct, err := conn.ReadMessage()
		if err != nil {
			return
		}
		pt, err := session.Decrypt(ct)
		if err != nil {
			log.Printf("decrypt: %v", err)
			return
		}

		msgType, payload, err := decodeMessage(pt)
		if err != nil {
			continue
		}

		switch msgType {
		case MsgTelemetry:
			var t Telemetry
			if err := json.Unmarshal(payload, &t); err != nil {
				continue
			}
			store.add(t)
			log.Printf("telemetry from %s: %d sensors, total=%d",
				t.DeviceID, len(t.SensorData), store.count(t.DeviceID))

			ack := &ControlCommand{
				DeviceID: t.DeviceID,
				Action:   "ack",
				Params:   map[string]string{"count": "1"},
			}
			ackBytes, _ := encodeCommand(ack)
			enc, _ := session.Encrypt(ackBytes)
			conn.WriteMessage(enc)

		default:
			log.Printf("unknown message type: 0x%02x", msgType)
		}
	}
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", ":4433", "QUIC address")
	deviceID := flag.String("device", "sensor-001", "device ID (client mode)")
	flag.Parse()

	if *mode == "server" {
		runServer(*addr)
	} else {
		runClient(*addr, *deviceID)
	}
}

func runClient(addr, deviceID string) {
	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("keygen: %v", err)
	}

	tlsConf := &tls.Config{InsecureSkipVerify: true}
	ctx := context.Background()

	conn, err := transport.DialQUIC(ctx, addr, tlsConf)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Fatalf("handshake: %v", err)
	}
	log.Printf("connected as %s", deviceID)

	for i := 0; i < 10; i++ {
		t := &Telemetry{
			DeviceID: deviceID,
			SensorData: map[string]float64{
				"temperature": 22.5 + float64(i)*0.1,
				"humidity":    55.0 - float64(i)*0.5,
				"pressure":    1013.25,
			},
			TimestampMs: nowMs(),
		}

		msg, _ := encodeTelemetry(t)
		ct, _ := session.Encrypt(msg)
		if err := conn.WriteMessage(ct); err != nil {
			log.Fatalf("write: %v", err)
		}

		respCT, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		respPT, _ := session.Decrypt(respCT)
		respType, respPayload, _ := decodeMessage(respPT)
		if respType == MsgControlCommand {
			var cmd ControlCommand
			json.Unmarshal(respPayload, &cmd)
			log.Printf("ack from server: %s %v", cmd.Action, cmd.Params)
		}
	}
	log.Println("telemetry session complete")
}
