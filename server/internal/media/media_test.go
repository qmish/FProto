package media

import (
	"testing"
)

func TestUploadModel(t *testing.T) {
	u := Upload{
		StreamID:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		UserID:    []byte{0xaa, 0xbb},
		MimeType:  "image/jpeg",
		TotalSize: 1024 * 1024,
		Status:    StatusUploading,
		S3Key:     "uploads/aabb/0102030405060708090a0b0c0d0e0f10",
	}

	if u.Status != StatusUploading {
		t.Fatal("expected uploading")
	}
	if u.MimeType != "image/jpeg" {
		t.Fatal("mime mismatch")
	}
}

func TestStatusValues(t *testing.T) {
	if StatusUploading != "uploading" {
		t.Fatal("uploading")
	}
	if StatusCompleted != "completed" {
		t.Fatal("completed")
	}
	if StatusFailed != "failed" {
		t.Fatal("failed")
	}
	if StatusExpired != "expired" {
		t.Fatal("expired")
	}
}

func TestMigrateSQL(t *testing.T) {
	if migrateSQL == "" {
		t.Fatal("migrateSQL should not be empty")
	}
}

func TestS3Constants(t *testing.T) {
	if DefaultBucket != "fproto-media" {
		t.Fatalf("expected fproto-media, got %s", DefaultBucket)
	}
	if UploadExpiry.Hours() != 1 {
		t.Fatal("upload expiry should be 1h")
	}
	if DownloadExpiry.Hours() != 24 {
		t.Fatal("download expiry should be 24h")
	}
}

func TestServiceNilS3(t *testing.T) {
	svc := NewService(nil, nil)
	if svc == nil {
		t.Fatal("service should not be nil")
	}
}
