package dispatcher

import (
	"encoding/hex"
	"fmt"

	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// Route determines the Kafka topic and partition key for an AppMessage.
type Route struct {
	Topic        string
	PartitionKey []byte
}

// RouteMessage inspects the AppMessage body and returns the appropriate Route.
func RouteMessage(msg *pb.AppMessage) (Route, error) {
	if msg == nil {
		return Route{}, fmt.Errorf("nil message")
	}

	switch body := msg.Body.(type) {
	case *pb.AppMessage_Chat:
		return routeChat(body.Chat)
	case *pb.AppMessage_Command:
		return routeCommand(body.Command)
	case *pb.AppMessage_Media:
		return routeMedia(body.Media)
	case *pb.AppMessage_SenderKey:
		return routeSenderKey(body.SenderKey)
	case *pb.AppMessage_Signaling:
		return routeSignaling(body.Signaling)
	default:
		return Route{
			Topic:        kafkapkg.TopicEvents,
			PartitionKey: msg.MessageId,
		}, nil
	}
}

func routeChat(chat *pb.ChatMessage) (Route, error) {
	if chat == nil {
		return Route{}, fmt.Errorf("nil chat message")
	}

	if len(chat.GroupId) > 0 {
		return Route{
			Topic:        "group-" + hex.EncodeToString(chat.GroupId),
			PartitionKey: chat.GroupId,
		}, nil
	}

	if len(chat.RecipientId) > 0 {
		return Route{
			Topic:        "user-" + hex.EncodeToString(chat.RecipientId),
			PartitionKey: chat.RecipientId,
		}, nil
	}

	return Route{}, fmt.Errorf("chat message has no recipient_id or group_id")
}

func routeCommand(cmd *pb.Command) (Route, error) {
	if cmd == nil {
		return Route{}, fmt.Errorf("nil command")
	}
	return Route{
		Topic:        kafkapkg.TopicCommands,
		PartitionKey: []byte(cmd.Type.String()),
	}, nil
}

func routeMedia(media *pb.MediaChunk) (Route, error) {
	if media == nil {
		return Route{}, fmt.Errorf("nil media chunk")
	}
	return Route{
		Topic:        kafkapkg.TopicMediaEvents,
		PartitionKey: media.StreamId,
	}, nil
}

func routeSenderKey(sk *pb.SenderKeyDistribution) (Route, error) {
	if sk == nil {
		return Route{}, fmt.Errorf("nil sender key distribution")
	}
	if len(sk.SenderId) == 0 {
		return Route{}, fmt.Errorf("sender key has no sender_id")
	}
	return Route{
		Topic:        kafkapkg.TopicEvents,
		PartitionKey: sk.GroupId,
	}, nil
}

func routeSignaling(sig *pb.SignalingMessage) (Route, error) {
	if sig == nil {
		return Route{}, fmt.Errorf("nil signaling message")
	}
	if len(sig.RecipientId) == 0 {
		return Route{}, fmt.Errorf("signaling has no recipient_id")
	}
	return Route{
		Topic:        "user-" + hex.EncodeToString(sig.RecipientId),
		PartitionKey: sig.RecipientId,
	}, nil
}
