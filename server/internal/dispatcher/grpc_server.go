package dispatcher

import (
	"context"

	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer implements the DispatcherService gRPC API.
type GRPCServer struct {
	pb.UnimplementedDispatcherServiceServer
	dispatcher *Dispatcher
}

func NewGRPCServer(d *Dispatcher) *GRPCServer {
	return &GRPCServer{dispatcher: d}
}

func (g *GRPCServer) Dispatch(ctx context.Context, req *pb.DispatchRequest) (*pb.DispatchResponse, error) {
	if req.Message == nil {
		return nil, status.Errorf(codes.InvalidArgument, "message is required")
	}

	if err := g.dispatcher.Dispatch(ctx, req.SenderId, req.Message); err != nil {
		return nil, status.Errorf(codes.Internal, "dispatch: %v", err)
	}

	return &pb.DispatchResponse{
		Success:      true,
		AckMessageId: req.Message.MessageId,
	}, nil
}
