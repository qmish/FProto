<?php

declare(strict_types=1);

namespace FProto\Tests;

use FProto\Crypto\CryptoUtils;
use FProto\Session\InvalidTransitionException;
use FProto\Session\SessionConfig;
use FProto\Session\SessionEvent;
use FProto\Session\SessionMachine;
use FProto\Session\SessionStatus;
use FProto\Transport\TransportType;
use PHPUnit\Framework\TestCase;

final class ConformanceTest extends TestCase
{
    private function testKey(): string
    {
        $key = '';
        for ($i = 0; $i < 32; $i++) {
            $key .= chr($i);
        }
        return $key;
    }

    // --- Crypto: AEAD ---

    public function testAeadEncryptDecrypt(): void
    {
        $key = $this->testKey();
        $sid = 'test-session';
        $plaintext = 'secret data';

        $ct = CryptoUtils::encryptAead($key, $sid, 1, $plaintext);
        $pt = CryptoUtils::decryptAead($key, $sid, 1, $ct);
        $this->assertSame($plaintext, $pt);
    }

    public function testAeadWrongSeqFails(): void
    {
        $key = $this->testKey();
        $ct = CryptoUtils::encryptAead($key, 'session', 1, 'data');
        $this->expectException(\RuntimeException::class);
        CryptoUtils::decryptAead($key, 'session', 2, $ct);
    }

    public function testAeadWrongSessionFails(): void
    {
        $key = $this->testKey();
        $ct = CryptoUtils::encryptAead($key, 'session-a', 1, 'data');
        $this->expectException(\RuntimeException::class);
        CryptoUtils::decryptAead($key, 'session-b', 1, $ct);
    }

    public function testAeadTamperDetection(): void
    {
        $key = $this->testKey();
        $ct = CryptoUtils::encryptAead($key, 'session', 1, 'data');
        $ct[0] = chr(ord($ct[0]) ^ 0xFF);
        $this->expectException(\RuntimeException::class);
        CryptoUtils::decryptAead($key, 'session', 1, $ct);
    }

    public function testLargePayloadAead(): void
    {
        $key = $this->testKey();
        $payload = '';
        for ($i = 0; $i < 32768; $i++) {
            $payload .= chr($i % 256);
        }

        $ct = CryptoUtils::encryptAead($key, 'large', 1, $payload);
        $pt = CryptoUtils::decryptAead($key, 'large', 1, $ct);
        $this->assertSame($payload, $pt);
    }

    public function testSequentialAead(): void
    {
        $key = $this->testKey();
        for ($seq = 0; $seq < 100; $seq++) {
            $msg = "message-{$seq}";
            $ct = CryptoUtils::encryptAead($key, 'seq-session', $seq, $msg);
            $pt = CryptoUtils::decryptAead($key, 'seq-session', $seq, $ct);
            $this->assertSame($msg, $pt);
        }
    }

    public function testInvalidKeySize(): void
    {
        $this->expectException(\InvalidArgumentException::class);
        CryptoUtils::encryptAead(str_repeat("\0", 16), 's', 0, 'd');
    }

    // --- Crypto: Nonce ---

    public function testNonceUniqueness(): void
    {
        $seen = [];
        for ($i = 0; $i < 1000; $i++) {
            $n = CryptoUtils::buildNonce($i);
            $this->assertArrayNotHasKey($n, $seen, "duplicate nonce at seq {$i}");
            $seen[$n] = true;
        }
    }

    public function testNonceStructure(): void
    {
        $n0 = CryptoUtils::buildNonce(0);
        $this->assertSame(str_repeat("\0", 12), $n0);
    }

    // --- Crypto: AAD ---

    public function testBuildAad(): void
    {
        $sid = 'session-1';
        $aad = CryptoUtils::buildAad($sid, 42);
        $this->assertSame(strlen($sid) + 8, strlen($aad));
        $this->assertSame($sid, substr($aad, 0, strlen($sid)));
    }

    // --- Crypto: Rekey ---

    public function testRekeyDeterministic(): void
    {
        $key = $this->testKey();
        $k1 = CryptoUtils::rekey($key);
        $k2 = CryptoUtils::rekey($key);
        $this->assertSame($k1, $k2);
        $this->assertNotSame($key, $k1);
        $this->assertSame(32, strlen($k1));
    }

