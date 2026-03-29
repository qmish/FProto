# FProto Cross-Language Interoperability Tests

Тесты совместимости между Go echo-сервером и клиентами на разных языках.

## Архитектура

```
┌─────────────────┐
│  Go Echo Server  │  (proto-core WebSocket + Noise XX)
│  localhost:9500  │
└────────┬────────┘
         │ WebSocket
    ┌────┴────┐
    │         │
    ▼         ▼
  Client    Client   ...
  (any)     (any)
```

## Проверяемые аспекты

1. **Noise XX Handshake** — Корректность 3-way рукопожатия между реализациями на разных языках
2. **AEAD Encrypt/Decrypt** — Совместимость шифрования ChaCha20-Poly1305 (одинаковые nonce/AAD)
3. **Message Round-trip** — Клиент отправляет → сервер дешифрует → шифрует → клиент дешифрует

## Запуск

### 1. Запустить Go echo-сервер

```bash
cd server && go run . -addr localhost:9500
```

### 2. Запустить клиент на нужном языке

```bash
# Go
cd ../starters/go && go run . -mode client -addr localhost:9500

# Python
cd ../starters/python && python examples/echo_client.py

# Node.js
cd ../starters/node && npx ts-node examples/echo.ts client 127.0.0.1:9500
```

## AEAD Compatibility Test Vectors

Для проверки совместимости AEAD между языками используются фиксированные тестовые вектора.

Все SDK должны получать одинаковые результаты при:
- Key: `[0, 1, 2, ..., 31]`
- Session ID: `"interop-test"`
- Seq: `42`
- Plaintext: `"hello cross-language"`

Тест-вектор также проверяет Rekey:
- Key: `[0, 1, 2, ..., 31]`
- Rekey(key) должен давать одинаковый результат на всех языках
