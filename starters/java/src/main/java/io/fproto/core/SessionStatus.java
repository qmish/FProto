package io.fproto.core;

public enum SessionStatus {
    CONNECTING("connecting"),
    ACTIVE("active"),
    SLEEPING("sleeping"),
    EXPIRED("expired");

    private final String value;

    SessionStatus(String value) {
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
