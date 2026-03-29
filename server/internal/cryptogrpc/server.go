package cryptogrpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/flynn/noise"
	"github.com/qmish/FProto/proto-core/crypto"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var cipherSuite = noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashBLAKE2s)

// Server implements the CryptoService gRPC API.
type Server struct {
	pb.UnimplementedCryptoServiceServer

	serverKey *crypto.KeyPair

	mu         sync.Mutex
	handshakes map[string]*noise.HandshakeState
}

// NewServer creates a new crypto gRPC server.
func NewServer(serverKey *crypto.KeyPair) *Server {
	return &Server{
		serverKey:  serverKey,
		handshakes: make(map[string]*noise.HandshakeState),
	}
}

func (s *Server) InitiateHandshake(_ context.Context, req *pb.HandshakeInitRequest) (*pb.HandshakeInitResponse, error) {
	hs, err := noise.NewHandshakeState(noise.Config{
		CipherSuite:   cipherSuite,
		Pattern:       noise.HandshakeXX,
		Initiator:     req.IsInitiator,
		StaticKeypair: noise.DHKey{Private: s.serverKey.Private, Public: s.serverKey.Public},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "init handshake: %v", err)
	}

	stateID := make([]byte, 16)
	if _, err := rand.Read(stateID); err != nil {
		return nil, status.Errorf(codes.Internal, "generate state id: %v", err)
	}

	s.mu.Lock()
	s.handshakes[hex.EncodeToString(stateID)] = hs
	s.mu.Unlock()

	return &pb.HandshakeInitResponse{HandshakeStateId: stateID}, nil
}

func (s *Server) ProcessHandshake(_ context.Context, req *pb.HandshakeStepRequest) (*pb.HandshakeStepResponse, error) {
	key := hex.EncodeToString(req.HandshakeStateId)

	s.mu.Lock()
	hs, ok := s.handshakes[key]
	s.mu.Unlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "handshake state not found")
	}

	if len(req.MessagePayload) > 0 {
		_, csRecv, csSend, err := hs.ReadMessage(nil, req.MessagePayload)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "read message: %v", err)
		}
		if csRecv != nil && csSend != nil {
			s.mu.Lock()
			delete(s.handshakes, key)
			s.mu.Unlock()

			cryptoState := serializeCipherStates(csSend, csRecv)
			return &pb.HandshakeStepResponse{
				Completed:   true,
				CryptoState: cryptoState,
			}, nil
		}
	}

	msg, csSend, csRecv, err := hs.WriteMessage(nil, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "write message: %v", err)
	}

	if csSend != nil && csRecv != nil {
		s.mu.Lock()
		delete(s.handshakes, key)
		s.mu.Unlock()

		cryptoState := serializeCipherStates(csSend, csRecv)
		return &pb.HandshakeStepResponse{
			ResponsePayload: msg,
			Completed:       true,
			CryptoState:     cryptoState,
		}, nil
	}

	return &pb.HandshakeStepResponse{
		ResponsePayload: msg,
		Completed:       false,
	}, nil
}

func (s *Server) Encrypt(_ context.Context, req *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	aead, err := crypto.NewAEAD(req.CryptoState[:32])
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crypto state: %v", err)
	}

	ciphertext := crypto.EncryptAEAD(aead, req.SessionId, req.Seq, req.Plaintext)
	return &pb.EncryptResponse{
		Ciphertext:         ciphertext,
		UpdatedCryptoState: req.CryptoState,
	}, nil
}

func (s *Server) Decrypt(_ context.Context, req *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	aead, err := crypto.NewAEAD(req.CryptoState[:32])
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid crypto state: %v", err)
	}

	plaintext, err := crypto.DecryptAEAD(aead, req.SessionId, req.Seq, req.Ciphertext)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decrypt: %v", err)
	}
	return &pb.DecryptResponse{
		Plaintext:          plaintext,
		UpdatedCryptoState: req.CryptoState,
	}, nil
}

func (s *Server) Rekey(_ context.Context, req *pb.RekeyRequest) (*pb.RekeyResponse, error) {
	if len(req.CryptoState) < 32 {
		return nil, status.Errorf(codes.InvalidArgument, "crypto state too short")
	}

	newKey, err := crypto.Rekey(req.CryptoState[:32])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rekey: %v", err)
	}

	newState := make([]byte, len(req.CryptoState))
	copy(newState, req.CryptoState)
	copy(newState[:32], newKey)
	crypto.Zeroize(newKey)

	return &pb.RekeyResponse{UpdatedCryptoState: newState}, nil
}

func serializeCipherStates(send, recv *noise.CipherState) []byte {
	_ = send
	_ = recv
	placeholder := make([]byte, 64)
	if _, err := rand.Read(placeholder); err != nil {
		panic(fmt.Sprintf("rand read: %v", err))
	}
	return placeholder
}
