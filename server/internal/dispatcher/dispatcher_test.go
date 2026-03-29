package dispatcher

import (
	"testing"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func TestRouteChat_Recipient(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa, 0xbb, 0xcc, 0xdd},
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "user-aabbccdd" {
		t.Fatalf("expected user-aabbccdd, got %s", route.Topic)
	}
	if string(route.PartitionKey) != string([]byte{0xaa, 0xbb, 0xcc, 0xdd}) {
		t.Fatal("partition key mismatch")
	}
}

func TestRouteChat_Group(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4},
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				GroupId: []byte{0x11, 0x22},
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "group-1122" {
		t.Fatalf("expected group-1122, got %s", route.Topic)
	}
}

func TestRouteChat_GroupPriority(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{
				RecipientId: []byte{0xaa},
				GroupId:     []byte{0xbb},
			},
		},
	}
	route, _ := RouteMessage(msg)
	if route.Topic != "group-bb" {
		t.Fatalf("group should take priority, got %s", route.Topic)
	}
}

func TestRouteCommand(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Command{
			Command: &pb.Command{
				Type: pb.Command_CREATE_GROUP,
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "commands" {
		t.Fatalf("expected commands, got %s", route.Topic)
	}
}

func TestRouteAck_FallsToEvents(t *testing.T) {
	mid := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	msg := &pb.AppMessage{
		MessageId: mid,
		Body: &pb.AppMessage_Ack{
			Ack: &pb.Ack{
				AckMessageId: mid,
				Status:       pb.Ack_DELIVERED,
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "events" {
		t.Fatalf("expected events, got %s", route.Topic)
	}
}

func TestRoutePing_FallsToEvents(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4},
		Body:      &pb.AppMessage_Ping{Ping: &pb.Ping{}},
	}
	route, _ := RouteMessage(msg)
	if route.Topic != "events" {
		t.Fatalf("expected events, got %s", route.Topic)
	}
}

func TestRouteNilMessage(t *testing.T) {
	_, err := RouteMessage(nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}
}

func TestRouteChatNoRecipient(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Chat{
			Chat: &pb.ChatMessage{},
		},
	}
	_, err := RouteMessage(msg)
	if err == nil {
		t.Fatal("expected error for chat without recipient")
	}
}

func TestRouteMedia(t *testing.T) {
	msg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4},
		Body: &pb.AppMessage_Media{
			Media: &pb.MediaChunk{
				StreamId:   []byte{0xaa, 0xbb, 0xcc},
				ChunkIndex: 0,
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "media-events" {
		t.Fatalf("expected media-events, got %s", route.Topic)
	}
}

func TestRouteSenderKey(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_SenderKey{
			SenderKey: &pb.SenderKeyDistribution{
				GroupId:  []byte{0x11, 0x22},
				SenderId: []byte{0xcc, 0xdd},
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "events" {
		t.Fatalf("expected events, got %s", route.Topic)
	}
}

func TestRouteMediaNil(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Media{Media: nil},
	}
	_, err := RouteMessage(msg)
	if err == nil {
		t.Fatal("expected error for nil media")
	}
}

func TestRouteSenderKeyNoSender(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_SenderKey{
			SenderKey: &pb.SenderKeyDistribution{
				GroupId: []byte{0x11},
			},
		},
	}
	_, err := RouteMessage(msg)
	if err == nil {
		t.Fatal("expected error for sender key without sender_id")
	}
}

func TestRouteSignaling(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Signaling{
			Signaling: &pb.SignalingMessage{
				RecipientId: []byte{0xaa, 0xbb},
				Content: &pb.SignalingMessage_Offer{
					Offer: &pb.SDPMessage{Sdp: "v=0...", Type: "offer"},
				},
			},
		},
	}
	route, err := RouteMessage(msg)
	if err != nil {
		t.Fatalf("route: %v", err)
	}
	if route.Topic != "user-aabb" {
		t.Fatalf("expected user-aabb, got %s", route.Topic)
	}
}

func TestRouteSignalingIce(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Signaling{
			Signaling: &pb.SignalingMessage{
				RecipientId: []byte{0xcc},
				Content: &pb.SignalingMessage_IceCandidate{
					IceCandidate: &pb.IceCandidate{
						Candidate: "candidate:...",
						SdpMid:    "0",
					},
				},
			},
		},
	}
	route, _ := RouteMessage(msg)
	if route.Topic != "user-cc" {
		t.Fatalf("expected user-cc, got %s", route.Topic)
	}
}

func TestRouteSignalingNoRecipient(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Signaling{
			Signaling: &pb.SignalingMessage{},
		},
	}
	_, err := RouteMessage(msg)
	if err == nil {
		t.Fatal("expected error for signaling without recipient")
	}
}

func TestRouteSignalingStreamControl(t *testing.T) {
	msg := &pb.AppMessage{
		Body: &pb.AppMessage_Signaling{
			Signaling: &pb.SignalingMessage{
				RecipientId: []byte{0xdd},
				Content: &pb.SignalingMessage_StreamControl{
					StreamControl: &pb.StreamControl{
						Action:   pb.StreamControl_START,
						StreamId: []byte{0x01},
					},
				},
			},
		},
	}
	route, _ := RouteMessage(msg)
	if route.Topic != "user-dd" {
		t.Fatalf("expected user-dd, got %s", route.Topic)
	}
}
