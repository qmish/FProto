"""Crypto layer — Noise XX handshake, AEAD, Rekey, Zeroize."""

from __future__ import annotations

import ctypes
import hashlib
import hmac
import struct
from dataclasses import dataclass
from typing import Awaitable, Callable

from cryptography.hazmat.primitives.ciphers.aead import ChaCha20Poly1305
from noise.connection import NoiseConnection, Keypair


@dataclass
class KeyPair:
    private: bytes
    public: bytes


def generate_keypair() -> KeyPair:
    """Generate an X25519 key pair using the Noise library."""
    noise = NoiseConnection.from_name(b"Noise_XX_25519_ChaChaPoly_BLAKE2s")
    noise.noise_protocol.keypairs["s"] = noise.noise_protocol.dh_fn.generate_keypair()
    kp = noise.noise_protocol.keypairs["s"]
    return KeyPair(private=bytes(kp.private), public=bytes(kp.public))


class NoiseSession:
    """Encrypted session after Noise XX handshake."""

    def __init__(self, noise: NoiseConnection) -> None:
        self._noise = noise

    def encrypt(self, plaintext: bytes) -> bytes:
        return self._noise.encrypt(plaintext)

    def decrypt(self, ciphertext: bytes) -> bytes:
        return self._noise.decrypt(ciphertext)


async def server_handshake(
    keypair: KeyPair,
    read: Callable[[], Awaitable[bytes]],
    write: Callable[[bytes], Awaitable[None]],
) -> NoiseSession:
    """Perform Noise_XX server-side (responder) handshake."""
    noise = NoiseConnection.from_name(b"Noise_XX_25519_ChaChaPoly_BLAKE2s")
    noise.set_as_responder()
    noise.set_keypair_from_private_bytes(Keypair.STATIC, keypair.private)
    noise.start_handshake()

    # <- e
    msg1 = await read()
    noise.read_message(msg1)

    # -> e, ee, s, es
    msg2 = noise.write_message()
    await write(msg2)

    # <- s, se
    msg3 = await read()
    noise.read_message(msg3)

    return NoiseSession(noise)


async def client_handshake(
    keypair: KeyPair,
    read: Callable[[], Awaitable[bytes]],
    write: Callable[[bytes], Awaitable[None]],
) -> NoiseSession:
    """Perform Noise_XX client-side (initiator) handshake."""
    noise = NoiseConnection.from_name(b"Noise_XX_25519_ChaChaPoly_BLAKE2s")
    noise.set_as_initiator()
    noise.set_keypair_from_private_bytes(Keypair.STATIC, keypair.private)
    noise.start_handshake()

    # -> e
    msg1 = noise.write_message()
    await write(msg1)

    # <- e, ee, s, es
    msg2 = await read()
    noise.read_message(msg2)

    # -> s, se
    msg3 = noise.write_message()
    await write(msg3)

    return NoiseSession(noise)


def new_aead(key: bytes) -> ChaCha20Poly1305:
    """Create a ChaCha20-Poly1305 AEAD cipher from a 32-byte key."""
    if len(key) != 32:
        raise ValueError(f"invalid key size: expected 32, got {len(key)}")
    return ChaCha20Poly1305(key)


def build_nonce(seq: int) -> bytes:
    """Build a 96-bit nonce: 4 zero bytes + LE64(seq)."""
    return b"\x00" * 4 + struct.pack("<Q", seq)


def build_aad(session_id: bytes, seq: int) -> bytes:
    """Build AAD: session_id || LE64(seq)."""
    return session_id + struct.pack("<Q", seq)


def encrypt_aead(
    cipher: ChaCha20Poly1305,
    session_id: bytes,
    seq: int,
    plaintext: bytes,
) -> bytes:
    """Encrypt using ChaCha20-Poly1305 with seq-based nonce and AAD."""
    nonce = build_nonce(seq)
    aad = build_aad(session_id, seq)
    return cipher.encrypt(nonce, plaintext, aad)


def decrypt_aead(
    cipher: ChaCha20Poly1305,
    session_id: bytes,
    seq: int,
    ciphertext: bytes,
) -> bytes:
    """Decrypt using ChaCha20-Poly1305 with seq-based nonce and AAD."""
    nonce = build_nonce(seq)
    aad = build_aad(session_id, seq)
    return cipher.decrypt(nonce, ciphertext, aad)


def rekey(current_key: bytes) -> bytes:
    """Derive a new 32-byte key via HKDF-SHA256."""
    prk = hmac.new(b"", current_key, hashlib.sha256).digest()
    info = b"fproto-rekey"
    okm = hmac.new(prk, info + b"\x01", hashlib.sha256).digest()
    return okm[:32]


def zeroize(buf: bytearray) -> None:
    """Overwrite buffer with zeros. Only works with mutable bytearray."""
    ctypes.memset(ctypes.addressof((ctypes.c_char * len(buf)).from_buffer(buf)), 0, len(buf))
