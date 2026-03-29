"""FProto Protocol SDK for Python."""

from .crypto import (
    KeyPair,
    generate_keypair,
    NoiseSession,
    server_handshake,
    client_handshake,
    new_aead,
    build_nonce,
    build_aad,
    encrypt_aead,
    decrypt_aead,
    rekey,
    zeroize,
)
from .session import Status, Event, transition, InvalidTransition, Config
from .transport import TransportType

__version__ = "0.1.0"
__all__ = [
    "KeyPair", "generate_keypair",
    "NoiseSession", "server_handshake", "client_handshake",
    "new_aead", "build_nonce", "build_aad", "encrypt_aead", "decrypt_aead",
    "rekey", "zeroize",
    "Status", "Event", "transition", "InvalidTransition", "Config",
    "TransportType",
]
