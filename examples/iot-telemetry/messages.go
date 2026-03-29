package main

import (
	"encoding/binary"
	"encoding/json"
	"time"
)

const (
	MsgTelemetry      byte = 0x01
	MsgControlCommand byte = 0x02
)

type Telemetry struct {
	DeviceID    string            `json:"device_id"`
	SensorData  map[string]float64 `json:"sensor_data"`
	TimestampMs int64             `json:"timestamp_ms"`
}

type ControlCommand struct {
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
	Params   map[string]string `json:"params,omitempty"`
}

func encodeTelemetry(t *Telemetry) ([]byte, error) {
	payload, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgTelemetry
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func encodeCommand(c *ControlCommand) ([]byte, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgControlCommand
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func decodeMessage(data []byte) (byte, []byte, error) {
	if len(data) < 5 {
		return 0, nil, nil
	}
	msgType := data[0]
	payloadLen := binary.BigEndian.Uint32(data[1:5])
	payload := data[5 : 5+payloadLen]
	return msgType, payload, nil
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
