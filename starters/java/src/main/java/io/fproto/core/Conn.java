package io.fproto.core;

public interface Conn {
    byte[] readMessage() throws Exception;
    void writeMessage(byte[] data) throws Exception;
    void close() throws Exception;
    TransportType transportType();
    String remoteAddr();
}
