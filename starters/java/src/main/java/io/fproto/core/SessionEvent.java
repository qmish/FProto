package io.fproto.core;

public enum SessionEvent {
    HANDSHAKE_OK("handshake_ok"),
    HANDSHAKE_FAIL("handshake_fail"),
    PACKET_RECEIVED("packet_received"),
    IDLE_TIMEOUT("idle_timeout"),
    EXPIRY_TIMEOUT("expiry_timeout"),
    EXPLICIT_LOGOUT("explicit_logout");

    private final String value;

    SessionEvent(String value) {
        this.value = value;
    }

    public String getValue() {
        return value;
    }

    @Override
    public String toString() {
        return value;
    }
}
