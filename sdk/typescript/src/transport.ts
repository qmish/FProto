import WebSocket from 'ws';

export interface Transport {
  connect(): Promise<void>;
  send(data: Uint8Array): Promise<void>;
  receive(): Promise<Uint8Array>;
  close(): Promise<void>;
}

/**
 * WebSocketTransport implements the Transport interface using the ws library.
 */
export class WebSocketTransport implements Transport {
  private ws: WebSocket | null = null;
  private url: string;
  private messageQueue: Uint8Array[] = [];
  private waiters: Array<(data: Uint8Array) => void> = [];

  constructor(url: string) {
    this.url = url;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      this.ws.binaryType = 'nodebuffer';

      this.ws.on('open', () => resolve());
      this.ws.on('error', (err) => reject(err));
      this.ws.on('message', (data: Buffer) => {
        const bytes = new Uint8Array(data);
        if (this.waiters.length > 0) {
          const waiter = this.waiters.shift()!;
          waiter(bytes);
        } else {
          this.messageQueue.push(bytes);
        }
      });
    });
  }

  async send(data: Uint8Array): Promise<void> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket not connected');
    }
    this.ws.send(data);
  }

  receive(): Promise<Uint8Array> {
    if (this.messageQueue.length > 0) {
      return Promise.resolve(this.messageQueue.shift()!);
    }
    return new Promise((resolve) => {
      this.waiters.push(resolve);
    });
  }

  async close(): Promise<void> {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
