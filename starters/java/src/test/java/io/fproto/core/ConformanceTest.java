package io.fproto.core;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.Arrays;
import java.util.HashSet;

class ConformanceTest {

    // --- Crypto: AEAD ---

    @Test
    void testAeadEncryptDecrypt() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;
        byte[] sid = "test-session".getBytes();
        byte[] plaintext = "secret data".getBytes();

        byte[] ct = CryptoUtils.encryptAead(key, sid, 1, plaintext);
        byte[] pt = CryptoUtils.decryptAead(key, sid, 1, ct);
        assertArrayEquals(plaintext, pt);
    }

    @Test
    void testAeadWrongSeqFails() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;
        byte[] sid = "session".getBytes();

        byte[] ct = CryptoUtils.encryptAead(key, sid, 1, "data".getBytes());
        assertThrows(Exception.class, () -> CryptoUtils.decryptAead(key, sid, 2, ct));
    }

    @Test
    void testAeadWrongSessionFails() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;

        byte[] ct = CryptoUtils.encryptAead(key, "session-a".getBytes(), 1, "data".getBytes());
        assertThrows(Exception.class, () -> CryptoUtils.decryptAead(key, "session-b".getBytes(), 1, ct));
    }

    @Test
    void testAeadTamperDetection() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;
        byte[] sid = "session".getBytes();

        byte[] ct = CryptoUtils.encryptAead(key, sid, 1, "data".getBytes());
        ct[0] ^= (byte) 0xFF;
        assertThrows(Exception.class, () -> CryptoUtils.decryptAead(key, sid, 1, ct));
    }

    @Test
    void testLargePayloadAead() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;
        byte[] sid = "large".getBytes();

        byte[] payload = new byte[32768];
        for (int i = 0; i < payload.length; i++) payload[i] = (byte) (i % 256);

        byte[] ct = CryptoUtils.encryptAead(key, sid, 1, payload);
        byte[] pt = CryptoUtils.decryptAead(key, sid, 1, ct);
        assertArrayEquals(payload, pt);
    }

    @Test
    void testSequentialAead() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;
        byte[] sid = "seq-session".getBytes();

        for (int seq = 0; seq < 100; seq++) {
            byte[] msg = ("message-" + seq).getBytes();
            byte[] ct = CryptoUtils.encryptAead(key, sid, seq, msg);
            byte[] pt = CryptoUtils.decryptAead(key, sid, seq, ct);
            assertArrayEquals(msg, pt);
        }
    }

    @Test
    void testInvalidKeySize() {
        byte[] shortKey = new byte[16];
        assertThrows(IllegalArgumentException.class,
            () -> CryptoUtils.encryptAead(shortKey, "s".getBytes(), 0, "d".getBytes()));
    }

    // --- Crypto: Nonce ---

    @Test
    void testNonceUniqueness() {
        var seen = new HashSet<String>();
        for (int i = 0; i < 1000; i++) {
            byte[] n = CryptoUtils.buildNonce(i);
            assertTrue(seen.add(Arrays.toString(n)), "duplicate nonce at seq " + i);
        }
    }

    @Test
    void testNonceStructure() {
        byte[] n = CryptoUtils.buildNonce(0);
        assertArrayEquals(new byte[12], n);

        byte[] n1 = CryptoUtils.buildNonce(1);
        assertArrayEquals(new byte[4], Arrays.copyOf(n1, 4));
    }

    // --- Crypto: AAD ---

    @Test
    void testBuildAad() {
        byte[] sid = "session-1".getBytes();
        byte[] aad = CryptoUtils.buildAad(sid, 42);
        assertEquals(sid.length + 8, aad.length);
        assertArrayEquals(sid, Arrays.copyOf(aad, sid.length));
    }

    // --- Crypto: Rekey ---

    @Test
    void testRekeyDeterministic() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;

        byte[] k1 = CryptoUtils.rekey(key);
        byte[] k2 = CryptoUtils.rekey(key);
        assertArrayEquals(k1, k2);
        assertFalse(Arrays.equals(k1, key));
        assertEquals(32, k1.length);
    }

    @Test
    void testRekeyChain() throws Exception {
        byte[] key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte) i;

        for (int j = 0; j < 10; j++) {
            byte[] newKey = CryptoUtils.rekey(key);
            assertFalse(Arrays.equals(newKey, key));
            assertEquals(32, newKey.length);
            key = newKey;
        }
    }

    // --- Crypto: Zeroize ---

    @Test
    void testZeroize() {
        byte[] buf = new byte[]{(byte) 0xAA, (byte) 0xBB, (byte) 0xCC};
        CryptoUtils.zeroize(buf);
        for (byte b : buf) {
            assertEquals(0, b);
        }
    }

    // --- Session: State Machine ---

    @Test
    void testConnectingHandshakeOk() throws Exception {
        var r = SessionMachine.transition(SessionStatus.CONNECTING, SessionEvent.HANDSHAKE_OK);
        assertEquals(SessionStatus.ACTIVE, r.next());
        assertFalse(r.destroyed());
    }

    @Test
    void testConnectingHandshakeFail() throws Exception {
        var r = SessionMachine.transition(SessionStatus.CONNECTING, SessionEvent.HANDSHAKE_FAIL);
        assertTrue(r.destroyed());
    }

    @Test
    void testActiveIdleTimeout() throws Exception {
        var r = SessionMachine.transition(SessionStatus.ACTIVE, SessionEvent.IDLE_TIMEOUT);
        assertEquals(SessionStatus.SLEEPING, r.next());
        assertFalse(r.destroyed());
    }

    @Test
    void testActivePacketReceived() throws Exception {
        var r = SessionMachine.transition(SessionStatus.ACTIVE, SessionEvent.PACKET_RECEIVED);
        assertEquals(SessionStatus.ACTIVE, r.next());
    }

    @Test
    void testActiveExplicitLogout() throws Exception {
        var r = SessionMachine.transition(SessionStatus.ACTIVE, SessionEvent.EXPLICIT_LOGOUT);
        assertEquals(SessionStatus.EXPIRED, r.next());
    }

    @Test
    void testSleepingPacketReceived() throws Exception {
        var r = SessionMachine.transition(SessionStatus.SLEEPING, SessionEvent.PACKET_RECEIVED);
        assertEquals(SessionStatus.ACTIVE, r.next());
    }

    @Test
    void testSleepingExpiryTimeout() throws Exception {
        var r = SessionMachine.transition(SessionStatus.SLEEPING, SessionEvent.EXPIRY_TIMEOUT);
        assertEquals(SessionStatus.EXPIRED, r.next());
        assertFalse(r.destroyed());
    }

    @Test
    void testInvalidTransitions() {
        assertThrows(InvalidTransitionException.class,
            () -> SessionMachine.transition(SessionStatus.EXPIRED, SessionEvent.PACKET_RECEIVED));
        assertThrows(InvalidTransitionException.class,
            () -> SessionMachine.transition(SessionStatus.CONNECTING, SessionEvent.IDLE_TIMEOUT));
        assertThrows(InvalidTransitionException.class,
            () -> SessionMachine.transition(SessionStatus.SLEEPING, SessionEvent.HANDSHAKE_OK));
    }

    @Test
    void testFullLifecycle() throws Exception {
        var r = SessionMachine.transition(SessionStatus.CONNECTING, SessionEvent.HANDSHAKE_OK);
        assertEquals(SessionStatus.ACTIVE, r.next());

        r = SessionMachine.transition(r.next(), SessionEvent.PACKET_RECEIVED);
        assertEquals(SessionStatus.ACTIVE, r.next());

        r = SessionMachine.transition(r.next(), SessionEvent.IDLE_TIMEOUT);
        assertEquals(SessionStatus.SLEEPING, r.next());

        r = SessionMachine.transition(r.next(), SessionEvent.PACKET_RECEIVED);
        assertEquals(SessionStatus.ACTIVE, r.next());

        r = SessionMachine.transition(r.next(), SessionEvent.IDLE_TIMEOUT);
        r = SessionMachine.transition(r.next(), SessionEvent.EXPIRY_TIMEOUT);
        assertEquals(SessionStatus.EXPIRED, r.next());
    }

    // --- Session: Config ---

    @Test
    void testDefaultConfig() {
        var cfg = SessionConfig.defaults();
        assertEquals(300, cfg.idleTimeoutSecs());
        assertEquals(86400, cfg.expiryTimeoutSecs());
        assertEquals(30, cfg.handshakeTimeoutSecs());
        assertEquals(10000, cfg.maxConnections());
    }

    // --- Transport ---

    @Test
    void testTransportTypes() {
        assertEquals("websocket", TransportType.WEBSOCKET.getValue());
        assertEquals("quic", TransportType.QUIC.getValue());
        assertEquals("grpc", TransportType.GRPC.getValue());
    }
}
