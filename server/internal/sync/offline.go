package sync

import (
	"context"
	"fmt"

	"github.com/qmish/FProto/server/internal/inbox"
)

const DefaultPageSize = 100

// OfflineSync handles backfilling missed messages from the Inbox for a reconnecting user.
type OfflineSync struct {
	inboxStore *inbox.Store
	pageSize   int
}

func NewOfflineSync(inboxStore *inbox.Store) *OfflineSync {
	return &OfflineSync{
		inboxStore: inboxStore,
		pageSize:   DefaultPageSize,
	}
}

// SyncResult contains a batch of messages and pagination info.
type SyncResult struct {
	Messages []inbox.Entry
	HasMore  bool
}

// GetMissedMessages returns messages that the user missed after lastSeenMessageID.
// Uses cursor-based pagination via Inbox GetAfterMessageID.
func (o *OfflineSync) GetMissedMessages(ctx context.Context, userID, lastSeenMessageID []byte) (*SyncResult, error) {
	if len(lastSeenMessageID) == 0 {
		return o.getUndelivered(ctx, userID)
	}

	entries, err := o.inboxStore.GetAfterMessageID(ctx, userID, lastSeenMessageID, o.pageSize+1)
	if err != nil {
		return nil, fmt.Errorf("get after message_id: %w", err)
	}

	hasMore := len(entries) > o.pageSize
	if hasMore {
		entries = entries[:o.pageSize]
	}

	return &SyncResult{
		Messages: entries,
		HasMore:  hasMore,
	}, nil
}

// GetMissedMessagesPaginated iterates in pages, calling callback for each batch.
func (o *OfflineSync) GetMissedMessagesPaginated(ctx context.Context, userID, lastSeenMessageID []byte, callback func([]inbox.Entry) error) error {
	cursor := lastSeenMessageID
	for {
		result, err := o.GetMissedMessages(ctx, userID, cursor)
		if err != nil {
			return err
		}
		if len(result.Messages) == 0 {
			return nil
		}
		if err := callback(result.Messages); err != nil {
			return fmt.Errorf("callback: %w", err)
		}
		if !result.HasMore {
			return nil
		}
		cursor = result.Messages[len(result.Messages)-1].MessageID
	}
}

func (o *OfflineSync) getUndelivered(ctx context.Context, userID []byte) (*SyncResult, error) {
	entries, err := o.inboxStore.GetUndelivered(ctx, userID, o.pageSize+1)
	if err != nil {
		return nil, fmt.Errorf("get undelivered: %w", err)
	}
	hasMore := len(entries) > o.pageSize
	if hasMore {
		entries = entries[:o.pageSize]
	}
	return &SyncResult{
		Messages: entries,
		HasMore:  hasMore,
	}, nil
}
