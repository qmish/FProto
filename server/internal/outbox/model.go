package outbox

import "time"

type Status string

const (
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusFailed    Status = "failed"
)

type Entry struct {
	ID           int64
	MessageID    []byte // UUID
	Topic        string
	PartitionKey []byte
	Payload      []byte
	Status       Status
	RetryCount   int
	CreatedAt    time.Time
	PublishedAt  *time.Time
}
