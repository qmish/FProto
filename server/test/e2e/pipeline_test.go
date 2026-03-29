package e2e

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/dispatcher"
	"github.com/qmish/FProto/server/internal/inbox"
	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func TestPipelineRouting_ChatToUser(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa, 0xbb},
				SenderId:    []byte{0xcc, 0xdd},
				Text: "hello world",
			},
		},
	}

	route, err := dispatcher.RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "user-aabb" {
		t.Fatalf("expected topic user-aabb, got %s", route.Topic)
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("empty payload")
	}

	var decoded pb.AppMessage
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	chat := decoded.GetChat()
	if chat == nil {
		t.Fatal("expected chat body")
	}
	if chat.Text != "hello world" {
		t.Fatal("text mismatch")
	}
}

func TestPipelineRouting_ChatToGroup(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				GroupId:    []byte{0x11, 0x22, 0x33},
				SenderId:   []byte{0xcc},
				Text: "group msg",
			},
		},
	}

	route, err := dispatcher.RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "group-112233" {
		t.Fatalf("expected group-112233, got %s", route.Topic)
	}
}

func TestPipelineRouting_Command(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Body: &pb.AppMessage_Command{
			Command: &pb.Command{
				Type: pb.Command_CREATE_GROUP,
			},
		},
	}
	route, _ := dispatcher.RouteMessage(msg)
	if route.Topic != kafkapkg.TopicCommands {
		t.Fatalf("expected commands, got %s", route.Topic)
	}
}

func TestPipelineInboxEntry(t *testing.T) {
	entry := inbox.Entry{
		UserID:         []byte{0xaa, 0xbb},
		MessageID:      []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		ConversationID: []byte{0xcc, 0xdd},
		SenderID:       []byte{0xee},
		Payload:        []byte("encrypted payload"),
	}

	if entry.Delivered {
		t.Fatal("should not be delivered")
	}
	if len(entry.Payload) != 17 {
		t.Fatalf("payload len: %d", len(entry.Payload))
	}
}

func TestPipelineFullFlow(t *testing.T) {
	senderID := []byte{0x01, 0x02}
	recipientID := []byte{0xaa, 0xbb}
	messageID := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	appMsg := &pb.AppMessage{
		MessageId: messageID,
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: recipientID,
				SenderId:    senderID,
				Text: "test message",
			},
		},
	}

	// 1. Route
	route, err := dispatcher.RouteMessage(appMsg)
	if err != nil {
		t.Fatal(err)
	}
	if route.Topic != "user-aabb" {
		t.Fatalf("wrong topic: %s", route.Topic)
	}

	// 2. Serialize for outbox/kafka
	payload, err := proto.Marshal(appMsg)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Deserialize (simulates consumer)
	var consumed pb.AppMessage
	if err := proto.Unmarshal(payload, &consumed); err != nil {
		t.Fatal(err)
	}

	// 4. Create inbox entry
	entry := inbox.Entry{
		UserID:         recipientID,
		MessageID:      consumed.MessageId,
		ConversationID: recipientID,
		SenderID:       consumed.GetChat().SenderId,
		Payload:        payload,
	}

	if string(entry.SenderID) != string(senderID) {
		t.Fatal("sender mismatch")
	}
	if string(entry.MessageID) != string(messageID) {
		t.Fatal("message_id mismatch")
	}
}
