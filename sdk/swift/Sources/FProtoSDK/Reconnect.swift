import Foundation

/// Configuration for automatic reconnection with exponential backoff.
public struct ReconnectConfig {
    public let initialDelay: TimeInterval  // seconds
    public let maxDelay: TimeInterval
    public let multiplier: Double
    public let jitter: Double  // ±fraction
    public let maxAttempts: Int  // 0 = unlimited
    
    public init(
        initialDelay: TimeInterval = 1.0,
        maxDelay: TimeInterval = 30.0,
        multiplier: Double = 2.0,
        jitter: Double = 0.2,
        maxAttempts: Int = 0
    ) {
        self.initialDelay = initialDelay
        self.maxDelay = maxDelay
        self.multiplier = multiplier
        self.jitter = jitter
        self.maxAttempts = maxAttempts
    }
    
    public static let `default` = ReconnectConfig()
}

/// Compute the delay for a given reconnection attempt.
public func computeDelay(attempt: Int, config: ReconnectConfig) -> TimeInterval {
    var delay = config.initialDelay * pow(config.multiplier, Double(attempt))
    delay = min(delay, config.maxDelay)
    if config.jitter > 0 {
        let delta = delay * config.jitter
        delay = delay - delta + Double.random(in: 0..<1) * 2 * delta
    }
    return delay
}

extension FProtoClient {
    /// Connect with automatic reconnection using exponential backoff.
    public func connectWithReconnect(config: ReconnectConfig = .default) async throws {
        var attempt = 0
        while true {
            do {
                try await connect()
                return
            } catch {
                attempt += 1
                if config.maxAttempts > 0 && attempt >= config.maxAttempts {
                    throw error
                }
                let delay = computeDelay(attempt: attempt - 1, config: config)
                try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
            }
        }
    }
}
