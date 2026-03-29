export enum Status {
  Connecting = 'connecting',
  Active = 'active',
  Sleeping = 'sleeping',
  Expired = 'expired',
}

export enum Event {
  HandshakeOK = 'handshake_ok',
  HandshakeFail = 'handshake_fail',
  PacketReceived = 'packet_received',
  IdleTimeout = 'idle_timeout',
  ExpiryTimeout = 'expiry_timeout',
  ExplicitLogout = 'explicit_logout',
}

export interface TransitionResult {
  next: Status;
  destroyed: boolean;
}

export class InvalidTransitionError extends Error {
  constructor(
    public status: Status,
    public event: Event,
  ) {
    super(`invalid transition: ${status} + ${event}`);
    this.name = 'InvalidTransitionError';
  }
}

type TransitionKey = `${Status}:${Event}`;

const transitions = new Map<TransitionKey, TransitionResult>([
  [`${Status.Connecting}:${Event.HandshakeOK}`, { next: Status.Active, destroyed: false }],
  [`${Status.Connecting}:${Event.HandshakeFail}`, { next: Status.Expired, destroyed: true }],
  [`${Status.Active}:${Event.PacketReceived}`, { next: Status.Active, destroyed: false }],
  [`${Status.Active}:${Event.IdleTimeout}`, { next: Status.Sleeping, destroyed: false }],
  [`${Status.Active}:${Event.ExplicitLogout}`, { next: Status.Expired, destroyed: false }],
  [`${Status.Sleeping}:${Event.PacketReceived}`, { next: Status.Active, destroyed: false }],
  [`${Status.Sleeping}:${Event.ExpiryTimeout}`, { next: Status.Expired, destroyed: false }],
]);

export function transition(current: Status, event: Event): TransitionResult {
  const key: TransitionKey = `${current}:${event}`;
  const result = transitions.get(key);
  if (!result) {
    throw new InvalidTransitionError(current, event);
  }
  return { ...result };
}

export interface SessionConfig {
  idleTimeoutSecs: number;
  expiryTimeoutSecs: number;
  handshakeTimeoutSecs: number;
  maxConnections: number;
}

export function defaultConfig(): SessionConfig {
  return {
    idleTimeoutSecs: 300,
    expiryTimeoutSecs: 86400,
    handshakeTimeoutSecs: 30,
    maxConnections: 10000,
  };
}
