package sessiongrpc

import (
	"context"
	"time"

	"github.com/qmish/FProto/proto-core/session"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the SessionService gRPC API.
type Server struct {
	pb.UnimplementedSessionServiceServer
	mgr *session.Manager
}

// NewServer creates a new session gRPC server.
func NewServer(mgr *session.Manager) *Server {
	return &Server{mgr: mgr}
}

func (g *Server) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	s, err := g.mgr.CreateSession(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create session: %v", err)
	}
	return &pb.CreateSessionResponse{
		SessionId: s.SessionID,
		Session:   sessionToProto(s),
	}, nil
}

func (g *Server) GetSession(ctx context.Context, req *pb.GetSessionRequest) (*pb.Session, error) {
	s, err := g.mgr.GetSession(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}
	return sessionToProto(s), nil
}

func (g *Server) UpdateSessionStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.Session, error) {
	event, err := statusToEvent(req.Status)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	s, err := g.mgr.TransitionSession(ctx, req.SessionId, event)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "transition: %v", err)
	}
	if s == nil {
		return nil, status.Errorf(codes.NotFound, "session destroyed")
	}
	return sessionToProto(s), nil
}

func (g *Server) RegisterConnection(ctx context.Context, req *pb.RegisterConnectionRequest) (*pb.RegisterConnectionResponse, error) {
	conn := &session.Connection{
		ConnectionID: req.ConnectionId,
		Transport:    protoToTransport(req.Transport),
		GatewayID:    req.GatewayId,
		RemoteAddr:   req.RemoteAddr,
		ConnectedAt:  time.Now(),
	}
	count, err := g.mgr.RegisterConnection(ctx, req.SessionId, conn)
	if err != nil {
		return nil, status.Errorf(codes.ResourceExhausted, "%v", err)
	}
	return &pb.RegisterConnectionResponse{ActiveConnections: uint32(count)}, nil
}

func (g *Server) UnregisterConnection(ctx context.Context, req *pb.UnregisterConnectionRequest) (*emptypb.Empty, error) {
	if err := g.mgr.UnregisterConnection(ctx, req.SessionId, req.ConnectionId); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &emptypb.Empty{}, nil
}

func (g *Server) RouteToSession(_ context.Context, _ *pb.RouteRequest) (*pb.RouteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "route_to_session not implemented yet")
}

func sessionToProto(s *session.Session) *pb.Session {
	p := &pb.Session{
		SessionId:   s.SessionID,
		UserId:      s.UserID,
		DeviceId:    s.DeviceID,
		Status:      statusToProto(s.Status),
		CryptoState: s.CryptoState,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		LastActive:  timestamppb.New(s.LastActive),
		ExpiresAt:   timestamppb.New(s.ExpiresAt),
	}
	for _, c := range s.Connections {
		p.Connections = append(p.Connections, &pb.Connection{
			ConnectionId:  c.ConnectionID,
			Transport:     transportToProto(c.Transport),
			GatewayId:     c.GatewayID,
			RemoteAddr:    c.RemoteAddr,
			ConnectedAt:   timestamppb.New(c.ConnectedAt),
			LastPacketSeq: c.LastPacketSeq,
		})
	}
	return p
}

func statusToProto(s session.Status) pb.SessionStatus {
	switch s {
	case session.StatusConnecting:
		return pb.SessionStatus_CONNECTING
	case session.StatusActive:
		return pb.SessionStatus_ACTIVE
	case session.StatusSleeping:
		return pb.SessionStatus_SLEEPING
	case session.StatusExpired:
		return pb.SessionStatus_EXPIRED
	default:
		return pb.SessionStatus_SESSION_STATUS_UNSPECIFIED
	}
}

func statusToEvent(s pb.SessionStatus) (session.Event, error) {
	switch s {
	case pb.SessionStatus_ACTIVE:
		return session.EventHandshakeOK, nil
	case pb.SessionStatus_SLEEPING:
		return session.EventIdleTimeout, nil
	case pb.SessionStatus_EXPIRED:
		return session.EventExplicitLogout, nil
	default:
		return 0, session.ErrInvalidTransition
	}
}

func transportToProto(t session.TransportType) pb.TransportType {
	switch t {
	case session.TransportWebSocket:
		return pb.TransportType_WEBSOCKET
	case session.TransportQUIC:
		return pb.TransportType_QUIC
	case session.TransportGRPC:
		return pb.TransportType_GRPC
	default:
		return pb.TransportType_TRANSPORT_TYPE_UNSPECIFIED
	}
}

func protoToTransport(t pb.TransportType) session.TransportType {
	switch t {
	case pb.TransportType_WEBSOCKET:
		return session.TransportWebSocket
	case pb.TransportType_QUIC:
		return session.TransportQUIC
	case pb.TransportType_GRPC:
		return session.TransportGRPC
	default:
		return session.TransportWebSocket
	}
}
