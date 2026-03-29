import WebSocket, { WebSocketServer } from 'ws';
import { WsServerConn, WsClientConn } from '../src/transport';

const mode = process.argv[2] || 'server';
const addr = process.argv[3] || '127.0.0.1:9400';

if (mode === 'server') {
  const [host, port] = addr.split(':');
  const wss = new WebSocketServer({ host, port: parseInt(port) });

  console.log(`[server] listening on ws://${addr}`);

  wss.on('connection', (ws, req) => {
    const peer = `${req.socket.remoteAddress}:${req.socket.remotePort}`;
    const conn = new WsServerConn(ws, peer);
    console.log(`[server] client connected: ${peer}`);

    const echoLoop = async () => {
      try {
        while (true) {
          const data = await conn.readMessage();
          console.log(`[server] received: ${data.toString()}`);
          await conn.writeMessage(data);
        }
      } catch {
        console.log(`[server] client disconnected: ${peer}`);
      }
    };
    echoLoop();
  });
} else {
  (async () => {
    const conn = await WsClientConn.connect(`ws://${addr}`);
    console.log(`[client] connected to ${addr}`);

    const messages = ['Hello FProto!', 'Encrypted TypeScript!', 'Goodbye!'];
    for (const msg of messages) {
      await conn.writeMessage(Buffer.from(msg));
      const reply = await conn.readMessage();
      console.log(`[client] echo: ${reply.toString()}`);
    }

    await conn.close();
    console.log('[client] done');
  })();
}
