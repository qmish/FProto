use std::fmt;
use thiserror::Error;
use tokio::net::TcpStream;
use tokio_tungstenite::{
    connect_async, accept_async,
    tungstenite::Message,
    MaybeTlsStream, WebSocketStream,
};
use futures_util::{SinkExt, StreamExt};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TransportType {
    WebSocket,
    Quic,
    Grpc,
}

impl fmt::Display for TransportType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            TransportType::WebSocket => write!(f, "websocket"),
            TransportType::Quic => write!(f, "quic"),
            TransportType::Grpc => write!(f, "grpc"),
        }
    }
}

#[derive(Error, Debug)]
pub enum TransportError {
    #[error("websocket error: {0}")]
    WebSocket(#[from] tokio_tungstenite::tungstenite::Error),
    #[error("connection closed")]
    Closed,
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),
}

pub struct WsServerConn {
    ws: WebSocketStream<TcpStream>,
    addr: String,
}

impl WsServerConn {
    pub fn new(ws: WebSocketStream<TcpStream>, addr: String) -> Self {
        Self { ws, addr }
    }

    pub async fn read_message(&mut self) -> Result<Vec<u8>, TransportError> {
        match self.ws.next().await {
            Some(Ok(Message::Binary(data))) => Ok(data),
            Some(Ok(Message::Close(_))) | None => Err(TransportError::Closed),
            Some(Ok(_)) => self.read_message().await,
            Some(Err(e)) => Err(TransportError::WebSocket(e)),
        }
    }

    pub async fn write_message(&mut self, data: &[u8]) -> Result<(), TransportError> {
        self.ws.send(Message::Binary(data.to_vec().into())).await?;
        Ok(())
    }

    pub async fn close(&mut self) -> Result<(), TransportError> {
        self.ws.close(None).await?;
        Ok(())
    }

    pub fn transport_type(&self) -> TransportType {
        TransportType::WebSocket
    }

    pub fn remote_addr(&self) -> &str {
        &self.addr
    }
}

pub struct WsClientConn {
    ws: WebSocketStream<MaybeTlsStream<TcpStream>>,
    addr: String,
}

impl WsClientConn {
    pub async fn connect(url: &str) -> Result<Self, TransportError> {
        let (ws, _) = connect_async(url).await?;
        Ok(Self {
            ws,
            addr: url.to_string(),
        })
    }

    pub async fn read_message(&mut self) -> Result<Vec<u8>, TransportError> {
        match self.ws.next().await {
            Some(Ok(Message::Binary(data))) => Ok(data),
            Some(Ok(Message::Close(_))) | None => Err(TransportError::Closed),
            Some(Ok(_)) => self.read_message().await,
            Some(Err(e)) => Err(TransportError::WebSocket(e)),
        }
    }

    pub async fn write_message(&mut self, data: &[u8]) -> Result<(), TransportError> {
        self.ws.send(Message::Binary(data.to_vec().into())).await?;
        Ok(())
    }

    pub async fn close(&mut self) -> Result<(), TransportError> {
        self.ws.close(None).await?;
        Ok(())
    }

    pub fn transport_type(&self) -> TransportType {
        TransportType::WebSocket
    }

    pub fn remote_addr(&self) -> &str {
        &self.addr
    }
}

pub async fn accept_ws(stream: TcpStream, addr: String) -> Result<WsServerConn, TransportError> {
    let ws = accept_async(stream).await?;
    Ok(WsServerConn::new(ws, addr))
}
