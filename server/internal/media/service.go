package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Service provides media upload/download operations.
type Service struct {
	store *Store
	s3    *S3Client
}

func NewService(store *Store, s3 *S3Client) *Service {
	return &Service{store: store, s3: s3}
}

// InitUpload creates a media upload record and returns a presigned PUT URL.
func (svc *Service) InitUpload(ctx context.Context, userID []byte, mimeType string, totalSize int64, fileHash []byte) (streamID []byte, uploadURL string, err error) {
	streamID = make([]byte, 16)
	if _, err := rand.Read(streamID); err != nil {
		return nil, "", fmt.Errorf("generate stream_id: %w", err)
	}

	s3Key := fmt.Sprintf("uploads/%s/%s", hex.EncodeToString(userID), hex.EncodeToString(streamID))

	upload := &Upload{
		StreamID:  streamID,
		UserID:    userID,
		MimeType:  mimeType,
		TotalSize: totalSize,
		S3Key:     s3Key,
		FileHash:  fileHash,
	}

	if err := svc.store.Insert(ctx, upload); err != nil {
		return nil, "", fmt.Errorf("insert: %w", err)
	}

	uploadURL, err = svc.s3.PresignedPutURL(ctx, s3Key)
	if err != nil {
		return nil, "", fmt.Errorf("presigned url: %w", err)
	}

	return streamID, uploadURL, nil
}

// CompleteUpload verifies the upload in S3 and marks it complete.
func (svc *Service) CompleteUpload(ctx context.Context, streamID, fileHash []byte) (downloadURL string, err error) {
	upload, err := svc.store.GetByStreamID(ctx, streamID)
	if err != nil {
		return "", fmt.Errorf("get upload: %w", err)
	}

	if upload.Status != StatusUploading {
		return "", fmt.Errorf("upload status is %s, expected uploading", upload.Status)
	}

	exists, size, err := svc.s3.ObjectExists(ctx, upload.S3Key)
	if err != nil {
		return "", fmt.Errorf("check s3 object: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("object not found in S3")
	}
	if size != upload.TotalSize {
		return "", fmt.Errorf("size mismatch: expected %d, got %d", upload.TotalSize, size)
	}

	downloadURL, err = svc.s3.PresignedGetURL(ctx, upload.S3Key)
	if err != nil {
		return "", fmt.Errorf("download url: %w", err)
	}

	if err := svc.store.MarkCompleted(ctx, streamID, downloadURL); err != nil {
		return "", fmt.Errorf("mark completed: %w", err)
	}

	return downloadURL, nil
}

// GetDownloadURL generates a fresh presigned download URL.
func (svc *Service) GetDownloadURL(ctx context.Context, streamID []byte) (string, error) {
	upload, err := svc.store.GetByStreamID(ctx, streamID)
	if err != nil {
		return "", fmt.Errorf("get upload: %w", err)
	}
	if upload.Status != StatusCompleted {
		return "", fmt.Errorf("upload not completed (status=%s)", upload.Status)
	}
	return svc.s3.PresignedGetURL(ctx, upload.S3Key)
}
