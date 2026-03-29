package media

import (
	"context"
	"time"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer implements the MediaService gRPC API.
type GRPCServer struct {
	pb.UnimplementedMediaServiceServer
	svc *Service
}

func NewGRPCServer(svc *Service) *GRPCServer {
	return &GRPCServer{svc: svc}
}

func (g *GRPCServer) InitUpload(ctx context.Context, req *pb.InitUploadRequest) (*pb.InitUploadResponse, error) {
	if len(req.UserId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.MimeType == "" {
		return nil, status.Error(codes.InvalidArgument, "mime_type is required")
	}
	if req.TotalSize == 0 {
		return nil, status.Error(codes.InvalidArgument, "total_size is required")
	}

	streamID, uploadURL, err := g.svc.InitUpload(ctx, req.UserId, req.MimeType, int64(req.TotalSize), req.FileHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "init upload: %v", err)
	}

	return &pb.InitUploadResponse{
		StreamId:           streamID,
		PresignedUploadUrl: uploadURL,
		ExpiresAt:          time.Now().Add(UploadExpiry).Unix(),
	}, nil
}

func (g *GRPCServer) CompleteUpload(ctx context.Context, req *pb.CompleteUploadRequest) (*pb.CompleteUploadResponse, error) {
	if len(req.StreamId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	downloadURL, err := g.svc.CompleteUpload(ctx, req.StreamId, req.FileHash)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "complete upload: %v", err)
	}

	return &pb.CompleteUploadResponse{
		Success:     true,
		DownloadUrl: downloadURL,
	}, nil
}

func (g *GRPCServer) GetDownloadURL(ctx context.Context, req *pb.GetDownloadURLRequest) (*pb.GetDownloadURLResponse, error) {
	if len(req.StreamId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "stream_id is required")
	}

	url, err := g.svc.GetDownloadURL(ctx, req.StreamId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "download url: %v", err)
	}

	return &pb.GetDownloadURLResponse{
		PresignedDownloadUrl: url,
		ExpiresAt:            time.Now().Add(DownloadExpiry).Unix(),
	}, nil
}
