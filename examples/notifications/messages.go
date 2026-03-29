package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

const (
	MsgNotification byte = 0x01
	MsgSubscribe    byte = 0x02
	MsgAck          byte = 0x03
)

type Notification struct {
	ID          string `json:"id"`
	Channel     string `json:"channel"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Priority    string `json:"priority"`
	TimestampMs int64  `json:"timestamp_ms"`
}

type Subscription struct {
	Channels []string `json:"channels"`
	UserID   string   `json:"user_id"`
}

type NotifyAck struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
}

func encodeNotification(n *Notification) ([]byte, error) {
	payload, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgNotification
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func encodeSubscription(s *Subscription) ([]byte, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgSubscribe
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(payload)))
	copy(msg[5:], payload)
	return msg, nil
}

func encodeAck(a *NotifyAck) ([]byte, error) {
	payload, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	msg := make([]byte, 1+4+len(payload))
	msg[0] = MsgAck
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
