package core

import (
	"context"

	"github.com/qmish/FProto/proto-core/session"
	"github.com/qmish/FProto/proto-core/transport"
)

// MessageHandler is called when a decrypted application message is received.
type MessageHandler interface {
	HandleMessage(ctx context.Context, sessionID []byte, payload []byte) error
}

// MessageHandlerFunc adapts a plain function to the MessageHandler interface.
type MessageHandlerFunc func(ctx context.Context, sessionID []byte, payload []byte) error

func (f MessageHandlerFunc) HandleMessage(ctx context.Context, sessionID []byte, payload []byte) error {
	return f(ctx, sessionID, payload)
}

// SessionHook provides lifecycle callbacks for session state changes.
type SessionHook interface {
	OnCreate(ctx context.Context, s *session.Session)
	OnActive(ctx context.Context, s *session.Session)
	OnExpire(ctx context.Context, sessionID []byte)
}

// NoopSessionHook is a no-op implementation of SessionHook.
type NoopSessionHook struct{}

func (NoopSessionHook) OnCreate(_ context.Context, _ *session.Session) {}
func (NoopSessionHook) OnActive(_ context.Context, _ *session.Session) {}
func (NoopSessionHook) OnExpire(_ context.Context, _ []byte)           {}

// TransportHook provides callbacks for transport connection events.
type TransportHook interface {
	OnConnect(ctx context.Context, conn transport.Conn)
	OnDisconnect(ctx context.Context, conn transport.Conn, err error)
}

// NoopTransportHook is a no-op implementation of TransportHook.
type NoopTransportHook struct{}

func (NoopTransportHook) OnConnect(_ context.Context, _ transport.Conn)          {}
func (NoopTransportHook) OnDisconnect(_ context.Context, _ transport.Conn, _ error) {}
