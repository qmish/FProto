"""FProto echo client — WebSocket + Noise XX handshake."""

import asyncio
import logging

from fproto.crypto import generate_keypair, client_handshake
from fproto.transport import WsClientConn

logging.basicConfig(level=logging.INFO)
log = logging.getLogger("fproto.echo")


async def main() -> None:
    conn = await WsClientConn.connect("ws://localhost:9300")
    client_key = generate_keypair()

    session = await client_handshake(
        client_key,
        read=conn.read_message,
        write=conn.write_message,
    )
    log.info("handshake complete")

    for msg in ["Hello FProto!", "Encrypted Python!", "Goodbye!"]:
        ct = session.encrypt(msg.encode())
        await conn.write_message(ct)

        reply_ct = await conn.read_message()
        reply = session.decrypt(reply_ct)
        log.info("echo: %s", reply.decode())

    await conn.close()
    log.info("session complete")


if __name__ == "__main__":
    asyncio.run(main())
