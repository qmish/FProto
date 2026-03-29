# FProto Rust Starter Pack

Rust SDK для протокола FProto — `fproto-core` crate.

## Зависимости

- `snow` — Noise Protocol Framework (Noise_XX_25519_ChaChaPoly_BLAKE2s)
- `chacha20poly1305` — AEAD шифрование
- `tokio-tungstenite` — WebSocket транспорт (async)
- `hkdf` + `sha2` — Rekey через HKDF-SHA256
- `zeroize` — обнуление секретов в памяти

## Сборка

```bash
cd fproto-core
cargo build
```

## Тесты

```bash
cargo test
```

## Echo-пример

```bash
# Сервер
cargo run --example echo -- server 127.0.0.1:9200

# Клиент
cargo run --example echo -- client 127.0.0.1:9200
```

## Структура

```
fproto-core/
  src/
    lib.rs          # Модуль-корень
    transport.rs    # WebSocket Conn, accept/dial
    crypto.rs       # Noise XX, AEAD, Rekey, Zeroize
    session.rs      # Конечный автомат состояний
  examples/
    echo.rs         # Echo server + client
  tests/
    conformance.rs  # Conformance test suite
```
