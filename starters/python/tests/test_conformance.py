"""FProto conformance tests for Python SDK."""

import asyncio
import pytest
from fproto.crypto import (
    generate_keypair,
    new_aead,
    build_nonce,
    build_aad,
    encrypt_aead,
    decrypt_aead,
    rekey,
    zeroize,
)
from fproto.session import (
    Status,
    Event,
    transition,
    InvalidTransition,
    Config,
)
from fproto.transport import TransportType


# --- Crypto: Keypair ---

def test_keypair_generation():
    kp = generate_keypair()
    assert len(kp.public) == 32
    assert len(kp.private) == 32

    kp2 = generate_keypair()
    assert kp.public != kp2.public


# --- Crypto: AEAD ---

def test_aead_encrypt_decrypt():
    key = bytes(range(32))
    cipher = new_aead(key)
    sid = b"test-session"
    plaintext = b"secret data"

    ct = encrypt_aead(cipher, sid, 1, plaintext)
    pt = decrypt_aead(cipher, sid, 1, ct)
    assert pt == plaintext


def test_aead_wrong_seq_fails():
    key = bytes(range(32))
    cipher = new_aead(key)
    sid = b"session"

    ct = encrypt_aead(cipher, sid, 1, b"data")
    with pytest.raises(Exception):
        decrypt_aead(cipher, sid, 2, ct)


def test_aead_wrong_session_fails():
    key = bytes(range(32))
    cipher = new_aead(key)

    ct = encrypt_aead(cipher, b"session-a", 1, b"data")
    with pytest.raises(Exception):
        decrypt_aead(cipher, b"session-b", 1, ct)


def test_aead_tamper_detection():
    key = bytes(range(32))
    cipher = new_aead(key)
    sid = b"session"

    ct = bytearray(encrypt_aead(cipher, sid, 1, b"data"))
    ct[0] ^= 0xFF
    with pytest.raises(Exception):
        decrypt_aead(cipher, sid, 1, bytes(ct))


def test_large_payload_aead():
    key = bytes(range(32))
    cipher = new_aead(key)
    sid = b"large"

    payload = bytes(i % 256 for i in range(32768))
    ct = encrypt_aead(cipher, sid, 1, payload)
    pt = decrypt_aead(cipher, sid, 1, ct)
    assert pt == payload


def test_sequential_aead():
    key = bytes(range(32))
    cipher = new_aead(key)
    sid = b"seq-session"

    for seq in range(100):
        msg = f"message-{seq}".encode()
        ct = encrypt_aead(cipher, sid, seq, msg)
        pt = decrypt_aead(cipher, sid, seq, ct)
        assert pt == msg


# --- Crypto: Nonce ---

def test_nonce_uniqueness():
    seen = set()
    for i in range(1000):
        n = build_nonce(i)
        assert n not in seen, f"duplicate nonce at seq {i}"
        seen.add(n)


def test_nonce_structure():
    n = build_nonce(0)
    assert n == b"\x00" * 12

    n = build_nonce(1)
    assert n[:4] == b"\x00" * 4
    assert n[4:] == (1).to_bytes(8, "little")


def test_build_aad():
    sid = b"session-1"
    aad = build_aad(sid, 42)
    assert len(aad) == len(sid) + 8
    assert aad[:len(sid)] == sid


# --- Crypto: Rekey ---

def test_rekey_deterministic():
    key = bytes(range(32))
    k1 = rekey(key)
    k2 = rekey(key)
    assert k1 == k2
    assert k1 != key
    assert len(k1) == 32


def test_rekey_chain():
    key = bytes(range(32))
    for _ in range(10):
        new_key = rekey(key)
        assert new_key != key
        assert len(new_key) == 32
        key = new_key


# --- Crypto: Zeroize ---

def test_zeroize():
    buf = bytearray([0xAA] * 64)
    zeroize(buf)
    assert all(b == 0 for b in buf)


# --- Crypto: Key validation ---

def test_invalid_key_size():
    with pytest.raises(ValueError):
        new_aead(b"\x00" * 16)


# --- Session: State machine ---

def test_connecting_handshake_ok():
    r = transition(Status.CONNECTING, Event.HANDSHAKE_OK)
    assert r.next_status == Status.ACTIVE
    assert not r.destroyed


def test_connecting_handshake_fail():
    r = transition(Status.CONNECTING, Event.HANDSHAKE_FAIL)
    assert r.destroyed


def test_active_idle_timeout():
    r = transition(Status.ACTIVE, Event.IDLE_TIMEOUT)
    assert r.next_status == Status.SLEEPING
    assert not r.destroyed


def test_active_packet_received():
    r = transition(Status.ACTIVE, Event.PACKET_RECEIVED)
    assert r.next_status == Status.ACTIVE


def test_active_explicit_logout():
    r = transition(Status.ACTIVE, Event.EXPLICIT_LOGOUT)
    assert r.next_status == Status.EXPIRED


def test_sleeping_packet_received():
    r = transition(Status.SLEEPING, Event.PACKET_RECEIVED)
    assert r.next_status == Status.ACTIVE


def test_sleeping_expiry_timeout():
    r = transition(Status.SLEEPING, Event.EXPIRY_TIMEOUT)
    assert r.next_status == Status.EXPIRED
    assert not r.destroyed


def test_invalid_transitions():
    with pytest.raises(InvalidTransition):
        transition(Status.EXPIRED, Event.PACKET_RECEIVED)
    with pytest.raises(InvalidTransition):
        transition(Status.CONNECTING, Event.IDLE_TIMEOUT)
    with pytest.raises(InvalidTransition):
        transition(Status.SLEEPING, Event.HANDSHAKE_OK)


def test_full_lifecycle():
    r = transition(Status.CONNECTING, Event.HANDSHAKE_OK)
    assert r.next_status == Status.ACTIVE

    r = transition(r.next_status, Event.PACKET_RECEIVED)
    assert r.next_status == Status.ACTIVE

    r = transition(r.next_status, Event.IDLE_TIMEOUT)
    assert r.next_status == Status.SLEEPING

    r = transition(r.next_status, Event.PACKET_RECEIVED)
    assert r.next_status == Status.ACTIVE

    r = transition(r.next_status, Event.IDLE_TIMEOUT)
    r = transition(r.next_status, Event.EXPIRY_TIMEOUT)
    assert r.next_status == Status.EXPIRED


def test_default_config():
    cfg = Config()
    assert cfg.idle_timeout_secs == 300
    assert cfg.expiry_timeout_secs == 86400
    assert cfg.handshake_timeout_secs == 30
    assert cfg.max_connections == 10000


# --- Transport ---

def test_transport_types():
    assert TransportType.WEBSOCKET.value == "websocket"
    assert TransportType.QUIC.value == "quic"
    assert TransportType.GRPC.value == "grpc"
