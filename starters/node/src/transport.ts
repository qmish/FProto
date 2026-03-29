import WebSocket, { WebSocketServer } from 'ws';

export enum TransportType {
  WebSocket = 'websocket',
  Quic = 'quic',
  Grpc = 'grpc',
}

export interface Conn {
  readMessage(): Promise<Buffer>;
  writeMessage(data: Buffer): Promise<void>;
  close(): Promise<void>;
  transportType(): TransportType;
  remoteAddr(): string;
}

export class WsServerConn implements Conn {
  private queue: Buffer[] = [];
  private waiters: Array<(data: Buffer) => void> = [];
  private closed = false;

  constructor(
    private ws: WebSocket,
    private addr: string,
  ) {
    ws.binaryType = 'nodebuffer';
    ws.on('message', (data: Buffer) => {
      if (this.waiters.length > 0) {
        this.waiters.shift()!(data);
      } else {
        this.queue.push(data);
      }
    });
    ws.on('close', () => {
      this.closed = true;
      for (const w of this.waiters) {
        w(Buffer.alloc(0));
      }
      this.waiters = [];
    });
  }

  readMessage(): Promise<Buffer> {
    if (this.queue.length > 0) {
      return Promise.resolve(this.queue.shift()!);
    }
    if (this.closed) {
      return Promise.reject(new Error('connection closed'));
    }
    return new Promise((resolve) => {
      this.waiters.push(resolve);
    });
  }

  writeMessage(data: Buffer): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws.send(data, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  close(): Promise<void> {
    this.ws.close();
    return Promise.resolve();
  }

  transportType(): TransportType {
    return TransportType.WebSocket;
  }

  remoteAddr(): string {
    return this.addr;
  }
}

export class WsClientConn implements Conn {
  private queue: Buffer[] = [];
  private waiters: Array<(data: Buffer) => void> = [];
  private closed = false;

  constructor(private ws: WebSocket) {
    ws.binaryType = 'nodebuffer';
    ws.on('message', (data: Buffer) => {
      if (this.waiters.length > 0) {
        this.waiters.shift()!(data);
      } else {
        this.queue.push(data);
      }
    });
    ws.on('close', () => {
      this.closed = true;
      for (const w of this.waiters) {
        w(Buffer.alloc(0));
      }
      this.waiters = [];
    });
  }

  static connect(url: string): Promise<WsClientConn> {
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(url);
      ws.on('open', () => resolve(new WsClientConn(ws)));
      ws.on('error', reject);
    });
  }

  readMessage(): Promise<Buffer> {
    if (this.queue.length > 0) {
      return Promise.resolve(this.queue.shift()!);
    }
    if (this.closed) {
      return Promise.reject(new Error('connection closed'));
    }
    return new Promise((resolve) => {
      this.waiters.push(resolve);
    });
  }

  writeMessage(data: Buffer): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws.send(data, (err) => {
        if (err) reject(err);
        else resolve();
      });
    });
  }

  close(): Promise<void> {
    this.ws.close();
    return Promise.resolve();
  }

  transportType(): TransportType {
    return TransportType.WebSocket;
  }

  remoteAddr(): string {
    return this.ws.url ?? 'unknown';
  }
}
