package dispatcher

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/qmish/FProto/server/internal/outbox"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// Dispatcher accepts AppMessages, routes them, and writes to the Outbox.
type Dispatcher struct {
	outboxStore *outbox.Store
}

func New(outboxStore *outbox.Store) *Dispatcher {
	return &Dispatcher{outboxStore: outboxStore}
}

// Dispatch routes a message and inserts it into the outbox for reliable delivery.
func (d *Dispatcher) Dispatch(ctx context.Context, senderID []byte, msg *pb.AppMessage) error {
	route, err := RouteMessage(msg)
	if err != nil {
		return fmt.Errorf("route: %w", err)
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	entry := &outbox.Entry{
		MessageID:    msg.MessageId,
		Topic:        route.Topic,
		PartitionKey: route.PartitionKey,
		Payload:      payload,
	}

	if err := d.outboxStore.Insert(ctx, entry); err != nil {
		return fmt.Errorf("outbox insert: %w", err)
	}

	return nil
}
