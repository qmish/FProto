package io.fproto.core;

public record SessionConfig(
    int idleTimeoutSecs,
    int expiryTimeoutSecs,
    int handshakeTimeoutSecs,
    int maxConnections
) {
    public static SessionConfig defaults() {
        return new SessionConfig(300, 86400, 30, 10000);
    }
}
