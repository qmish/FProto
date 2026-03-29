package sync

import (
	"testing"

	"github.com/qmish/FProto/server/internal/inbox"
)

func TestOfflineSyncConstants(t *testing.T) {
	if DefaultPageSize != 100 {
		t.Fatalf("expected DefaultPageSize=100, got %d", DefaultPageSize)
	}
}

func TestSyncResult_NoMessages(t *testing.T) {
	result := &SyncResult{
		Messages: nil,
		HasMore:  false,
	}
	if result.HasMore {
		t.Fatal("should not have more")
	}
	if len(result.Messages) != 0 {
		t.Fatal("should be empty")
	}
}

func TestSyncResult_WithMessages(t *testing.T) {
	entries := make([]inbox.Entry, 10)
	for i := range entries {
		entries[i] = inbox.Entry{
			ID:        int64(i + 1),
			UserID:    []byte{0xaa},
			MessageID: []byte{byte(i)},
			Payload:   []byte("test"),
		}
	}
	result := &SyncResult{
		Messages: entries,
		HasMore:  true,
	}
	if !result.HasMore {
		t.Fatal("expected more")
	}
	if len(result.Messages) != 10 {
		t.Fatalf("expected 10 messages, got %d", len(result.Messages))
	}
}

func TestSyncResult_Pagination(t *testing.T) {
	entries := make([]inbox.Entry, DefaultPageSize+1)
	for i := range entries {
		entries[i] = inbox.Entry{
			ID:        int64(i + 1),
			MessageID: []byte{byte(i % 256)},
		}
	}

	hasMore := len(entries) > DefaultPageSize
	if !hasMore {
		t.Fatal("expected hasMore=true for 101 entries")
	}

	trimmed := entries[:DefaultPageSize]
	if len(trimmed) != DefaultPageSize {
		t.Fatalf("expected %d after trim, got %d", DefaultPageSize, len(trimmed))
	}
}

func TestOfflineSync_NewWithInboxStore(t *testing.T) {
	os := NewOfflineSync(nil)
	if os.pageSize != DefaultPageSize {
		t.Fatalf("expected pageSize=%d, got %d", DefaultPageSize, os.pageSize)
	}
}

func TestSyncResult_MultipleConversations(t *testing.T) {
	entries := []inbox.Entry{
		{ID: 1, ConversationID: []byte{0x01}},
		{ID: 2, ConversationID: []byte{0x02}},
		{ID: 3, ConversationID: []byte{0x01}},
		{ID: 4, ConversationID: []byte{0x03}},
	}
	result := &SyncResult{Messages: entries, HasMore: false}

	convs := map[string]int{}
	for _, e := range result.Messages {
		convs[string(e.ConversationID)]++
	}
	if len(convs) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(convs))
	}
}
