package io.fproto.core;

public enum TransportType {
    WEBSOCKET("websocket"),
    QUIC("quic"),
    GRPC("grpc");

    private final String value;

    TransportType(String value) {
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
