import XCTest
@testable import FProtoSDK

final class ReconnectTests: XCTestCase {
    
    func testDefaultConfig() {
        let cfg = ReconnectConfig.default
        XCTAssertEqual(cfg.initialDelay, 1.0, accuracy: 0.01)
        XCTAssertEqual(cfg.maxDelay, 30.0, accuracy: 0.01)
        XCTAssertEqual(cfg.multiplier, 2.0, accuracy: 0.01)
        XCTAssertEqual(cfg.jitter, 0.2, accuracy: 0.01)
        XCTAssertEqual(cfg.maxAttempts, 0)
    }
    
    func testComputeDelayAttemptZero() {
        let cfg = ReconnectConfig.default
        let delay = computeDelay(attempt: 0, config: cfg)
        XCTAssertGreaterThan(delay, 0.8)
        XCTAssertLessThan(delay, 1.2)
    }
    
    func testComputeDelayCappedAtMax() {
        let cfg = ReconnectConfig.default
        let delay = computeDelay(attempt: 10, config: cfg)
        XCTAssertLessThanOrEqual(delay, 36.0)
    }
    
    func testComputeDelayIncreases() {
        let cfg = ReconnectConfig(jitter: 0.0)
        let d0 = computeDelay(attempt: 0, config: cfg)
        let d1 = computeDelay(attempt: 1, config: cfg)
        let d2 = computeDelay(attempt: 2, config: cfg)
        XCTAssertGreaterThan(d1, d0)
        XCTAssertGreaterThan(d2, d1)
    }
    
    func testClientInitialState() {
        let client = FProtoClient(config: ClientConfig(serverUrl: "ws://localhost:1"))
        XCTAssertFalse(client.isConnected)
    }
}
