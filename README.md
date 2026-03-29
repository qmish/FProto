# FProto

Transport-agnostic, cryptographically secure messaging protocol with native Apache Kafka integration.

## Purpose

FProto is a modular communication protocol designed primarily for a messenger application, but engineered for reuse across diverse distributed systems: IoT telemetry, financial transactions, real-time multiplayer games, and notification platforms.

The protocol provides a complete stack -- from transport framing to end-to-end encryption -- decoupled from any specific business logic.

## Core Principles

- **Transport Independence** -- single protocol over WebSocket, QUIC, gRPC, TCP, or SCTP
- **Security by Default** -- Noise Protocol Framework (XX pattern), X25519, ChaCha20-Poly1305, Sender Keys for group E2E encryption
- **Kafka-Native** -- messages map naturally to Kafka topics with ordering guarantees, idempotency, and offline sync
- **Modularity** -- loosely coupled layers (transport, session, crypto, application) that can be replaced or extended independently
- **Observability** -- built-in OpenTelemetry tracing, structured logs, Prometheus metrics

## Architecture

```
+--------------------------------------------------+
|          Application Layer (Protobuf)             |
|   ChatMessage | Command | MediaChunk | Ack | Ping |
+--------------------------------------------------+
|          Crypto Layer (Noise Framework)            |
|   Noise_XX | ChaCha20-Poly1305 | X25519 | BLAKE2s |
+--------------------------------------------------+
|          Session Layer                             |
|   session_id | device binding | key rotation       |
+--------------------------------------------------+
|          Transport Layer                           |
|   WebSocket | QUIC | gRPC streaming | TCP          |
+--------------------------------------------------+
```

**Server-side components:** Transport Gateway, Session Manager (Redis + PostgreSQL), Crypto Handler, Dispatcher, Kafka Cluster, Consumer Services (Sync, Notifications, Commands).

## Key Features

- Three transport backends: WebSocket (browsers, mobile), QUIC (mobile, IoT), gRPC (server-to-server)
- Noise_XX_25519_ChaChaPoly_BLAKE2s handshake with periodic key rotation
- Sender Keys for scalable group E2E encryption
- Outbox/Inbox pattern for guaranteed message delivery
- Offline sync via Kafka consumer seek
- Chunked media upload with S3-compatible storage
- Session persistence across connection drops (device-bound sessions)

## Technology Stack

| Component | Technology |
|-----------|-----------|
| Server | Go / Rust |
| Serialization | Protocol Buffers |
| Crypto | Noise Protocol, X25519, ChaCha20-Poly1305, BLAKE2s |
| Message Broker | Apache Kafka (KRaft) |
| Session Store | Redis Cluster |
| Persistence | PostgreSQL (sharded) |
| Object Storage | S3 / MinIO |
| Observability | OpenTelemetry, Prometheus, Grafana, Jaeger |
| Client SDKs | Swift (iOS), Kotlin (Android), TypeScript (Web) |

## Documentation

| Document | Description |
|----------|------------|
| [Protocol Plan](docs/План%20протокола%20Proto.md) | Original protocol specification and design rationale |
| [Roadmap](docs/Roadmap.md) | Implementation checklist with phases, tasks, and acceptance criteria |
| [HLD](docs/HLD.md) | High-Level Design: architecture, components, scaling and security strategies |
| [LLD](docs/LLD.md) | Low-Level Design: Protobuf schemas, Noise handshake, DB schemas, Kafka config |
| [Network Architecture](docs/Network-Architecture.md) | Transports, framing, load balancing, TLS, NAT traversal |
| [Data Flows](docs/Data-Flows.md) | Sequence diagrams for all key data flows |
| [Network Connectivity](docs/Network-Connectivity.md) | Component connectivity matrix, protocols, ports, service discovery |
| [Infrastructure](docs/Infrastructure.md) | Kubernetes, Kafka cluster, Redis, PostgreSQL, CI/CD, monitoring |

## Reuse Scenarios

The protocol core (transport + sessions + crypto + Kafka routing) can be adapted for:

- **IoT** -- device telemetry and control commands
- **Finance** -- order execution, confirmations, market data streaming
- **Gaming** -- real-time multiplayer state synchronization
- **Notifications** -- push notification delivery platform

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
