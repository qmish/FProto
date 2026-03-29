package sync

import (
	"testing"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func TestExtractRecipientID_Chat(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa, 0xbb, 0xcc, 0xdd},
			},
		},
	}
	uid := extractRecipientID(msg, "user-aabbccdd")
	if len(uid) != 4 || uid[0] != 0xaa {
		t.Fatalf("expected recipient from chat, got %x", uid)
	}
}

func TestExtractRecipientID_FromTopic(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Ack{Ack: &pb.Ack{}},
	}
	uid := extractRecipientID(msg, "user-0a0b")
	if uid == nil {
		t.Fatal("expected recipient from topic")
	}
	if uid[0] != 0x0a || uid[1] != 0x0b {
		t.Fatalf("unexpected uid: %x", uid)
	}
}

func TestExtractRecipientID_NoRecipient(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Ping{Ping: &pb.Ping{}},
	}
	uid := extractRecipientID(msg, "events")
	if uid != nil {
		t.Fatalf("expected nil recipient, got %x", uid)
	}
}

func TestExtractSenderID(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				SenderId: []byte{1, 2, 3},
			},
		},
	}
	sid := extractSenderID(msg)
	if len(sid) != 3 {
		t.Fatalf("expected sender_id of len 3, got %d", len(sid))
	}
}

func TestExtractConversationID_Group(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				GroupId: []byte{0x11, 0x22},
			},
		},
	}
	cid := extractConversationID(msg, "group-1122")
	if len(cid) != 2 || cid[0] != 0x11 {
		t.Fatalf("expected group_id, got %x", cid)
	}
}

func TestExtractConversationID_Recipient(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa},
			},
		},
	}
	cid := extractConversationID(msg, "user-aa")
	if len(cid) != 1 || cid[0] != 0xaa {
		t.Fatalf("expected recipient_id as conversation, got %x", cid)
	}
}

func TestHandlerProcessesMessage(t *testing.T) {
	appMsg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa, 0xbb},
				SenderId:    []byte{0xcc, 0xdd},
			},
		},
	}

	payload, err := proto.Marshal(appMsg)
	if err != nil {
		t.Fatal(err)
	}

	kafkaMsg := &sarama.ConsumerMessage{
		Topic: "user-aabb",
		Value: payload,
	}

	_ = kafkaMsg // handler requires live Redis + PG, validated in integration tests
}

func TestHexDecode(t *testing.T) {
	cases := []struct {
		input    string
		expected []byte
	}{
		{"aabb", []byte{0xaa, 0xbb}},
		{"0102", []byte{0x01, 0x02}},
		{"ff", []byte{0xff}},
	}
	for _, tc := range cases {
		got, err := hexDecode(tc.input)
		if err != nil {
			t.Fatalf("hexDecode(%s): %v", tc.input, err)
		}
		if len(got) != len(tc.expected) {
			t.Fatalf("hexDecode(%s): len %d != %d", tc.input, len(got), len(tc.expected))
		}
		for i := range got {
			if got[i] != tc.expected[i] {
				t.Fatalf("hexDecode(%s)[%d]: %x != %x", tc.input, i, got[i], tc.expected[i])
			}
		}
	}
}
