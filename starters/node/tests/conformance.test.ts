import { describe, it, expect } from 'vitest';
import {
  generateKeyPair,
  newAead,
  buildNonce,
  buildAad,
  encryptAead,
  decryptAead,
  rekey,
  zeroize,
} from '../src/crypto';
import {
  Status,
  Event,
  transition,
  InvalidTransitionError,
  defaultConfig,
} from '../src/session';
import { TransportType } from '../src/transport';

// --- Crypto: KeyPair ---

describe('KeyPair', () => {
  it('generates key pairs', () => {
    const kp = generateKeyPair();
    expect(kp.privateKey.length).toBeGreaterThan(0);
    expect(kp.publicKey.length).toBeGreaterThan(0);
  });

  it('generates unique key pairs', () => {
    const kp1 = generateKeyPair();
    const kp2 = generateKeyPair();
    expect(kp1.publicKey.equals(kp2.publicKey)).toBe(false);
  });
});

// --- Crypto: AEAD ---

describe('AEAD', () => {
  const key = Buffer.from(Array.from({ length: 32 }, (_, i) => i));
  const sid = Buffer.from('test-session');

  it('encrypts and decrypts', () => {
    const cipher = newAead(key);
    const plaintext = Buffer.from('secret data');
    const ct = encryptAead(cipher, sid, 1, plaintext);
    const pt = decryptAead(cipher, sid, 1, ct);
    expect(pt.equals(plaintext)).toBe(true);
  });

  it('fails with wrong seq', () => {
    const cipher = newAead(key);
    const ct = encryptAead(cipher, sid, 1, Buffer.from('data'));
    expect(() => decryptAead(cipher, sid, 2, ct)).toThrow();
  });

  it('fails with wrong session', () => {
    const cipher = newAead(key);
    const ct = encryptAead(cipher, Buffer.from('session-a'), 1, Buffer.from('data'));
    expect(() => decryptAead(cipher, Buffer.from('session-b'), 1, ct)).toThrow();
  });

  it('detects tampering', () => {
    const cipher = newAead(key);
    const ct = Buffer.from(encryptAead(cipher, sid, 1, Buffer.from('data')));
    ct[0] ^= 0xff;
    expect(() => decryptAead(cipher, sid, 1, ct)).toThrow();
  });

  it('handles large payloads', () => {
    const cipher = newAead(key);
    const payload = Buffer.alloc(32768);
    for (let i = 0; i < payload.length; i++) payload[i] = i % 256;
    const ct = encryptAead(cipher, sid, 1, payload);
    const pt = decryptAead(cipher, sid, 1, ct);
    expect(pt.equals(payload)).toBe(true);
  });

  it('handles sequential messages', () => {
    const cipher = newAead(key);
    for (let seq = 0; seq < 100; seq++) {
      const msg = Buffer.from(`message-${seq}`);
      const ct = encryptAead(cipher, sid, seq, msg);
      const pt = decryptAead(cipher, sid, seq, ct);
      expect(pt.equals(msg)).toBe(true);
    }
  });

  it('rejects invalid key size', () => {
    expect(() => newAead(Buffer.alloc(16))).toThrow('invalid key size');
  });
});

// --- Crypto: Nonce ---

describe('Nonce', () => {
  it('produces unique nonces', () => {
    const seen = new Set<string>();
    for (let i = 0; i < 1000; i++) {
      const n = buildNonce(i).toString('hex');
      expect(seen.has(n)).toBe(false);
      seen.add(n);
    }
  });

  it('has correct structure', () => {
    const n0 = buildNonce(0);
    expect(n0.equals(Buffer.alloc(12))).toBe(true);

    const n1 = buildNonce(1);
    expect(n1.subarray(0, 4).equals(Buffer.alloc(4))).toBe(true);
    const expected = Buffer.alloc(8);
    expected.writeBigUInt64LE(1n);
    expect(n1.subarray(4).equals(expected)).toBe(true);
  });
});

// --- Crypto: AAD ---

describe('AAD', () => {
  it('has correct structure', () => {
    const sid = Buffer.from('session-1');
    const aad = buildAad(sid, 42);
    expect(aad.length).toBe(sid.length + 8);
    expect(aad.subarray(0, sid.length).equals(sid)).toBe(true);
  });
});

