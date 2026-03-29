package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"math"
	"sync"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

type gameWorld struct {
	mu      sync.Mutex
	players map[string]*PlayerState
	tick    uint64
}

func newWorld() *gameWorld {
	return &gameWorld{players: make(map[string]*PlayerState)}
}

func (w *gameWorld) spawn(playerID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.players[playerID] = &PlayerState{
		PlayerID: playerID,
		Position: Vec2{X: 0, Y: 0},
		Health:   100,
		Score:    0,
		Alive:    true,
	}
}

func (w *gameWorld) applyAction(a *GameAction) {
	w.mu.Lock()
	defer w.mu.Unlock()

	p, ok := w.players[a.PlayerID]
	if !ok || !p.Alive {
		return
	}

	switch a.Action {
	case "move":
		speed := 5.0
		p.Position.X += a.Direction.X * speed
		p.Position.Y += a.Direction.Y * speed
	case "attack":
		if target, ok := w.players[a.TargetID]; ok && target.Alive {
			dx := target.Position.X - p.Position.X
			dy := target.Position.Y - p.Position.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 50.0 {
				target.Health -= 10
				if target.Health <= 0 {
					target.Alive = false
					p.Score += 100
				}
			}
		}
	}
}

func (w *gameWorld) snapshot() *StateUpdate {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tick++

	players := make([]PlayerState, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, *p)
	}
	return &StateUpdate{Tick: w.tick, Players: players}
}

func runServer(quicAddr string) {
	serverKey, _ := crypto.GenerateKeyPair()
	tlsConf := transport.GenerateSelfSignedTLS()

	listener, err := transport.ListenQUIC(quicAddr, tlsConf)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("Game QUIC server on %s", quicAddr)

	world := newWorld()
	ctx := context.Background()

	for {
		qconn, err := transport.AcceptQUIC(ctx, listener)
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handlePlayer(qconn, serverKey, world)
	}
}

func handlePlayer(conn transport.Conn, key *crypto.KeyPair, world *gameWorld) {
	defer conn.Close()

	session, err := crypto.ServerHandshake(key, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Printf("handshake: %v", err)
		return
	}

	playerID := conn.RemoteAddr()
	world.spawn(playerID)
	log.Printf("player joined: %s", playerID)

	for {
		ct, err := conn.ReadMessage()
		if err != nil {
			return
		}
		pt, err := session.Decrypt(ct)
		if err != nil {
			return
		}

		msgType, payload, err := decodeFrame(pt)
		if err != nil {
			continue
		}

		if msgType == MsgGameAction {
			var action GameAction
			if err := json.Unmarshal(payload, &action); err != nil {
				continue
			}
			action.PlayerID = playerID
			world.applyAction(&action)
		}

		state := world.snapshot()
		stateMsg, _ := encodeStateUpdate(state)
		enc, _ := session.Encrypt(stateMsg)
		conn.WriteMessage(enc)
	}
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", ":4434", "QUIC address")
	player := flag.String("player", "player-1", "player ID")
	flag.Parse()

	if *mode == "server" {
		runServer(*addr)
	} else {
		runClient(*addr, *player)
	}
}

func runClient(addr, playerID string) {
	clientKey, _ := crypto.GenerateKeyPair()
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
	log.Printf("joined as %s", playerID)

	actions := []GameAction{
		{PlayerID: playerID, Action: "move", Direction: Vec2{X: 1, Y: 0}, SequenceNum: 1, TimestampMs: nowMs()},
		{PlayerID: playerID, Action: "move", Direction: Vec2{X: 0, Y: 1}, SequenceNum: 2, TimestampMs: nowMs()},
		{PlayerID: playerID, Action: "move", Direction: Vec2{X: -1, Y: 0}, SequenceNum: 3, TimestampMs: nowMs()},
	}

	for _, a := range actions {
		msg, _ := encodeAction(&a)
		ct, _ := session.Encrypt(msg)
		if err := conn.WriteMessage(ct); err != nil {
			log.Fatalf("write: %v", err)
		}

		respCT, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		respPT, _ := session.Decrypt(respCT)
		respType, respPayload, _ := decodeFrame(respPT)
		if respType == MsgStateUpdate {
			var state StateUpdate
			json.Unmarshal(respPayload, &state)
			log.Printf("tick %d: %d players", state.Tick, len(state.Players))
		}
	}
}
