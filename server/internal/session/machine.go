package session

import (
	"fmt"
)

type Event int

const (
	EventHandshakeOK Event = iota
	EventHandshakeFail
	EventPacketReceived
	EventIdleTimeout
	EventExpiryTimeout
	EventExplicitLogout
)

func (e Event) String() string {
	switch e {
	case EventHandshakeOK:
		return "HandshakeOK"
	case EventHandshakeFail:
		return "HandshakeFail"
	case EventPacketReceived:
		return "PacketReceived"
	case EventIdleTimeout:
		return "IdleTimeout"
	case EventExpiryTimeout:
		return "ExpiryTimeout"
	case EventExplicitLogout:
		return "ExplicitLogout"
	default:
		return fmt.Sprintf("Event(%d)", e)
	}
}

// ErrInvalidTransition indicates an illegal state+event combination.
var ErrInvalidTransition = fmt.Errorf("invalid session state transition")

type transitionResult struct {
	next      Status
	destroyed bool
}

var transitions = map[Status]map[Event]transitionResult{
	StatusConnecting: {
		EventHandshakeOK:   {next: StatusActive},
		EventHandshakeFail: {destroyed: true},
	},
	StatusActive: {
		EventPacketReceived: {next: StatusActive},
		EventIdleTimeout:    {next: StatusSleeping},
		EventExplicitLogout: {next: StatusExpired},
	},
	StatusSleeping: {
		EventPacketReceived: {next: StatusActive},
		EventExpiryTimeout:  {next: StatusExpired},
	},
}

// Transition applies an event to the current session status and returns the
// new status. Returns (StatusExpired, true) when the session should be destroyed,
// or error if the transition is not allowed.
func Transition(current Status, event Event) (next Status, destroyed bool, err error) {
	events, ok := transitions[current]
	if !ok {
		return "", false, fmt.Errorf("%w: no transitions from status %q", ErrInvalidTransition, current)
	}

	result, ok := events[event]
	if !ok {
		return "", false, fmt.Errorf("%w: event %s not valid in status %q", ErrInvalidTransition, event, current)
	}

	return result.next, result.destroyed, nil
}
