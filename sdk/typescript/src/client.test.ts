import { Server as WebSocketServer } from 'ws';
import { FProtoClient } from './client';
import { computeDelay, defaultReconnectConfig } from './reconnect';

let wss: WebSocketServer;
let port: number;

beforeAll((done) => {
  wss = new WebSocketServer({ port: 0 }, () => {
    const addr = wss.address();
    if (typeof addr === 'object' && addr !== null) {
      port = addr.port;
    }
    done();
  });

  wss.on('connection', (ws) => {
    ws.on('message', (data: Buffer) => {
      const msg = Buffer.from(data);
      const echo = Buffer.concat([Buffer.from('echo: '), msg]);
      ws.send(echo);
    });
  });
});

afterAll(() => {
  wss.close();
});

test('connect and echo', async () => {
  const client = new FProtoClient({ serverUrl: `ws://localhost:${port}` });
  await client.connect();

  expect(client.isConnected).toBe(true);

  await client.send(new TextEncoder().encode('Hello TS!'));
  const resp = await client.receive();
  expect(new TextDecoder().decode(resp)).toBe('echo: Hello TS!');

  await client.close();
  expect(client.isConnected).toBe(false);
});

test('multiple messages', async () => {
  const client = new FProtoClient({ serverUrl: `ws://localhost:${port}` });
  await client.connect();

  for (let i = 0; i < 10; i++) {
    const msg = `msg-${i}`;
    await client.send(new TextEncoder().encode(msg));
    const resp = await client.receive();
    expect(new TextDecoder().decode(resp)).toBe(`echo: ${msg}`);
  }

  await client.close();
});

test('reconnect config defaults', () => {
  const cfg = defaultReconnectConfig();
  expect(cfg.initialDelayMs).toBe(1000);
  expect(cfg.maxDelayMs).toBe(30000);
  expect(cfg.multiplier).toBe(2.0);

  const d0 = computeDelay(0, cfg);
  expect(d0).toBeGreaterThan(800);
  expect(d0).toBeLessThan(1200);

  const d5 = computeDelay(5, cfg);
  expect(d5).toBeLessThanOrEqual(36000);
});

test('send before connect throws', async () => {
  const client = new FProtoClient({ serverUrl: 'ws://localhost:1' });
  await expect(client.send(new Uint8Array([1]))).rejects.toThrow('Not connected');
});
