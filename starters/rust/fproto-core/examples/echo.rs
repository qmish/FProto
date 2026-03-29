use fproto_core::crypto;
use fproto_core::transport;
use std::env;
use tokio::net::TcpListener;

#[tokio::main]
async fn main() {
    let mode = env::args().nth(1).unwrap_or_else(|| "server".into());
    let addr = env::args().nth(2).unwrap_or_else(|| "127.0.0.1:9200".into());

    match mode.as_str() {
        "server" => run_server(&addr).await,
        "client" => run_client(&addr).await,
        _ => eprintln!("Usage: echo [server|client] [addr]"),
    }
}

async fn run_server(addr: &str) {
    let server_key = crypto::generate_keypair().expect("keygen");
    let listener = TcpListener::bind(addr).await.expect("bind");
    println!("[server] listening on ws://{}", addr);

    loop {
        let (stream, peer) = listener.accept().await.expect("accept");
        let key = crypto::KeyPair {
            private: server_key.private.clone(),
            public: server_key.public.clone(),
        };

        tokio::spawn(async move {
            let mut conn = match transport::accept_ws(stream, peer.to_string()).await {
                Ok(c) => c,
                Err(e) => {
                    eprintln!("[server] ws accept: {}", e);
                    return;
                }
            };

            let read = || {
                let fut = async { Err("not implemented in async bridge".into()) };
                Box::pin(fut) as std::pin::Pin<Box<dyn std::future::Future<Output = Result<Vec<u8>, Box<dyn std::error::Error + Send>>> + Send>>
            };
            let write = |_data: Vec<u8>| {
                let fut = async { Err("not implemented in async bridge".into()) };
                Box::pin(fut) as std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), Box<dyn std::error::Error + Send>>> + Send>>
            };

            // For the echo example, we skip the full Noise handshake
            // and demonstrate raw WebSocket echo with the transport layer
            println!("[server] client connected: {}", conn.remote_addr());

            loop {
                match conn.read_message().await {
                    Ok(data) => {
                        println!("[server] received {} bytes", data.len());
                        if let Err(e) = conn.write_message(&data).await {
                            eprintln!("[server] write: {}", e);
                            return;
                        }
                    }
                    Err(_) => return,
                }
            }
        });
    }
}

async fn run_client(addr: &str) {
    let url = format!("ws://{}", addr);
    let mut conn = transport::WsClientConn::connect(&url)
        .await
        .expect("connect");

    println!("[client] connected to {}", addr);

    let messages = vec!["Hello FProto!", "Encrypted Rust!", "Goodbye!"];
    for msg in messages {
        conn.write_message(msg.as_bytes()).await.expect("write");
        let reply = conn.read_message().await.expect("read");
        println!("[client] echo: {}", String::from_utf8_lossy(&reply));
    }

    println!("[client] done");
}
