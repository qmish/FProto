package session

import (
	"errors"
	"testing"
)

func TestTransition_ConnectingToActive(t *testing.T) {
	next, destroyed, err := Transition(StatusConnecting, EventHandshakeOK)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusActive {
		t.Fatalf("expected Active, got %s", next)
	}
}

func TestTransition_ConnectingFail(t *testing.T) {
	_, destroyed, err := Transition(StatusConnecting, EventHandshakeFail)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !destroyed {
		t.Fatal("should be destroyed on handshake fail")
	}
}

func TestTransition_ActivePacket(t *testing.T) {
	next, destroyed, err := Transition(StatusActive, EventPacketReceived)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusActive {
		t.Fatalf("expected Active, got %s", next)
	}
}

func TestTransition_ActiveToSleeping(t *testing.T) {
	next, destroyed, err := Transition(StatusActive, EventIdleTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusSleeping {
		t.Fatalf("expected Sleeping, got %s", next)
	}
}

func TestTransition_ActiveToExpired(t *testing.T) {
	next, destroyed, err := Transition(StatusActive, EventExplicitLogout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusExpired {
		t.Fatalf("expected Expired, got %s", next)
	}
}

func TestTransition_SleepingToActive(t *testing.T) {
	next, destroyed, err := Transition(StatusSleeping, EventPacketReceived)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusActive {
		t.Fatalf("expected Active, got %s", next)
	}
}

func TestTransition_SleepingToExpired(t *testing.T) {
	next, destroyed, err := Transition(StatusSleeping, EventExpiryTimeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destroyed {
		t.Fatal("should not be destroyed")
	}
	if next != StatusExpired {
		t.Fatalf("expected Expired, got %s", next)
	}
}

func TestTransition_InvalidFromExpired(t *testing.T) {
	_, _, err := Transition(StatusExpired, EventPacketReceived)
	if err == nil {
		t.Fatal("expected error for transition from Expired")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestTransition_InvalidEventInConnecting(t *testing.T) {
	_, _, err := Transition(StatusConnecting, EventPacketReceived)
	if err == nil {
		t.Fatal("expected error for PacketReceived in Connecting")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestTransition_InvalidEventInActive(t *testing.T) {
	_, _, err := Transition(StatusActive, EventHandshakeOK)
	if err == nil {
		t.Fatal("expected error for HandshakeOK in Active")
	}
}

func TestTransition_InvalidEventInSleeping(t *testing.T) {
	_, _, err := Transition(StatusSleeping, EventIdleTimeout)
	if err == nil {
		t.Fatal("expected error for IdleTimeout in Sleeping")
	}
}

func TestTransition_FullLifecycle(t *testing.T) {
	status := StatusConnecting

	next, destroyed, err := Transition(status, EventHandshakeOK)
	if err != nil || destroyed {
		t.Fatalf("step 1: %v, destroyed=%v", err, destroyed)
	}
	status = next

	next, destroyed, err = Transition(status, EventPacketReceived)
	if err != nil || destroyed || next != StatusActive {
		t.Fatalf("step 2: %v, destroyed=%v, next=%s", err, destroyed, next)
	}
	status = next

	next, _, err = Transition(status, EventIdleTimeout)
	if err != nil || next != StatusSleeping {
		t.Fatalf("step 3: %v, next=%s", err, next)
	}
	status = next

	next, _, err = Transition(status, EventPacketReceived)
	if err != nil || next != StatusActive {
		t.Fatalf("step 4: %v, next=%s", err, next)
	}
	status = next

	next, _, err = Transition(status, EventIdleTimeout)
	if err != nil {
		t.Fatalf("step 5: %v", err)
	}
	status = next

	next, _, err = Transition(status, EventExpiryTimeout)
	if err != nil || next != StatusExpired {
		t.Fatalf("step 6: %v, next=%s", err, next)
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		event Event
		want  string
	}{
		{EventHandshakeOK, "HandshakeOK"},
		{EventHandshakeFail, "HandshakeFail"},
		{EventPacketReceived, "PacketReceived"},
		{EventIdleTimeout, "IdleTimeout"},
		{EventExpiryTimeout, "ExpiryTimeout"},
		{EventExplicitLogout, "ExplicitLogout"},
		{Event(99), "Event(99)"},
	}
	for _, tt := range tests {
		if got := tt.event.String(); got != tt.want {
			t.Errorf("Event(%d).String() = %q, want %q", tt.event, got, tt.want)
		}
	}
}
