package media

import "time"

type Status string

const (
	StatusUploading Status = "uploading"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusExpired   Status = "expired"
)

type Upload struct {
	ID        int64
	StreamID  []byte // UUID
	UserID    []byte
	MimeType  string
	TotalSize int64
	Status    Status
	S3Key     string
	S3URL     string
	FileHash  []byte // SHA-256
	CreatedAt time.Time
	UpdatedAt time.Time
}
