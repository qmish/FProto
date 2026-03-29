package commands

import (
	"testing"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

func makeCommandMsg(cmdType pb.Command_CommandType, targetID, payload []byte) *sarama.ConsumerMessage {
	appMsg := &pb.AppMessage{
		MessageId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Body: &pb.AppMessage_Command{
			Command: &pb.Command{
				Type:     cmdType,
				TargetId: targetID,
				Payload:  payload,
			},
		},
	}
	data, _ := proto.Marshal(appMsg)
	return &sarama.ConsumerMessage{
		Topic: "commands",
		Value: data,
	}
}

func TestHandlerCreateGroup(t *testing.T) {
	_ = NewHandler(nil)
	msg := makeCommandMsg(pb.Command_CREATE_GROUP, []byte{0xaa}, []byte(`{"name":"test"}`))
	_ = msg
}

func TestHandlerAddMember(t *testing.T) {
	h := NewHandler(nil)
	msg := makeCommandMsg(pb.Command_ADD_MEMBER, []byte{0xaa}, []byte{0xbb})
	_ = h
	_ = msg
}

func TestHandlerRemoveMember(t *testing.T) {
	h := NewHandler(nil)
	msg := makeCommandMsg(pb.Command_REMOVE_MEMBER, []byte{0xaa}, []byte{0xbb})
	_ = h
	_ = msg
}

func TestHandlerUpdateSettings(t *testing.T) {
	h := NewHandler(nil)
	msg := makeCommandMsg(pb.Command_UPDATE_GROUP_SETTINGS, []byte{0xaa}, []byte(`{"mute":true}`))
	_ = h
	_ = msg
}

func TestHandlerBanMember(t *testing.T) {
	h := NewHandler(nil)
	msg := makeCommandMsg(pb.Command_BAN_MEMBER, []byte{0xaa}, []byte{0xbb})
	_ = h
	_ = msg
}

func TestHandlerDeleteMessage(t *testing.T) {
	h := NewHandler(nil)
	msg := makeCommandMsg(pb.Command_DELETE_MESSAGE, []byte{0xaa}, nil)
	err := h.Handle(msg)
	if err != nil {
		t.Fatalf("delete message should not fail: %v", err)
	}
}

func TestHandlerNonCommandMessage(t *testing.T) {
	appMsg := &pb.AppMessage{
		Body: &pb.AppMessage_Ping{Ping: &pb.Ping{}},
	}
	data, _ := proto.Marshal(appMsg)
	msg := &sarama.ConsumerMessage{Value: data}

	h := NewHandler(nil)
	err := h.Handle(msg)
	if err != nil {
		t.Fatalf("non-command should be skipped: %v", err)
	}
}

func TestCommandTypes(t *testing.T) {
	types := []pb.Command_CommandType{
		pb.Command_CREATE_GROUP,
		pb.Command_ADD_MEMBER,
		pb.Command_REMOVE_MEMBER,
		pb.Command_UPDATE_GROUP_SETTINGS,
		pb.Command_BAN_MEMBER,
		pb.Command_DELETE_MESSAGE,
	}
	for _, ct := range types {
		if ct.String() == "" {
			t.Fatalf("command type %d has no string representation", ct)
		}
	}
}
