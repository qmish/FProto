use chacha20poly1305::{
    aead::{Aead, KeyInit},
    ChaCha20Poly1305, Nonce,
};
use hkdf::Hkdf;
use sha2::Sha256;
use snow::{Builder, TransportState};
use thiserror::Error;
use zeroize::Zeroize;

const NOISE_PATTERN: &str = "Noise_XX_25519_ChaChaPoly_BLAKE2s";

#[derive(Error, Debug)]
pub enum CryptoError {
    #[error("noise error: {0}")]
    Noise(#[from] snow::Error),
    #[error("aead error")]
    Aead,
    #[error("invalid key size: expected {expected}, got {got}")]
    InvalidKeySize { expected: usize, got: usize },
    #[error("hkdf error")]
    Hkdf,
}

pub struct KeyPair {
    pub private: Vec<u8>,
    pub public: Vec<u8>,
}

impl Drop for KeyPair {
    fn drop(&mut self) {
        self.private.zeroize();
    }
}

pub fn generate_keypair() -> Result<KeyPair, CryptoError> {
    let builder = Builder::new(NOISE_PATTERN.parse()?);
    let keypair = builder.generate_keypair()?;
    Ok(KeyPair {
        private: keypair.private.to_vec(),
        public: keypair.public.to_vec(),
    })
}

pub struct NoiseSession {
    transport: TransportState,
}

impl NoiseSession {
    pub fn encrypt(&mut self, plaintext: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let mut buf = vec![0u8; plaintext.len() + 64];
        let len = self.transport.write_message(plaintext, &mut buf)?;
        buf.truncate(len);
        Ok(buf)
    }

    pub fn decrypt(&mut self, ciphertext: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let mut buf = vec![0u8; ciphertext.len()];
        let len = self.transport.read_message(ciphertext, &mut buf)?;
        buf.truncate(len);
        Ok(buf)
    }
}

pub async fn server_handshake<R, W>(
    keypair: &KeyPair,
    mut read: R,
    mut write: W,
) -> Result<NoiseSession, CryptoError>
where
    R: FnMut() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<Vec<u8>, Box<dyn std::error::Error + Send>>> + Send>>,
    W: FnMut(Vec<u8>) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), Box<dyn std::error::Error + Send>>> + Send>>,
{
    let builder = Builder::new(NOISE_PATTERN.parse()?)
        .local_private_key(&keypair.private);
    let mut hs = builder.build_responder()?;
    let mut buf = vec![0u8; 65535];

    // <- e
    let msg1 = read().await.map_err(|_| snow::Error::Input)?;
    hs.read_message(&msg1, &mut buf)?;

    // -> e, ee, s, es
    let len = hs.write_message(&[], &mut buf)?;
    write(buf[..len].to_vec()).await.map_err(|_| snow::Error::Input)?;

    // <- s, se
    let msg3 = read().await.map_err(|_| snow::Error::Input)?;
    hs.read_message(&msg3, &mut buf)?;

    let transport = hs.into_transport_mode()?;
    Ok(NoiseSession { transport })
}

pub async fn client_handshake<R, W>(
    keypair: &KeyPair,
    mut read: R,
    mut write: W,
) -> Result<NoiseSession, CryptoError>
where
    R: FnMut() -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<Vec<u8>, Box<dyn std::error::Error + Send>>> + Send>>,
    W: FnMut(Vec<u8>) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), Box<dyn std::error::Error + Send>>> + Send>>,
{
    let builder = Builder::new(NOISE_PATTERN.parse()?)
        .local_private_key(&keypair.private);
    let mut hs = builder.build_initiator()?;
    let mut buf = vec![0u8; 65535];

    // -> e
    let len = hs.write_message(&[], &mut buf)?;
    write(buf[..len].to_vec()).await.map_err(|_| snow::Error::Input)?;

    // <- e, ee, s, es
    let msg2 = read().await.map_err(|_| snow::Error::Input)?;
    hs.read_message(&msg2, &mut buf)?;

    // -> s, se
    let len = hs.write_message(&[], &mut buf)?;
    write(buf[..len].to_vec()).await.map_err(|_| snow::Error::Input)?;

    let transport = hs.into_transport_mode()?;
    Ok(NoiseSession { transport })
}

