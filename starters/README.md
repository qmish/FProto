# FProto Starter Packs

Мультиязычные SDK-библиотеки и шаблоны проектов для протокола FProto.
Каждый стартер-пак реализует ядро протокола (`proto-core` API surface) на своём языке и содержит conformance-тесты.

## Поддерживаемые языки

| Язык | Пакет | Транспорт | Noise | Статус |
|------|-------|-----------|-------|--------|
| **Go** | `github.com/qmish/FProto/proto-core` | WebSocket, QUIC, gRPC | `flynn/noise` | Основная реализация |
| **Rust** | `fproto-core` (crate) | WebSocket, QUIC | `snow` | Starter Pack |
| **Python** | `fproto` (pip) | WebSocket, QUIC | `noiseprotocol` | Starter Pack |
| **Node.js/TS** | `@fproto/core` (npm) | WebSocket | `noise-protocol` | Starter Pack |
| **Java** | `io.fproto:fproto-core` (Maven) | WebSocket | `noise-java` | Starter Pack |
| **.NET** | `FProto.Core` (NuGet) | WebSocket, QUIC | `Noise.NET` | Starter Pack |
| **PHP** | `fproto/core` (Composer) | WebSocket | `noise-c` FFI | Starter Pack |

## Архитектура SDK

Каждый SDK реализует четыре слоя:

```
┌──────────────────────────────────┐
│       Application Layer          │  Protobuf-сообщения, бизнес-логика
├──────────────────────────────────┤
│       Session Layer              │  Конечный автомат состояний
├──────────────────────────────────┤
│       Crypto Layer               │  Noise XX handshake, AEAD, Rekey
├──────────────────────────────────┤
│       Transport Layer            │  WebSocket / QUIC / gRPC
└──────────────────────────────────┘
```

### API-поверхность (контракт)

**Transport:**
- `Conn` — интерфейс: `ReadMessage()`, `WriteMessage()`, `Close()`, `Type()`, `RemoteAddr()`
- Типы: `WebSocket`, `QUIC`, `gRPC`

**Crypto:**
- `GenerateKeyPair()` — генерация X25519 ключевой пары
- `ServerHandshake()` / `ClientHandshake()` — Noise_XX_25519_ChaChaPoly_BLAKE2s
- `NoiseSession.Encrypt()` / `Decrypt()` — шифрование после handshake
- `NewAEAD(key)` — ChaCha20-Poly1305
- `BuildNonce(seq)`, `BuildAAD(sessionID, seq)` — детерминированные nonce/AAD
- `EncryptAEAD()` / `DecryptAEAD()` — низкоуровневое AEAD
- `Rekey(currentKey)` — ротация ключей через HKDF-SHA256
- `Zeroize(bytes)` — обнуление секретов в памяти

**Session:**
- Состояния: `Connecting` → `Active` ↔ `Sleeping` → `Expired`
- События: `HandshakeOK`, `HandshakeFail`, `PacketReceived`, `IdleTimeout`, `ExpiryTimeout`, `ExplicitLogout`
- `Transition(status, event)` — переход конечного автомата

**Hooks:**
- `MessageHandler` — обработка входящих сообщений
- `SessionHook` — `OnCreate`, `OnActive`, `OnExpire`
- `TransportHook` — `OnConnect`, `OnDisconnect`

## Conformance-тесты

Каждый SDK содержит conformance-тесты трёх уровней:

| Уровень | Тесты |
|---------|-------|
| **Minimal** | Noise XX handshake, AEAD encrypt/decrypt, rekey, zeroize, nonce uniqueness |
| **Standard** | State machine transitions, WebSocket round-trip, large payload (32KB) |
| **Full** | Bidirectional encrypted exchange, multi-client, sequential ordering |

## Быстрый старт

### Go
```bash
cd starters/go && go run .
```

### Rust
```bash
cd starters/rust/fproto-core && cargo run --example echo
```

### Python
```bash
cd starters/python && pip install -e . && python examples/echo_server.py
```

### Node.js/TypeScript
```bash
cd starters/node && npm install && npx ts-node examples/echo.ts
```

### Java
```bash
cd starters/java && mvn compile exec:java -Dexec.mainClass="io.fproto.examples.Echo"
```

### .NET
```bash
cd starters/dotnet && dotnet run --project FProto.Examples
```

### PHP
```bash
cd starters/php && composer install && php examples/echo_server.php
```

## Интероперабельность

В `starters/interop/` находятся тесты, где Go echo-сервер принимает подключения от клиентов на всех языках, проверяя:

1. Корректность Noise XX handshake между реализациями
2. Round-trip зашифрованных сообщений
3. Совместимость конечного автомата сессий
