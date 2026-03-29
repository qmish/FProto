package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

const (
	MsgGameAction  byte = 0x01
	MsgStateUpdate byte = 0x02
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GameAction struct {
	PlayerID    string `json:"player_id"`
	Action      string `json:"action"`
	Direction   Vec2   `json:"direction,omitempty"`
	TargetID    string `json:"target_id,omitempty"`
	SequenceNum uint64 `json:"seq"`
	TimestampMs int64  `json:"timestamp_ms"`
}

type PlayerState struct {
	PlayerID string  `json:"player_id"`
	Position Vec2    `json:"position"`
	Health   int     `json:"health"`
	Score    int     `json:"score"`
	Alive    bool    `json:"alive"`
}

type StateUpdate struct {
	Tick    uint64        `json:"tick"`
	Players []PlayerState `json:"players"`
}

func encodeAction(a *GameAction) ([]byte, error) {
	payload, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgGameAction
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func encodeStateUpdate(s *StateUpdate) ([]byte, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgStateUpdate
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