pub fn new_aead(key: &[u8]) -> Result<ChaCha20Poly1305, CryptoError> {
    if key.len() != 32 {
        return Err(CryptoError::InvalidKeySize {
            expected: 32,
            got: key.len(),
        });
    }
    Ok(ChaCha20Poly1305::new_from_slice(key).map_err(|_| CryptoError::Aead)?)
}

pub fn build_nonce(seq: u64) -> [u8; 12] {
    let mut nonce = [0u8; 12];
    nonce[4..12].copy_from_slice(&seq.to_le_bytes());
    nonce
}

pub fn build_aad(session_id: &[u8], seq: u64) -> Vec<u8> {
    let mut aad = Vec::with_capacity(session_id.len() + 8);
    aad.extend_from_slice(session_id);
    aad.extend_from_slice(&seq.to_le_bytes());
    aad
}

pub fn encrypt_aead(
    cipher: &ChaCha20Poly1305,
    session_id: &[u8],
    seq: u64,
    plaintext: &[u8],
) -> Result<Vec<u8>, CryptoError> {
    let nonce_bytes = build_nonce(seq);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let aad = build_aad(session_id, seq);
    use chacha20poly1305::aead::Payload;
    let payload = Payload { msg: plaintext, aad: &aad };
    cipher.encrypt(nonce, payload).map_err(|_| CryptoError::Aead)
}

pub fn decrypt_aead(
    cipher: &ChaCha20Poly1305,
    session_id: &[u8],
    seq: u64,
    ciphertext: &[u8],
) -> Result<Vec<u8>, CryptoError> {
    let nonce_bytes = build_nonce(seq);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let aad = build_aad(session_id, seq);
    use chacha20poly1305::aead::Payload;
    let payload = Payload { msg: ciphertext, aad: &aad };
    cipher.decrypt(nonce, payload).map_err(|_| CryptoError::Aead)
}

pub fn rekey(current_key: &[u8]) -> Result<Vec<u8>, CryptoError> {
    let hk = Hkdf::<Sha256>::new(None, current_key);
    let mut new_key = vec![0u8; 32];
    hk.expand(b"fproto-rekey", &mut new_key)
        .map_err(|_| CryptoError::Hkdf)?;
    Ok(new_key)
}

pub fn zeroize_bytes(buf: &mut [u8]) {
    buf.zeroize();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_keypair_generation() {
        let kp = generate_keypair().unwrap();
        assert_eq!(kp.public.len(), 32);
        assert_eq!(kp.private.len(), 32);
    }

    #[test]
    fn test_aead_encrypt_decrypt() {
        let key: Vec<u8> = (0..32).collect();
        let cipher = new_aead(&key).unwrap();
        let sid = b"test-session";
        let plaintext = b"secret data";

        let ct = encrypt_aead(&cipher, sid, 1, plaintext).unwrap();
        let pt = decrypt_aead(&cipher, sid, 1, &ct).unwrap();
        assert_eq!(pt, plaintext);
    }

    #[test]
    fn test_aead_tamper_detection() {
        let key: Vec<u8> = (0..32).collect();
        let cipher = new_aead(&key).unwrap();
        let sid = b"test-session";

        let mut ct = encrypt_aead(&cipher, sid, 1, b"data").unwrap();
        ct[0] ^= 0xff;
        assert!(decrypt_aead(&cipher, sid, 1, &ct).is_err());
    }

    #[test]
    fn test_rekey_deterministic() {
        let key: Vec<u8> = (0..32).collect();
        let k1 = rekey(&key).unwrap();
        let k2 = rekey(&key).unwrap();
        assert_eq!(k1, k2);
        assert_ne!(k1, key);
    }

    #[test]
    fn test_zeroize() {
        let mut buf = vec![1u8, 2, 3, 4, 5];
        zeroize_bytes(&mut buf);
        assert!(buf.iter().all(|&b| b == 0));
    }

    #[test]
    fn test_nonce_uniqueness() {
        let mut seen = std::collections::HashSet::new();
        for i in 0u64..1000 {
            let n = build_nonce(i);
            assert!(seen.insert(n));
        }
    }

    #[test]
    fn test_build_aad() {
        let sid = b"session-1";
        let aad = build_aad(sid, 42);
        assert_eq!(aad.len(), sid.len() + 8);
        assert_eq!(&aad[..sid.len()], sid);
    }

    #[test]
    fn test_invalid_key_size() {
        let short_key = vec![0u8; 16];
        assert!(new_aead(&short_key).is_err());
    }
}
