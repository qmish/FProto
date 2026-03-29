# FProto Conformance Test Suite

Набор тестов для валидации реализаций протокола FProto.

## Уровни соответствия

| Уровень | Тесты | Описание |
|---------|-------|----------|
| **Minimal** | Handshake, Encrypt/Decrypt, Rekey, Zeroize | Базовая криптография |
| **Standard** | + Session state machine, Seq ordering, Frame round-trip | Управление сессиями |
| **Full** | + Group encryption, Offline sync patterns, Transport multi | Полный протокол |

## Запуск

```bash
cd <корень проекта>

# Все тесты
go test github.com/qmish/FProto/tests/conformance -v

# Только минимальный уровень
go test github.com/qmish/FProto/tests/conformance -run "Minimal" -v

# Только стандартный
go test github.com/qmish/FProto/tests/conformance -run "Standard" -v

# Полный
go test github.com/qmish/FProto/tests/conformance -run "Full" -v
```

## Структура тестов

- `handshake_test.go` — Noise XX handshake (3-way, mismatched keys, timeout)
- `crypto_test.go` — AEAD encrypt/decrypt, rekey, zeroize
- `session_test.go` — State machine transitions, config defaults
- `transport_test.go` — WebSocket & QUIC round-trip, Connection pool
- `group_test.go` — Sender Keys: generation, ratchet, encrypt/decrypt, rotation
- `integration_test.go` — End-to-end scenarios combining all layers
