# ADR-002: Выбор языков клиентских SDK

**Статус:** Принято
**Дата:** 2026-03-29

## Контекст

Протокол FProto должен поддерживать клиентские SDK для основных платформ: iOS, Android, Web, а также reference-реализацию на серверном языке.

## Решение

| Платформа | Язык | Транспорт | Криптография |
|-----------|-----|----------|-------------|
| Reference | Go | WS / QUIC / gRPC | `flynn/noise` |
| iOS | Swift | Starscream (WebSocket) | CryptoKit |
| Android | Kotlin | OkHttp (WebSocket) | `noise-java` |
| Web | TypeScript | Native WebSocket | `noise-c.wasm` |

## Обоснование

- **Go SDK** -- reference-реализация, используется для интеграционных тестов и серверного взаимодействия
- **Swift** -- нативный язык iOS, CryptoKit предоставляет X25519 и ChaCha20-Poly1305
- **Kotlin** -- нативный язык Android, OkHttp -- стандартная HTTP/WS библиотека
- **TypeScript** -- единственный вариант для браузеров, `noise-c.wasm` -- WebAssembly-порт Noise

## Последствия

- В Фазе 0 реализуется только Go SDK (reference)
- SDK для остальных платформ реализуются в Фазе 1
- Protobuf-схемы генерируются для всех четырёх языков с самого начала
- Все SDK следуют единому интерфейсу: Connect, Handshake, Send, Receive, Close
