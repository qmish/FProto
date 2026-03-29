import { Transport, WebSocketTransport } from './transport';
import { ReconnectConfig, defaultReconnectConfig, computeDelay } from './reconnect';

export interface ClientConfig {
  serverUrl: string;
  reconnect?: ReconnectConfig;
}

/**
 * FProtoClient is the TypeScript SDK client for the FProto protocol.
 * Provides WebSocket transport with automatic reconnection.
 */
export class FProtoClient {
  private transport: Transport | null = null;
  private config: ClientConfig;
  private closed = false;

  constructor(config: ClientConfig) {
    this.config = config;
  }

  async connect(): Promise<void> {
    this.transport = new WebSocketTransport(this.config.serverUrl);
    await this.transport.connect();
    this.closed = false;
  }

  async connectWithReconnect(): Promise<void> {
    const cfg = this.config.reconnect ?? defaultReconnectConfig();
    let attempt = 0;

    while (true) {
      try {
        await this.connect();
        return;
      } catch (err) {
        attempt++;
        if (cfg.maxAttempts > 0 && attempt >= cfg.maxAttempts) {
          throw err;
        }
        const delay = computeDelay(attempt - 1, cfg);
        await sleep(delay);
      }
    }
  }

  async send(data: Uint8Array): Promise<void> {
    if (!this.transport || this.closed) {
      throw new Error('Not connected');
    }
    await this.transport.send(data);
  }

  async receive(): Promise<Uint8Array> {
    if (!this.transport || this.closed) {
      throw new Error('Not connected');
    }
    return this.transport.receive();
  }

  async close(): Promise<void> {
    this.closed = true;
    if (this.transport) {
      await this.transport.close();
      this.transport = null;
    }
  }

  get isConnected(): boolean {
    return this.transport !== null && !this.closed;
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
