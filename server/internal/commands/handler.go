package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/groups"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// Handler processes Command messages from Kafka.
type Handler struct {
	groupStore *groups.Store
}

func NewHandler(groupStore *groups.Store) *Handler {
	return &Handler{groupStore: groupStore}
}

// Handle processes a single Kafka message containing a Command.
func (h *Handler) Handle(msg *sarama.ConsumerMessage) error {
	var appMsg pb.AppMessage
	if err := proto.Unmarshal(msg.Value, &appMsg); err != nil {
		return fmt.Errorf("unmarshal AppMessage: %w", err)
	}

	cmd, ok := appMsg.Body.(*pb.AppMessage_Command)
	if !ok || cmd.Command == nil {
		log.Printf("command handler: not a command message, skipping")
		return nil
	}

	return h.processCommand(context.Background(), cmd.Command)
}

func (h *Handler) processCommand(ctx context.Context, cmd *pb.Command) error {
	switch cmd.Type {
	case pb.Command_CREATE_GROUP:
		return h.handleCreateGroup(ctx, cmd)
	case pb.Command_ADD_MEMBER:
		return h.handleAddMember(ctx, cmd)
	case pb.Command_REMOVE_MEMBER:
		return h.handleRemoveMember(ctx, cmd)
	case pb.Command_UPDATE_GROUP_SETTINGS:
		return h.handleUpdateSettings(ctx, cmd)
	case pb.Command_BAN_MEMBER:
		return h.handleBanMember(ctx, cmd)
	case pb.Command_DELETE_MESSAGE:
		return h.handleDeleteMessage(ctx, cmd)
	default:
		log.Printf("command handler: unknown command type %s", cmd.Type)
		return nil
	}
}

func (h *Handler) handleCreateGroup(ctx context.Context, cmd *pb.Command) error {
	if len(cmd.TargetId) == 0 {
		return fmt.Errorf("create_group: target_id (group_id) required")
	}

	var settings struct {
		Name string `json:"name"`
	}
	if len(cmd.Payload) > 0 {
		if err := json.Unmarshal(cmd.Payload, &settings); err != nil {
			return fmt.Errorf("parse settings: %w", err)
		}
	}
	if settings.Name == "" {
		settings.Name = "New Group"
	}

	g := &groups.Group{
		ID:        cmd.TargetId,
		Name:      settings.Name,
		CreatorID: cmd.Payload[:0], // sender from AppMessage context
		Settings:  []byte("{}"),
	}

	// Use TargetId as both group ID and creator fallback
	if len(cmd.Payload) > 0 {
		g.Settings = cmd.Payload
	}

	return h.groupStore.CreateGroup(ctx, g)
}

func (h *Handler) handleAddMember(ctx context.Context, cmd *pb.Command) error {
	if len(cmd.TargetId) == 0 || len(cmd.Payload) == 0 {
		return fmt.Errorf("add_member: target_id (group_id) and payload (user_id) required")
	}
	return h.groupStore.AddMember(ctx, cmd.TargetId, cmd.Payload, groups.RoleMember)
}

func (h *Handler) handleRemoveMember(ctx context.Context, cmd *pb.Command) error {
	if len(cmd.TargetId) == 0 || len(cmd.Payload) == 0 {
		return fmt.Errorf("remove_member: target_id (group_id) and payload (user_id) required")
	}
	return h.groupStore.RemoveMember(ctx, cmd.TargetId, cmd.Payload)
}

func (h *Handler) handleUpdateSettings(ctx context.Context, cmd *pb.Command) error {
	if len(cmd.TargetId) == 0 {
		return fmt.Errorf("update_settings: target_id (group_id) required")
	}
	return h.groupStore.UpdateSettings(ctx, cmd.TargetId, cmd.Payload)
}

func (h *Handler) handleBanMember(ctx context.Context, cmd *pb.Command) error {
	if len(cmd.TargetId) == 0 || len(cmd.Payload) == 0 {
		return fmt.Errorf("ban_member: target_id (group_id) and payload (user_id) required")
	}
	return h.groupStore.BanMember(ctx, cmd.TargetId, cmd.Payload)
}

func (h *Handler) handleDeleteMessage(_ context.Context, cmd *pb.Command) error {
	log.Printf("command handler: delete_message for target=%x (inbox deletion deferred)", cmd.TargetId)
	return nil
}
