package inbox

import "time"

type Entry struct {
	ID             int64
	UserID         []byte
	MessageID      []byte // UUID
	ConversationID []byte
	SenderID       []byte
	Payload        []byte
	Delivered      bool
	Read           bool
	CreatedAt      time.Time
}
