package io.fproto.core;

import java.util.Map;

public final class SessionMachine {

    private SessionMachine() {}

    private record Key(SessionStatus status, SessionEvent event) {}

    private static final Map<Key, TransitionResult> TRANSITIONS = Map.of(
        new Key(SessionStatus.CONNECTING, SessionEvent.HANDSHAKE_OK),
            new TransitionResult(SessionStatus.ACTIVE, false),
        new Key(SessionStatus.CONNECTING, SessionEvent.HANDSHAKE_FAIL),
            new TransitionResult(SessionStatus.EXPIRED, true),
        new Key(SessionStatus.ACTIVE, SessionEvent.PACKET_RECEIVED),
            new TransitionResult(SessionStatus.ACTIVE, false),
        new Key(SessionStatus.ACTIVE, SessionEvent.IDLE_TIMEOUT),
            new TransitionResult(SessionStatus.SLEEPING, false),
        new Key(SessionStatus.ACTIVE, SessionEvent.EXPLICIT_LOGOUT),
            new TransitionResult(SessionStatus.EXPIRED, false),
        new Key(SessionStatus.SLEEPING, SessionEvent.PACKET_RECEIVED),
            new TransitionResult(SessionStatus.ACTIVE, false),
        new Key(SessionStatus.SLEEPING, SessionEvent.EXPIRY_TIMEOUT),
            new TransitionResult(SessionStatus.EXPIRED, false)
    );

    public static TransitionResult transition(SessionStatus current, SessionEvent event)
            throws InvalidTransitionException {
        var key = new Key(current, event);
        var result = TRANSITIONS.get(key);
        if (result == null) {
            throw new InvalidTransitionException(current, event);
        }
        return result;
    }
}
