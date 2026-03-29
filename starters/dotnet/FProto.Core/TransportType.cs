namespace FProto.Core;

public enum TransportType
{
    WebSocket,
    Quic,
    Grpc
}

public interface IConn
{
    Task<byte[]> ReadMessageAsync(CancellationToken ct = default);
    Task WriteMessageAsync(byte[] data, CancellationToken ct = default);
    Task CloseAsync(CancellationToken ct = default);
    TransportType TransportType { get; }
    string RemoteAddr { get; }
}
