use std::fmt;
use thiserror::Error;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Status {
    Connecting,
    Active,
    Sleeping,
    Expired,
}

impl fmt::Display for Status {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Status::Connecting => write!(f, "connecting"),
            Status::Active => write!(f, "active"),
            Status::Sleeping => write!(f, "sleeping"),
            Status::Expired => write!(f, "expired"),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Event {
    HandshakeOK,
    HandshakeFail,
    PacketReceived,
    IdleTimeout,
    ExpiryTimeout,
    ExplicitLogout,
}

impl fmt::Display for Event {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Event::HandshakeOK => write!(f, "handshake_ok"),
            Event::HandshakeFail => write!(f, "handshake_fail"),
            Event::PacketReceived => write!(f, "packet_received"),
            Event::IdleTimeout => write!(f, "idle_timeout"),
            Event::ExpiryTimeout => write!(f, "expiry_timeout"),
            Event::ExplicitLogout => write!(f, "explicit_logout"),
        }
    }
}

#[derive(Error, Debug)]
#[error("invalid transition: {status} + {event}")]
pub struct InvalidTransition {
    pub status: Status,
    pub event: Event,
}

pub struct TransitionResult {
    pub next: Status,
    pub destroyed: bool,
}

pub fn transition(current: Status, event: Event) -> Result<TransitionResult, InvalidTransition> {
    match (current, event) {
        (Status::Connecting, Event::HandshakeOK) => Ok(TransitionResult {
            next: Status::Active,
            destroyed: false,
        }),
        (Status::Connecting, Event::HandshakeFail) => Ok(TransitionResult {
            next: Status::Expired,
            destroyed: true,
        }),
        (Status::Active, Event::PacketReceived) => Ok(TransitionResult {
            next: Status::Active,
            destroyed: false,
        }),
        (Status::Active, Event::IdleTimeout) => Ok(TransitionResult {
            next: Status::Sleeping,
            destroyed: false,
        }),
        (Status::Active, Event::ExplicitLogout) => Ok(TransitionResult {
            next: Status::Expired,
            destroyed: false,
        }),
        (Status::Sleeping, Event::PacketReceived) => Ok(TransitionResult {
            next: Status::Active,
            destroyed: false,
        }),
        (Status::Sleeping, Event::ExpiryTimeout) => Ok(TransitionResult {
            next: Status::Expired,
            destroyed: false,
        }),
        _ => Err(InvalidTransition {
            status: current,
            event,
        }),
    }
}

#[derive(Debug, Clone)]
pub struct Config {
    pub idle_timeout_secs: u64,
    pub expiry_timeout_secs: u64,
    pub handshake_timeout_secs: u64,
    pub max_connections: usize,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            idle_timeout_secs: 300,
            expiry_timeout_secs: 86400,
            handshake_timeout_secs: 30,
            max_connections: 10000,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_connecting_handshake_ok() {
        let r = transition(Status::Connecting, Event::HandshakeOK).unwrap();
        assert_eq!(r.next, Status::Active);
        assert!(!r.destroyed);
    }

    #[test]
    fn test_connecting_handshake_fail() {
        let r = transition(Status::Connecting, Event::HandshakeFail).unwrap();
        assert!(r.destroyed);
    }

    #[test]
    fn test_active_idle_timeout() {
        let r = transition(Status::Active, Event::IdleTimeout).unwrap();
        assert_eq!(r.next, Status::Sleeping);
        assert!(!r.destroyed);
    }

    #[test]
    fn test_active_packet_received() {
        let r = transition(Status::Active, Event::PacketReceived).unwrap();
        assert_eq!(r.next, Status::Active);
    }

    #[test]
    fn test_active_explicit_logout() {
        let r = transition(Status::Active, Event::ExplicitLogout).unwrap();
        assert_eq!(r.next, Status::Expired);
    }

    #[test]
    fn test_sleeping_packet_received() {
        let r = transition(Status::Sleeping, Event::PacketReceived).unwrap();
        assert_eq!(r.next, Status::Active);
    }

    #[test]
    fn test_sleeping_expiry_timeout() {
        let r = transition(Status::Sleeping, Event::ExpiryTimeout).unwrap();
        assert_eq!(r.next, Status::Expired);
    }

    #[test]
    fn test_invalid_transition() {
        assert!(transition(Status::Expired, Event::PacketReceived).is_err());
        assert!(transition(Status::Connecting, Event::IdleTimeout).is_err());
    }

    #[test]
    fn test_default_config() {
        let cfg = Config::default();
        assert_eq!(cfg.idle_timeout_secs, 300);
        assert_eq!(cfg.expiry_timeout_secs, 86400);
        assert_eq!(cfg.handshake_timeout_secs, 30);
        assert_eq!(cfg.max_connections, 10000);
    }

    #[test]
    fn test_full_lifecycle() {
        let r = transition(Status::Connecting, Event::HandshakeOK).unwrap();
        assert_eq!(r.next, Status::Active);

        let r = transition(r.next, Event::PacketReceived).unwrap();
        assert_eq!(r.next, Status::Active);

        let r = transition(r.next, Event::IdleTimeout).unwrap();
        assert_eq!(r.next, Status::Sleeping);

        let r = transition(r.next, Event::PacketReceived).unwrap();
        assert_eq!(r.next, Status::Active);

        let r = transition(r.next, Event::IdleTimeout).unwrap();
        let r = transition(r.next, Event::ExpiryTimeout).unwrap();
        assert_eq!(r.next, Status::Expired);
    }
}
