"""Session layer — state machine."""

from __future__ import annotations

import enum
from dataclasses import dataclass, field


class Status(enum.Enum):
    CONNECTING = "connecting"
    ACTIVE = "active"
    SLEEPING = "sleeping"
    EXPIRED = "expired"


class Event(enum.Enum):
    HANDSHAKE_OK = "handshake_ok"
    HANDSHAKE_FAIL = "handshake_fail"
    PACKET_RECEIVED = "packet_received"
    IDLE_TIMEOUT = "idle_timeout"
    EXPIRY_TIMEOUT = "expiry_timeout"
    EXPLICIT_LOGOUT = "explicit_logout"


class InvalidTransition(Exception):
    def __init__(self, status: Status, event: Event) -> None:
        self.status = status
        self.event = event
        super().__init__(f"invalid transition: {status.value} + {event.value}")


@dataclass
class TransitionResult:
    next_status: Status
    destroyed: bool


_TRANSITIONS: dict[tuple[Status, Event], TransitionResult] = {
    (Status.CONNECTING, Event.HANDSHAKE_OK): TransitionResult(Status.ACTIVE, False),
    (Status.CONNECTING, Event.HANDSHAKE_FAIL): TransitionResult(Status.EXPIRED, True),
    (Status.ACTIVE, Event.PACKET_RECEIVED): TransitionResult(Status.ACTIVE, False),
    (Status.ACTIVE, Event.IDLE_TIMEOUT): TransitionResult(Status.SLEEPING, False),
    (Status.ACTIVE, Event.EXPLICIT_LOGOUT): TransitionResult(Status.EXPIRED, False),
    (Status.SLEEPING, Event.PACKET_RECEIVED): TransitionResult(Status.ACTIVE, False),
    (Status.SLEEPING, Event.EXPIRY_TIMEOUT): TransitionResult(Status.EXPIRED, False),
}


def transition(current: Status, event: Event) -> TransitionResult:
    """Perform a state machine transition. Raises InvalidTransition on illegal state+event."""
    key = (current, event)
    if key not in _TRANSITIONS:
        raise InvalidTransition(current, event)
    result = _TRANSITIONS[key]
    return TransitionResult(next_status=result.next_status, destroyed=result.destroyed)


@dataclass
class Config:
    idle_timeout_secs: int = 300
    expiry_timeout_secs: int = 86400
    handshake_timeout_secs: int = 30
    max_connections: int = 10000