// --- Crypto: Rekey ---

describe('Rekey', () => {
  const key = Buffer.from(Array.from({ length: 32 }, (_, i) => i));

  it('is deterministic', () => {
    const k1 = rekey(key);
    const k2 = rekey(key);
    expect(k1.equals(k2)).toBe(true);
    expect(k1.equals(key)).toBe(false);
    expect(k1.length).toBe(32);
  });

  it('chains correctly', () => {
    let k = Buffer.from(key);
    for (let i = 0; i < 10; i++) {
      const newKey = rekey(k);
      expect(newKey.equals(k)).toBe(false);
      expect(newKey.length).toBe(32);
      k = newKey;
    }
  });
});

// --- Crypto: Zeroize ---

describe('Zeroize', () => {
  it('zeros buffer', () => {
    const buf = Buffer.from([0xaa, 0xbb, 0xcc, 0xdd]);
    zeroize(buf);
    expect(buf.every((b) => b === 0)).toBe(true);
  });
});

// --- Session: State Machine ---

describe('Session State Machine', () => {
  it('connecting -> handshake_ok -> active', () => {
    const r = transition(Status.Connecting, Event.HandshakeOK);
    expect(r.next).toBe(Status.Active);
    expect(r.destroyed).toBe(false);
  });

  it('connecting -> handshake_fail -> destroyed', () => {
    const r = transition(Status.Connecting, Event.HandshakeFail);
    expect(r.destroyed).toBe(true);
  });

  it('active -> idle_timeout -> sleeping', () => {
    const r = transition(Status.Active, Event.IdleTimeout);
    expect(r.next).toBe(Status.Sleeping);
    expect(r.destroyed).toBe(false);
  });

  it('active -> packet_received -> active', () => {
    const r = transition(Status.Active, Event.PacketReceived);
    expect(r.next).toBe(Status.Active);
  });

  it('active -> explicit_logout -> expired', () => {
    const r = transition(Status.Active, Event.ExplicitLogout);
    expect(r.next).toBe(Status.Expired);
  });

  it('sleeping -> packet_received -> active', () => {
    const r = transition(Status.Sleeping, Event.PacketReceived);
    expect(r.next).toBe(Status.Active);
  });

  it('sleeping -> expiry_timeout -> expired', () => {
    const r = transition(Status.Sleeping, Event.ExpiryTimeout);
    expect(r.next).toBe(Status.Expired);
    expect(r.destroyed).toBe(false);
  });

  it('throws on invalid transitions', () => {
    expect(() => transition(Status.Expired, Event.PacketReceived)).toThrow(InvalidTransitionError);
    expect(() => transition(Status.Connecting, Event.IdleTimeout)).toThrow(InvalidTransitionError);
    expect(() => transition(Status.Sleeping, Event.HandshakeOK)).toThrow(InvalidTransitionError);
  });

  it('full lifecycle', () => {
    let r = transition(Status.Connecting, Event.HandshakeOK);
    expect(r.next).toBe(Status.Active);

    r = transition(r.next, Event.PacketReceived);
    expect(r.next).toBe(Status.Active);

    r = transition(r.next, Event.IdleTimeout);
    expect(r.next).toBe(Status.Sleeping);

    r = transition(r.next, Event.PacketReceived);
    expect(r.next).toBe(Status.Active);

    r = transition(r.next, Event.IdleTimeout);
    r = transition(r.next, Event.ExpiryTimeout);
    expect(r.next).toBe(Status.Expired);
  });
});

// --- Session: Config ---

describe('Session Config', () => {
  it('has correct defaults', () => {
    const cfg = defaultConfig();
    expect(cfg.idleTimeoutSecs).toBe(300);
    expect(cfg.expiryTimeoutSecs).toBe(86400);
    expect(cfg.handshakeTimeoutSecs).toBe(30);
    expect(cfg.maxConnections).toBe(10000);
  });
});

// --- Transport ---

describe('Transport Type', () => {
  it('has correct values', () => {
    expect(TransportType.WebSocket).toBe('websocket');
    expect(TransportType.Quic).toBe('quic');
    expect(TransportType.Grpc).toBe('grpc');
  });
});
