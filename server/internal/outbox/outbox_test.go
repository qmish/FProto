package outbox

import (
	"testing"
)

func TestEntryModel(t *testing.T) {
	e := Entry{
		MessageID:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Topic:        "user-0a0b0c0d",
		PartitionKey: []byte{0x0a, 0x0b, 0x0c, 0x0d},
		Payload:      []byte("test payload"),
		Status:       StatusPending,
	}

	if e.Status != StatusPending {
		t.Fatalf("expected pending, got %s", e.Status)
	}
	if e.Topic != "user-0a0b0c0d" {
		t.Fatalf("unexpected topic: %s", e.Topic)
	}
}

func TestStatusValues(t *testing.T) {
	if StatusPending != "pending" {
		t.Fatal("StatusPending")
	}
	if StatusPublished != "published" {
		t.Fatal("StatusPublished")
	}
	if StatusFailed != "failed" {
		t.Fatal("StatusFailed")
	}
}

func TestRelayConstants(t *testing.T) {
	if MaxRetries != 5 {
		t.Fatalf("expected MaxRetries=5, got %d", MaxRetries)
	}
	if DefaultBatchSize != 100 {
		t.Fatalf("expected batch=100, got %d", DefaultBatchSize)
	}
}
