package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"testing"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

func TestGame_MoveAndStateUpdate(t *testing.T) {
	serverKey, _ := crypto.GenerateKeyPair()
	tlsConf := transport.GenerateSelfSignedTLS()

	listener, err := transport.ListenQUIC("127.0.0.1:0", tlsConf)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	world := newWorld()

	go func() {
		ctx := context.Background()
		qconn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			return
		}
		handlePlayer(qconn, serverKey, world)
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

	action := &GameAction{
		Action:      "move",
		Direction:   Vec2{X: 1, Y: 0},
		SequenceNum: 1,
		TimestampMs: nowMs(),
	}
	msg, _ := encodeAction(action)
	ct, _ := session.Encrypt(msg)
	if err := conn.WriteMessage(ct); err != nil {
		t.Fatalf("write: %v", err)
	}

	respCT, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	respPT, _ := session.Decrypt(respCT)
	msgType, payload, _ := decodeFrame(respPT)

	if msgType != MsgStateUpdate {
		t.Fatalf("expected state update, got 0x%02x", msgType)
	}

	var state StateUpdate
	if err := json.Unmarshal(payload, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if state.Tick == 0 {
		t.Fatal("tick should be > 0")
	}
	if len(state.Players) == 0 {
		t.Fatal("expected at least one player")
	}
}

func TestGame_WorldSpawnAndMove(t *testing.T) {
	world := newWorld()
	world.spawn("p1")

	world.applyAction(&GameAction{
		PlayerID:  "p1",
		Action:    "move",
		Direction: Vec2{X: 1, Y: 0},
	})

	snap := world.snapshot()
	if snap.Tick != 1 {
		t.Fatalf("expected tick 1, got %d", snap.Tick)
	}

	found := false
	for _, p := range snap.Players {
		if p.PlayerID == "p1" {
			found = true
			if p.Position.X != 5.0 {
				t.Fatalf("expected x=5, got %.1f", p.Position.X)
			}
		}
	}
	if !found {
		t.Fatal("player p1 not in snapshot")
	}
}

func TestGame_AttackAndDamage(t *testing.T) {
	world := newWorld()
	world.spawn("attacker")
	world.spawn("target")

	for i := 0; i < 10; i++ {
		world.applyAction(&GameAction{
			PlayerID: "attacker",
			Action:   "attack",
			TargetID: "target",
		})
	}

	snap := world.snapshot()
	for _, p := range snap.Players {
		if p.PlayerID == "target" {
			if p.Alive {
				t.Fatal("target should be dead")
			}
		}
		if p.PlayerID == "attacker" {
			if p.Score != 100 {
				t.Fatalf("expected score 100, got %d", p.Score)
			}
		}
	}
}
