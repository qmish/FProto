package sync

import (
	"context"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/inbox"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// Handler processes consumed Kafka messages: dedup -> inbox insert -> push.
type Handler struct {
	dedup      *Dedup
	inboxStore *inbox.Store
	pushClient pb.PushServiceClient
}

func NewHandler(dedup *Dedup, inboxStore *inbox.Store, pushClient pb.PushServiceClient) *Handler {
	return &Handler{
		dedup:      dedup,
		inboxStore: inboxStore,
		pushClient: pushClient,
	}
}

// Handle processes a single Kafka message.
func (h *Handler) Handle(msg *sarama.ConsumerMessage) error {
	var appMsg pb.AppMessage
	if err := proto.Unmarshal(msg.Value, &appMsg); err != nil {
		return fmt.Errorf("unmarshal AppMessage: %w", err)
	}

	userID := extractRecipientID(&appMsg, msg.Topic)
	if userID == nil {
		log.Printf("sync handler: no recipient for topic=%s, skipping", msg.Topic)
		return nil
	}

	dup, err := h.dedup.IsDuplicate(context.Background(), userID, appMsg.MessageId)
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if dup {
		log.Printf("sync handler: duplicate message_id=%x for user=%x, skipping", appMsg.MessageId, userID)
		return nil
	}

	senderID := extractSenderID(&appMsg)
	conversationID := extractConversationID(&appMsg, msg.Topic)

	entry := &inbox.Entry{
		UserID:         userID,
		MessageID:      appMsg.MessageId,
		ConversationID: conversationID,
		SenderID:       senderID,
		Payload:        msg.Value,
	}

	if err := h.inboxStore.Insert(context.Background(), entry); err != nil {
		return fmt.Errorf("inbox insert: %w", err)
	}

	if h.pushClient != nil {
		_, err := h.pushClient.PushToUser(context.Background(), &pb.PushToUserRequest{
			UserId:  userID,
			Message: &appMsg,
		})
		if err != nil {
			log.Printf("sync handler: push failed for user=%x: %v (will deliver on sync)", userID, err)
		}
	}

	return nil
}

func extractRecipientID(msg *pb.AppMessage, topic string) []byte {
	if chat, ok := msg.Body.(*pb.AppMessage_Chat); ok && chat.Chat != nil {
		if len(chat.Chat.RecipientId) > 0 {
			return chat.Chat.RecipientId
		}
	}
	// For user-<hex> topics, the partition key is the user ID in the message key
	if len(topic) > 5 && topic[:5] == "user-" {
		uid, err := hexDecode(topic[5:])
		if err == nil {
			return uid
		}
	}
	return nil
}

func extractSenderID(msg *pb.AppMessage) []byte {
	if chat, ok := msg.Body.(*pb.AppMessage_Chat); ok && chat.Chat != nil {
		return chat.Chat.SenderId
	}
	return nil
}

func extractConversationID(msg *pb.AppMessage, topic string) []byte {
	if chat, ok := msg.Body.(*pb.AppMessage_Chat); ok && chat.Chat != nil {
		if len(chat.Chat.GroupId) > 0 {
			return chat.Chat.GroupId
		}
		return chat.Chat.RecipientId
	}
	return []byte(topic)
}

func hexDecode(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s)/2; i++ {
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &b[i])
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}
