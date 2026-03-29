package protocol_test

import (
	"crypto/rand"
	"testing"
	"time"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newUUID() []byte {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return b
}

func TestFrameRoundTrip(t *testing.T) {
	original := &pb.Frame{
		SessionId:        newUUID(),
		Seq:              42,
		Ack:              41,
		EncryptedPayload: []byte("encrypted-data-here"),
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal Frame: %v", err)
	}

	decoded := &pb.Frame{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal Frame: %v", err)
	}

	if !proto.Equal(original, decoded) {
		t.Fatalf("Frame round-trip mismatch:\n  original: %v\n  decoded:  %v", original, decoded)
	}
}

func TestHandshakeMessageRoundTrip(t *testing.T) {
	types := []pb.HandshakeMessage_HandshakeType{
		pb.HandshakeMessage_INIT,
		pb.HandshakeMessage_RESPONSE,
		pb.HandshakeMessage_FINISH,
	}

	for _, ht := range types {
		original := &pb.HandshakeMessage{
			Type:    ht,
			Payload: []byte("handshake-payload"),
		}

		data, err := proto.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal HandshakeMessage(%v): %v", ht, err)
		}

		decoded := &pb.HandshakeMessage{}
		if err := proto.Unmarshal(data, decoded); err != nil {
			t.Fatalf("Unmarshal HandshakeMessage(%v): %v", ht, err)
		}

		if !proto.Equal(original, decoded) {
			t.Fatalf("HandshakeMessage(%v) round-trip mismatch", ht)
		}
	}
}

func TestAppMessageWithChat(t *testing.T) {
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.New(time.Now()),
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				ConversationId: newUUID(),
				SenderId:       newUUID(),
				RecipientId:    newUUID(),
				Action:         pb.ChatMessage_SEND,
				Text:           "Привет, мир!",
				Attachments: []*pb.Attachment{
					{
						MimeType:  "image/jpeg",
						Url:       "https://cdn.example.com/photo.jpg",
						SizeBytes: 1024000,
						Thumbnail: []byte{0xFF, 0xD8, 0xFF},
					},
				},
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal AppMessage(Chat): %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal AppMessage(Chat): %v", err)
	}

	if !proto.Equal(original, decoded) {
		t.Fatal("AppMessage(Chat) round-trip mismatch")
	}

	chat := decoded.GetChat()
	if chat == nil {
		t.Fatal("expected Chat body, got nil")
	}
	if chat.Text != "Привет, мир!" {
		t.Fatalf("unexpected text: %q", chat.Text)
	}
	if chat.Action != pb.ChatMessage_SEND {
		t.Fatalf("unexpected action: %v", chat.Action)
	}
	if len(chat.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(chat.Attachments))
	}
}

func TestAppMessageWithCommand(t *testing.T) {
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_Command{
			Command: &pb.Command{
				Type:     pb.Command_CREATE_GROUP,
				TargetId: newUUID(),
				Payload:  []byte(`{"name":"Test Group"}`),
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal AppMessage(Command): %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal AppMessage(Command): %v", err)
	}

	cmd := decoded.GetCommand()
	if cmd == nil {
		t.Fatal("expected Command body, got nil")
	}
	if cmd.Type != pb.Command_CREATE_GROUP {
		t.Fatalf("unexpected command type: %v", cmd.Type)
	}
}

func TestAppMessageWithMedia(t *testing.T) {
	chunkData := make([]byte, 64*1024)
	_, _ = rand.Read(chunkData)

	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_Media{
			Media: &pb.MediaChunk{
				StreamId:    newUUID(),
				ChunkIndex:  0,
				TotalChunks: 4,
				Data:        chunkData,
				Hash:        newUUID(),
				MimeType:    "image/png",
				TotalSize:   262144,
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal AppMessage(Media): %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal AppMessage(Media): %v", err)
	}

	media := decoded.GetMedia()
	if media == nil {
		t.Fatal("expected Media body, got nil")
	}
	if media.TotalChunks != 4 {
		t.Fatalf("unexpected total_chunks: %d", media.TotalChunks)
	}
	if len(media.Data) != 64*1024 {
		t.Fatalf("unexpected chunk data size: %d", len(media.Data))
	}
}

func TestAppMessageWithAck(t *testing.T) {
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_Ack{
			Ack: &pb.Ack{
				AckMessageId: newUUID(),
				Status:       pb.Ack_DELIVERED,
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal AppMessage(Ack): %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal AppMessage(Ack): %v", err)
	}

	ack := decoded.GetAck()
	if ack == nil {
		t.Fatal("expected Ack body, got nil")
	}
	if ack.Status != pb.Ack_DELIVERED {
		t.Fatalf("unexpected ack status: %v", ack.Status)
	}
}

func TestAppMessageWithPing(t *testing.T) {
	now := uint64(time.Now().UnixMilli())
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_Ping{
			Ping: &pb.Ping{
				TimestampMs: now,
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	ping := decoded.GetPing()
	if ping == nil {
		t.Fatal("expected Ping body")
	}
	if ping.TimestampMs != now {
		t.Fatalf("timestamp mismatch: %d != %d", ping.TimestampMs, now)
	}
}

func TestAppMessageWithSyncRequest(t *testing.T) {
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_SyncRequest{
			SyncRequest: &pb.SyncRequest{
				LastSeenMessageId: newUUID(),
				MaxMessages:       100,
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	sr := decoded.GetSyncRequest()
	if sr == nil {
		t.Fatal("expected SyncRequest body")
	}
	if sr.MaxMessages != 100 {
		t.Fatalf("unexpected max_messages: %d", sr.MaxMessages)
	}
}

func TestAppMessageWithSenderKeyDistribution(t *testing.T) {
	original := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_SenderKey{
			SenderKey: &pb.SenderKeyDistribution{
				GroupId:    newUUID(),
				SenderId:   newUUID(),
				KeyId:      1,
				ChainKey:   make([]byte, 32),
				SigningKey:  make([]byte, 32),
			},
		},
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := &pb.AppMessage{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	sk := decoded.GetSenderKey()
	if sk == nil {
		t.Fatal("expected SenderKeyDistribution body")
	}
	if sk.KeyId != 1 {
		t.Fatalf("unexpected key_id: %d", sk.KeyId)
	}
}

func TestAppMessageOneofExclusivity(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: newUUID(),
		Timestamp: timestamppb.Now(),
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{Text: "hello"},
		},
	}

	if msg.GetChat() == nil {
		t.Fatal("expected Chat")
	}
	if msg.GetCommand() != nil {
		t.Fatal("Command should be nil when Chat is set")
	}
	if msg.GetAck() != nil {
		t.Fatal("Ack should be nil when Chat is set")
	}
	if msg.GetMedia() != nil {
		t.Fatal("Media should be nil when Chat is set")
	}
	if msg.GetPing() != nil {
		t.Fatal("Ping should be nil when Chat is set")
	}
}

func TestEmptyFrameRoundTrip(t *testing.T) {
	original := &pb.Frame{}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal empty Frame: %v", err)
	}

	decoded := &pb.Frame{}
	if err := proto.Unmarshal(data, decoded); err != nil {
		t.Fatalf("Unmarshal empty Frame: %v", err)
	}

	if !proto.Equal(original, decoded) {
		t.Fatal("empty Frame round-trip mismatch")
	}
}
