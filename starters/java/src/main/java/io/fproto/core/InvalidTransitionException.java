package io.fproto.core;

public class InvalidTransitionException extends Exception {
    private final SessionStatus status;
    private final SessionEvent event;

    public InvalidTransitionException(SessionStatus status, SessionEvent event) {
        super("invalid transition: " + status + " + " + event);
        this.status = status;
        this.event = event;
    }

    public SessionStatus getStatus() {
        return status;
    }

    public SessionEvent getEvent() {
        return event;
    }
}
