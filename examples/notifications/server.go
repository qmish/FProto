package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/qmish/FProto/proto-core/crypto"
	"github.com/qmish/FProto/proto-core/transport"
)

type subscriber struct {
	session  *crypto.NoiseSession
	conn     transport.Conn
	channels map[string]bool
	userID   string
}

type notifyHub struct {
	mu          sync.RWMutex
	subscribers map[string]*subscriber
}

func newHub() *notifyHub {
	return &notifyHub{subscribers: make(map[string]*subscriber)}
}

func (h *notifyHub) addSubscriber(id string, sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[id] = sub
}

func (h *notifyHub) removeSubscriber(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, id)
}

func (h *notifyHub) broadcast(n *Notification) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sent := 0
	msg, err := encodeNotification(n)
	if err != nil {
		return 0
	}

	for _, sub := range h.subscribers {
		if sub.channels[n.Channel] || sub.channels["*"] {
			ct, err := sub.session.Encrypt(msg)
			if err != nil {
				continue
			}
			if err := sub.conn.WriteMessage(ct); err != nil {
				continue
			}
			sent++
		}
	}
	return sent
}

func runServer(addr string) {
	serverKey, _ := crypto.GenerateKeyPair()
	hub := newHub()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := transport.UpgradeHTTP(w, r)
		if err != nil {
			log.Printf("upgrade: %v", err)
			return
		}
		handleClient(conn, serverKey, hub)
	})

	http.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var n Notification
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if n.TimestampMs == 0 {
			n.TimestampMs = nowMs()
		}
		sent := hub.broadcast(&n)
		fmt.Fprintf(w, `{"sent": %d}`, sent)
	})

	log.Printf("Notification WS server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleClient(conn transport.Conn, key *crypto.KeyPair, hub *notifyHub) {
	defer conn.Close()

	session, err := crypto.ServerHandshake(key, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Printf("handshake: %v", err)
		return
	}

	clientID := conn.RemoteAddr()
	sub := &subscriber{
		session:  session,
		conn:     conn,
		channels: map[string]bool{"*": true},
		userID:   clientID,
	}
	hub.addSubscriber(clientID, sub)
	defer hub.removeSubscriber(clientID)

	log.Printf("subscriber connected: %s", clientID)

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

		switch msgType {
		case MsgSubscribe:
			var s Subscription
			if err := json.Unmarshal(payload, &s); err != nil {
				continue
			}
			sub.channels = make(map[string]bool)
			for _, ch := range s.Channels {
				sub.channels[ch] = true
			}
			sub.userID = s.UserID
			log.Printf("subscriber %s subscribed to %v", clientID, s.Channels)

		case MsgAck:
			var ack NotifyAck
			if err := json.Unmarshal(payload, &ack); err != nil {
				continue
			}
			log.Printf("ack: %s -> %s", ack.NotificationID, ack.Status)
		}
	}
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", ":8081", "server address")
	user := flag.String("user", "user-1", "user ID")
	flag.Parse()

	if *mode == "server" {
		runServer(*addr)
	} else {
		runClient(*addr, *user)
	}
}

func runClient(addr, userID string) {
	clientKey, _ := crypto.GenerateKeyPair()

	url := fmt.Sprintf("ws://localhost%s/ws", addr)
	conn, err := transport.DialWS(url)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	session, err := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)
	if err != nil {
		log.Fatalf("handshake: %v", err)
	}

	sub := &Subscription{
		Channels: []string{"alerts", "updates"},
		UserID:   userID,
	}
	subMsg, _ := encodeSubscription(sub)
	ct, _ := session.Encrypt(subMsg)
	conn.WriteMessage(ct)
	log.Printf("subscribed to %v as %s", sub.Channels, userID)

	for {
		respCT, err := conn.ReadMessage()
		if err != nil {
			return
		}
		respPT, _ := session.Decrypt(respCT)
		msgType, payload, _ := decodeFrame(respPT)

		if msgType == MsgNotification {
			var n Notification
			json.Unmarshal(payload, &n)
			log.Printf("[%s] %s: %s", n.Channel, n.Title, n.Body)

			ack := &NotifyAck{NotificationID: n.ID, Status: "delivered"}
			ackMsg, _ := encodeAck(ack)
			ackCT, _ := session.Encrypt(ackMsg)
			conn.WriteMessage(ackCT)
		}
	}
}
