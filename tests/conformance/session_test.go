package conformance

import (
	"testing"

	"github.com/qmish/FProto/proto-core/session"
)

func TestStandard_SessionStateMachine_ConnectingToActive(t *testing.T) {
	next, destroyed, err := session.Transition(session.StatusConnecting, session.EventHandshakeOK)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != session.StatusActive {
		t.Fatalf("expected active, got %v", next)
	}
}

func TestStandard_SessionStateMachine_ConnectingHandshakeFail(t *testing.T) {
	_, destroyed, err := session.Transition(session.StatusConnecting, session.EventHandshakeFail)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !destroyed {
		t.Fatal("should be destroyed on handshake fail")
	}
}

func TestStandard_SessionStateMachine_ActiveToSleeping(t *testing.T) {
	next, destroyed, err := session.Transition(session.StatusActive, session.EventIdleTimeout)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != session.StatusSleeping {
		t.Fatalf("expected sleeping, got %v", next)
	}
}

func TestStandard_SessionStateMachine_SleepingToActive(t *testing.T) {
	next, destroyed, err := session.Transition(session.StatusSleeping, session.EventPacketReceived)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != session.StatusActive {
		t.Fatalf("expected active, got %v", next)
	}
}

func TestStandard_SessionStateMachine_SleepingToExpired(t *testing.T) {
	next, destroyed, err := session.Transition(session.StatusSleeping, session.EventExpiryTimeout)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != session.StatusExpired {
		t.Fatalf("expected expired, got %v", next)
	}
}

func TestStandard_SessionStateMachine_ActiveLogout(t *testing.T) {
	next, _, err := session.Transition(session.StatusActive, session.EventExplicitLogout)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if next != session.StatusExpired {
		t.Fatalf("expected expired, got %v", next)
	}
}

func TestStandard_SessionStateMachine_ActivePacketStaysActive(t *testing.T) {
	next, _, err := session.Transition(session.StatusActive, session.EventPacketReceived)
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if next != session.StatusActive {
		t.Fatalf("expected active, got %v", next)
	}
}

func TestStandard_SessionStateMachine_InvalidTransitionFromExpired(t *testing.T) {
	_, _, err := session.Transition(session.StatusExpired, session.EventHandshakeOK)
	if err == nil {
		t.Fatal("transition from expired must fail")
	}
}

func TestStandard_SessionStateMachine_InvalidTransitionFromSleeping(t *testing.T) {
	_, _, err := session.Transition(session.StatusSleeping, session.EventHandshakeOK)
	if err == nil {
		t.Fatal("handshake OK from sleeping must fail")
	}
}

func TestStandard_SessionStateMachine_FullLifecycle(t *testing.T) {
	steps := []struct {
		current session.Status
		event   session.Event
		expect  session.Status
	}{
		{session.StatusConnecting, session.EventHandshakeOK, session.StatusActive},
		{session.StatusActive, session.EventPacketReceived, session.StatusActive},
		{session.StatusActive, session.EventIdleTimeout, session.StatusSleeping},
		{session.StatusSleeping, session.EventPacketReceived, session.StatusActive},
		{session.StatusActive, session.EventExplicitLogout, session.StatusExpired},
	}

	for i, s := range steps {
		next, _, err := session.Transition(s.current, s.event)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		if next != s.expect {
			t.Fatalf("step %d: expected %v, got %v", i, s.expect, next)
		}
	}
}

func TestStandard_SessionDefaultConfig(t *testing.T) {
	cfg := session.DefaultConfig()

	if cfg.ExpiryTimeout <= 0 {
		t.Fatal("ExpiryTimeout must be positive")
	}
	if cfg.IdleTimeout <= 0 {
		t.Fatal("IdleTimeout must be positive")
	}
	if cfg.MaxConnections <= 0 {
		t.Fatal("MaxConnections must be positive")
	}
	if cfg.HandshakeTimeout <= 0 {
		t.Fatal("HandshakeTimeout must be positive")
	}
	if cfg.ExpiryTimeout < cfg.IdleTimeout {
		t.Fatal("ExpiryTimeout should be >= IdleTimeout")
	}
}

func TestStandard_SessionStatusValues(t *testing.T) {
	statuses := []session.Status{
		session.StatusConnecting,
		session.StatusActive,
		session.StatusSleeping,
		session.StatusExpired,
	}

	seen := make(map[session.Status]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Fatalf("duplicate status: %v", s)
		}
		seen[s] = true
	}
}
