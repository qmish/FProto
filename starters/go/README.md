# FProto Go Starter Pack

Шаблонный проект для быстрого старта с протоколом FProto на Go.
Использует `proto-core` как SDK-зависимость.

## Запуск

```bash
# Сервер
go run . -mode server -addr localhost:9100

# Клиент (в другом терминале)
go run . -mode client -addr localhost:9100
```

## Тесты

```bash
go test -v ./...
```

## Что включено

- Echo-сервер с WebSocket транспортом и Noise XX handshake
- Echo-клиент с шифрованием/дешифрованием сообщений
- Conformance-тесты: handshake, AEAD, rekey, zeroize, state machine, round-trip, large payload

## Структура

```
main.go         # Echo server + client
main_test.go    # Conformance + integration тесты
go.mod          # Зависимость от proto-core
```
