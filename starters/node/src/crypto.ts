import * as crypto from 'crypto';

export interface KeyPair {
  privateKey: Buffer;
  publicKey: Buffer;
}

export function generateKeyPair(): KeyPair {
  const keyPair = crypto.generateKeyPairSync('x25519', {
    publicKeyEncoding: { type: 'spki', format: 'der' },
    privateKeyEncoding: { type: 'pkcs8', format: 'der' },
  });
  return {
    privateKey: Buffer.from(keyPair.privateKey),
    publicKey: Buffer.from(keyPair.publicKey),
  };
}

export function newAead(key: Buffer): { key: Buffer } {
  if (key.length !== 32) {
    throw new Error(`invalid key size: expected 32, got ${key.length}`);
  }
  return { key: Buffer.from(key) };
}

export function buildNonce(seq: number): Buffer {
  const nonce = Buffer.alloc(12);
  nonce.writeBigUInt64LE(BigInt(seq), 4);
  return nonce;
}

export function buildAad(sessionId: Buffer, seq: number): Buffer {
  const aad = Buffer.alloc(sessionId.length + 8);
  sessionId.copy(aad);
  aad.writeBigUInt64LE(BigInt(seq), sessionId.length);
  return aad;
}

export function encryptAead(
  cipher: { key: Buffer },
  sessionId: Buffer,
  seq: number,
  plaintext: Buffer,
): Buffer {
  const nonce = buildNonce(seq);
  const aad = buildAad(sessionId, seq);
  const c = crypto.createCipheriv('chacha20-poly1305', cipher.key, nonce, {
    authTagLength: 16,
  });
  c.setAAD(aad);
  const encrypted = Buffer.concat([c.update(plaintext), c.final()]);
  const tag = c.getAuthTag();
  return Buffer.concat([encrypted, tag]);
}

export function decryptAead(
  cipher: { key: Buffer },
  sessionId: Buffer,
  seq: number,
  ciphertext: Buffer,
): Buffer {
  const nonce = buildNonce(seq);
  const aad = buildAad(sessionId, seq);
  const tag = ciphertext.subarray(ciphertext.length - 16);
  const data = ciphertext.subarray(0, ciphertext.length - 16);
  const d = crypto.createDecipheriv('chacha20-poly1305', cipher.key, nonce, {
    authTagLength: 16,
  });
  d.setAAD(aad);
  d.setAuthTag(tag);
  return Buffer.concat([d.update(data), d.final()]);
}

export function rekey(currentKey: Buffer): Buffer {
  const prk = crypto.createHmac('sha256', Buffer.alloc(0)).update(currentKey).digest();
  const info = Buffer.from('fproto-rekey');
  const okm = crypto
    .createHmac('sha256', prk)
    .update(Buffer.concat([info, Buffer.from([0x01])]))
    .digest();
  return okm.subarray(0, 32);
}

export function zeroize(buf: Buffer): void {
  buf.fill(0);
}
