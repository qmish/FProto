use fproto_core::crypto;
use fproto_core::session::{self, Event, Status};
use fproto_core::transport::TransportType;

#[test]
fn test_keypair_generation() {
    let kp = crypto::generate_keypair().unwrap();
    assert_eq!(kp.public.len(), 32);
    assert_eq!(kp.private.len(), 32);

    let kp2 = crypto::generate_keypair().unwrap();
    assert_ne!(kp.public, kp2.public);
}

#[test]
fn test_aead_encrypt_decrypt() {
    let key: Vec<u8> = (0..32).collect();
    let cipher = crypto::new_aead(&key).unwrap();
    let sid = b"conformance-session";
    let plaintext = b"conformance test data";

    let ct = crypto::encrypt_aead(&cipher, sid, 1, plaintext).unwrap();
    let pt = crypto::decrypt_aead(&cipher, sid, 1, &ct).unwrap();
    assert_eq!(pt, plaintext);
}

#[test]
fn test_aead_wrong_seq_fails() {
    let key: Vec<u8> = (0..32).collect();
    let cipher = crypto::new_aead(&key).unwrap();
    let sid = b"session";

    let ct = crypto::encrypt_aead(&cipher, sid, 1, b"data").unwrap();
    assert!(crypto::decrypt_aead(&cipher, sid, 2, &ct).is_err());
}

#[test]
fn test_aead_wrong_session_fails() {
    let key: Vec<u8> = (0..32).collect();
    let cipher = crypto::new_aead(&key).unwrap();

    let ct = crypto::encrypt_aead(&cipher, b"session-a", 1, b"data").unwrap();
    assert!(crypto::decrypt_aead(&cipher, b"session-b", 1, &ct).is_err());
}

#[test]
fn test_rekey_deterministic() {
    let key: Vec<u8> = (0..32).collect();
    let k1 = crypto::rekey(&key).unwrap();
    let k2 = crypto::rekey(&key).unwrap();
    assert_eq!(k1, k2);
    assert_ne!(k1, key);
    assert_eq!(k1.len(), 32);
}

#[test]
fn test_rekey_chain() {
    let mut key: Vec<u8> = (0..32).collect();
    for _ in 0..10 {
        let new_key = crypto::rekey(&key).unwrap();
        assert_ne!(new_key, key);
        assert_eq!(new_key.len(), 32);
        key = new_key;
    }
}

#[test]
fn test_zeroize() {
    let mut buf = vec![0xAA; 64];
    crypto::zeroize_bytes(&mut buf);
    assert!(buf.iter().all(|&b| b == 0));
}

#[test]
fn test_nonce_uniqueness() {
    let mut seen = std::collections::HashSet::new();
    for i in 0u64..1000 {
        let n = crypto::build_nonce(i);
        assert!(seen.insert(n), "duplicate nonce at seq {}", i);
    }
}

#[test]
fn test_nonce_structure() {
    let n = crypto::build_nonce(0);
    assert_eq!(n, [0u8; 12]);

    let n = crypto::build_nonce(1);
    assert_eq!(&n[..4], &[0, 0, 0, 0]);
    assert_eq!(&n[4..], &1u64.to_le_bytes());
}

#[test]
fn test_session_state_machine_full_lifecycle() {
    let r = session::transition(Status::Connecting, Event::HandshakeOK).unwrap();
    assert_eq!(r.next, Status::Active);
    assert!(!r.destroyed);

    let r = session::transition(Status::Active, Event::PacketReceived).unwrap();
    assert_eq!(r.next, Status::Active);

    let r = session::transition(Status::Active, Event::IdleTimeout).unwrap();
    assert_eq!(r.next, Status::Sleeping);

    let r = session::transition(Status::Sleeping, Event::PacketReceived).unwrap();
    assert_eq!(r.next, Status::Active);

    let r = session::transition(Status::Active, Event::ExplicitLogout).unwrap();
    assert_eq!(r.next, Status::Expired);
}

#[test]
fn test_session_handshake_fail_destroys() {
    let r = session::transition(Status::Connecting, Event::HandshakeFail).unwrap();
    assert!(r.destroyed);
}

#[test]
fn test_session_sleeping_expiry() {
    let r = session::transition(Status::Sleeping, Event::ExpiryTimeout).unwrap();
    assert_eq!(r.next, Status::Expired);
    assert!(!r.destroyed);
}

#[test]
fn test_session_invalid_transitions() {
    assert!(session::transition(Status::Expired, Event::PacketReceived).is_err());
    assert!(session::transition(Status::Connecting, Event::IdleTimeout).is_err());
    assert!(session::transition(Status::Sleeping, Event::HandshakeOK).is_err());
}

#[test]
fn test_transport_type_display() {
    assert_eq!(TransportType::WebSocket.to_string(), "websocket");
    assert_eq!(TransportType::Quic.to_string(), "quic");
    assert_eq!(TransportType::Grpc.to_string(), "grpc");
}

#[test]
fn test_large_payload_aead() {
    let key: Vec<u8> = (0..32).collect();
    let cipher = crypto::new_aead(&key).unwrap();
    let sid = b"large-payload";

    let payload: Vec<u8> = (0..32768).map(|i| (i % 256) as u8).collect();
    let ct = crypto::encrypt_aead(&cipher, sid, 1, &payload).unwrap();
    let pt = crypto::decrypt_aead(&cipher, sid, 1, &ct).unwrap();
    assert_eq!(pt, payload);
}

#[test]
fn test_sequential_aead() {
    let key: Vec<u8> = (0..32).collect();
    let cipher = crypto::new_aead(&key).unwrap();
    let sid = b"seq-session";

    for seq in 0u64..100 {
        let msg = format!("message-{}", seq);
        let ct = crypto::encrypt_aead(&cipher, sid, seq, msg.as_bytes()).unwrap();
        let pt = crypto::decrypt_aead(&cipher, sid, seq, &ct).unwrap();
        assert_eq!(pt, msg.as_bytes());
    }
}

#[tokio::test]
async fn test_ws_echo_round_trip() {
    use fproto_core::transport;
    use tokio::net::TcpListener;

    let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();

    let server = tokio::spawn(async move {
        let (stream, peer) = listener.accept().await.unwrap();
        let mut conn = transport::accept_ws(stream, peer.to_string()).await.unwrap();
        let data = conn.read_message().await.unwrap();
        conn.write_message(&data).await.unwrap();
    });

    let url = format!("ws://{}", addr);
    let mut client = transport::WsClientConn::connect(&url).await.unwrap();
    client.write_message(b"hello rust").await.unwrap();
    let reply = client.read_message().await.unwrap();
    assert_eq!(reply, b"hello rust");

    server.await.unwrap();
}
