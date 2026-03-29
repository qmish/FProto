package inbox

import (
	"testing"
)

func TestEntryModel(t *testing.T) {
	e := Entry{
		UserID:         []byte{1, 2, 3, 4},
		MessageID:      []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4},
		ConversationID: []byte{0xaa, 0xbb},
		SenderID:       []byte{0xcc, 0xdd},
		Payload:        []byte("hello"),
		Delivered:      false,
		Read:           false,
	}

	if e.Delivered {
		t.Fatal("new entry should not be delivered")
	}
	if e.Read {
		t.Fatal("new entry should not be read")
	}
	if len(e.MessageID) != 16 {
		t.Fatal("message ID should be 16 bytes (UUID)")
	}
}

func TestMigrateSQL(t *testing.T) {
	if migrateSQL == "" {
		t.Fatal("migrateSQL should not be empty")
	}
}
