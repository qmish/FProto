# FProto Python Starter Pack

Python SDK для протокола FProto — пакет `fproto`.

## Установка

```bash
pip install -e ".[dev]"
```

## Тесты

```bash
pytest tests/ -v
```

## Echo-пример

```bash
# Сервер
python examples/echo_server.py

# Клиент (в другом терминале)
python examples/echo_client.py
```

## Структура

```
fproto/
  __init__.py       # Публичный API
  transport.py      # WebSocket Conn (asyncio)
  crypto.py         # Noise XX, AEAD, Rekey, Zeroize
  session.py        # Конечный автомат состояний
examples/
  echo_server.py    # Echo сервер
  echo_client.py    # Echo клиент
tests/
  test_conformance.py  # Conformance тесты
```

## Зависимости

- `noiseprotocol` — Noise Protocol Framework
- `websockets` — WebSocket транспорт (asyncio)
- `cryptography` — ChaCha20-Poly1305 AEAD
