"""Transport layer — WebSocket implementation."""

from __future__ import annotations

import enum
from typing import Protocol

import websockets.client
import websockets.server


class TransportType(enum.Enum):
    WEBSOCKET = "websocket"
    QUIC = "quic"
    GRPC = "grpc"


class Conn(Protocol):
    """Transport connection interface matching proto-core Conn."""

    async def read_message(self) -> bytes: ...
    async def write_message(self, data: bytes) -> None: ...
    async def close(self) -> None: ...
    def transport_type(self) -> TransportType: ...
    def remote_addr(self) -> str: ...


class WsServerConn:
    """Server-side WebSocket connection wrapper."""

    def __init__(self, ws: websockets.server.WebSocketServerProtocol) -> None:
        self._ws = ws

    async def read_message(self) -> bytes:
        data = await self._ws.recv()
        if isinstance(data, str):
            return data.encode()
        return data

    async def write_message(self, data: bytes) -> None:
        await self._ws.send(data)

    async def close(self) -> None:
        await self._ws.close()

    def transport_type(self) -> TransportType:
        return TransportType.WEBSOCKET

    def remote_addr(self) -> str:
        peer = self._ws.remote_address
        if peer:
            return f"{peer[0]}:{peer[1]}"
        return "unknown"


class WsClientConn:
    """Client-side WebSocket connection wrapper."""

    def __init__(self, ws: websockets.client.WebSocketClientProtocol) -> None:
        self._ws = ws

    @classmethod
    async def connect(cls, url: str) -> "WsClientConn":
        ws = await websockets.client.connect(url)
        return cls(ws)

    async def read_message(self) -> bytes:
        data = await self._ws.recv()
        if isinstance(data, str):
            return data.encode()
        return data

    async def write_message(self, data: bytes) -> None:
        await self._ws.send(data)

    async def close(self) -> None:
        await self._ws.close()

    def transport_type(self) -> TransportType:
        return TransportType.WEBSOCKET

    def remote_addr(self) -> str:
        return str(self._ws.remote_address) if self._ws.remote_address else "unknown"
