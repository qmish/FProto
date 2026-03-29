"""FProto echo server — WebSocket + Noise XX handshake."""

import asyncio
import logging

import websockets.server

from fproto.crypto import generate_keypair, server_handshake
from fproto.transport import WsServerConn

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("fproto.echo")


async def handle_client(ws: websockets.server.WebSocketServerProtocol) -> None:
    conn = WsServerConn(ws)
    server_key = generate_keypair()

    session = await server_handshake(
        server_key,
        read=conn.read_message,
        write=conn.write_message,
    )
    log.info("client connected: %s", conn.remote_addr())

    try:
        while True:
            ct = await conn.read_message()
            pt = session.decrypt(ct)
            log.info("received: %s", pt.decode(errors="replace"))

            reply = session.encrypt(pt)
            await conn.write_message(reply)
    except Exception:
        log.info("client disconnected: %s", conn.remote_addr())


async def main() -> None:
    async with websockets.server.serve(handle_client, "localhost", 9300):
        log.info("echo server listening on ws://localhost:9300")
        await asyncio.Future()


if __name__ == "__main__":
    asyncio.run(main())
