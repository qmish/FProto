export interface ReconnectConfig {
  initialDelayMs: number;
  maxDelayMs: number;
  multiplier: number;
  jitter: number;  // ±fraction, e.g. 0.2 = ±20%
  maxAttempts: number; // 0 = unlimited
}

export function defaultReconnectConfig(): ReconnectConfig {
  return {
    initialDelayMs: 1000,
    maxDelayMs: 30000,
    multiplier: 2.0,
    jitter: 0.2,
    maxAttempts: 0,
  };
}

/**
 * Compute the reconnection delay for a given attempt.
 */
export function computeDelay(attempt: number, cfg: ReconnectConfig): number {
  let delay = cfg.initialDelayMs * Math.pow(cfg.multiplier, attempt);
  if (delay > cfg.maxDelayMs) {
    delay = cfg.maxDelayMs;
  }
  if (cfg.jitter > 0) {
    const delta = delay * cfg.jitter;
    delay = delay - delta + Math.random() * 2 * delta;
  }
  return delay;
}