    public function testRekeyChain(): void
    {
        $key = $this->testKey();
        for ($i = 0; $i < 10; $i++) {
            $newKey = CryptoUtils::rekey($key);
            $this->assertNotSame($key, $newKey);
            $this->assertSame(32, strlen($newKey));
            $key = $newKey;
        }
    }

    // --- Crypto: Zeroize ---

    public function testZeroize(): void
    {
        $buf = str_repeat("\xAA", 64);
        CryptoUtils::zeroize($buf);
        $this->assertSame(str_repeat("\0", 64), $buf);
    }

    // --- Session: State Machine ---

    public function testConnectingHandshakeOk(): void
    {
        $r = SessionMachine::transition(SessionStatus::Connecting, SessionEvent::HandshakeOK);
        $this->assertSame(SessionStatus::Active, $r->next);
        $this->assertFalse($r->destroyed);
    }

    public function testConnectingHandshakeFail(): void
    {
        $r = SessionMachine::transition(SessionStatus::Connecting, SessionEvent::HandshakeFail);
        $this->assertTrue($r->destroyed);
    }

    public function testActiveIdleTimeout(): void
    {
        $r = SessionMachine::transition(SessionStatus::Active, SessionEvent::IdleTimeout);
        $this->assertSame(SessionStatus::Sleeping, $r->next);
        $this->assertFalse($r->destroyed);
    }

    public function testActivePacketReceived(): void
    {
        $r = SessionMachine::transition(SessionStatus::Active, SessionEvent::PacketReceived);
        $this->assertSame(SessionStatus::Active, $r->next);
    }

    public function testActiveExplicitLogout(): void
    {
        $r = SessionMachine::transition(SessionStatus::Active, SessionEvent::ExplicitLogout);
        $this->assertSame(SessionStatus::Expired, $r->next);
    }

    public function testSleepingPacketReceived(): void
    {
        $r = SessionMachine::transition(SessionStatus::Sleeping, SessionEvent::PacketReceived);
        $this->assertSame(SessionStatus::Active, $r->next);
    }

    public function testSleepingExpiryTimeout(): void
    {
        $r = SessionMachine::transition(SessionStatus::Sleeping, SessionEvent::ExpiryTimeout);
        $this->assertSame(SessionStatus::Expired, $r->next);
        $this->assertFalse($r->destroyed);
    }

    public function testInvalidTransitions(): void
    {
        $this->expectException(InvalidTransitionException::class);
        SessionMachine::transition(SessionStatus::Expired, SessionEvent::PacketReceived);
    }

    public function testInvalidTransitionConnectingIdle(): void
    {
        $this->expectException(InvalidTransitionException::class);
        SessionMachine::transition(SessionStatus::Connecting, SessionEvent::IdleTimeout);
    }

    public function testFullLifecycle(): void
    {
        $r = SessionMachine::transition(SessionStatus::Connecting, SessionEvent::HandshakeOK);
        $this->assertSame(SessionStatus::Active, $r->next);

        $r = SessionMachine::transition($r->next, SessionEvent::PacketReceived);
        $this->assertSame(SessionStatus::Active, $r->next);

        $r = SessionMachine::transition($r->next, SessionEvent::IdleTimeout);
        $this->assertSame(SessionStatus::Sleeping, $r->next);

        $r = SessionMachine::transition($r->next, SessionEvent::PacketReceived);
        $this->assertSame(SessionStatus::Active, $r->next);

        $r = SessionMachine::transition($r->next, SessionEvent::IdleTimeout);
        $r = SessionMachine::transition($r->next, SessionEvent::ExpiryTimeout);
        $this->assertSame(SessionStatus::Expired, $r->next);
    }

    // --- Session: Config ---

    public function testDefaultConfig(): void
    {
        $cfg = new SessionConfig();
        $this->assertSame(300, $cfg->idleTimeoutSecs);
        $this->assertSame(86400, $cfg->expiryTimeoutSecs);
        $this->assertSame(30, $cfg->handshakeTimeoutSecs);
        $this->assertSame(10000, $cfg->maxConnections);
    }

    // --- Transport ---

    public function testTransportTypes(): void
    {
        $this->assertSame('websocket', TransportType::WebSocket->value);
        $this->assertSame('quic', TransportType::Quic->value);
        $this->assertSame('grpc', TransportType::Grpc->value);
    }
}
