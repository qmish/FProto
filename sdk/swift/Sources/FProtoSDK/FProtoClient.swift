import Foundation
import Starscream

/// FProto Swift SDK client with Starscream WebSocket transport.
public class FProtoClient: NSObject {
    private let config: ClientConfig
    private var socket: WebSocket?
    private var messageQueue: [Data] = []
    private let lock = NSLock()
    private var continuations: [CheckedContinuation<Data, Error>] = []
    
    public private(set) var isConnected: Bool = false
    
    public init(config: ClientConfig) {
        self.config = config
    }
    
    /// Connect to the server via WebSocket.
    public func connect() async throws {
        guard let url = URL(string: config.serverUrl) else {
            throw FProtoError.invalidURL
        }
        
        var request = URLRequest(url: url)
        request.timeoutInterval = 30
        
        let socket = WebSocket(request: request)
        self.socket = socket
        
        return try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
            socket.onEvent = { [weak self] event in
                guard let self = self else { return }
                switch event {
                case .connected:
                    self.isConnected = true
                    continuation.resume()
                case .binary(let data):
                    self.handleMessage(data)
                case .text(let text):
                    self.handleMessage(Data(text.utf8))
                case .disconnected(_, _):
                    self.isConnected = false
                case .error(let error):
                    self.isConnected = false
                    if let error = error {
                        continuation.resume(throwing: error)
                    }
                default:
                    break
                }
            }
            socket.connect()
        }
    }
    
    /// Send binary data.
    public func send(_ data: Data) throws {
        guard let socket = socket, isConnected else {
            throw FProtoError.notConnected
        }
        socket.write(data: data)
    }
    
    /// Receive binary data.
    public func receive() async throws -> Data {
        lock.lock()
        if !messageQueue.isEmpty {
            let data = messageQueue.removeFirst()
            lock.unlock()
            return data
        }
        lock.unlock()
        
        return try await withCheckedThrowingContinuation { continuation in
            lock.lock()
            continuations.append(continuation)
            lock.unlock()
        }
    }
    
    /// Close the connection.
    public func close() {
        socket?.disconnect()
        socket = nil
        isConnected = false
    }
    
    private func handleMessage(_ data: Data) {
        lock.lock()
        if !continuations.isEmpty {
            let cont = continuations.removeFirst()
            lock.unlock()
            cont.resume(returning: data)
        } else {
            messageQueue.append(data)
            lock.unlock()
        }
    }
}

public struct ClientConfig {
    public let serverUrl: String
    
    public init(serverUrl: String) {
        self.serverUrl = serverUrl
    }
}

public enum FProtoError: Error {
    case invalidURL
    case notConnected
    case receiveTimeout
}
