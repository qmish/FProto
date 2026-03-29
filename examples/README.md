# Примеры использования proto-core

Каждый пример — самостоятельный Go-модуль, демонстрирующий адаптацию `proto-core`
для конкретной предметной области.

## Структура

| Пример | Транспорт | Особенности |
|--------|-----------|-------------|
| [iot-telemetry](iot-telemetry/) | QUIC | Телеметрия датчиков, управляющие команды, самоподписанные TLS-сертификаты |
| [finance-orders](finance-orders/) | WebSocket | Торговые ордера с Ed25519-подписями, проверка подлинности |
| [game-realtime](game-realtime/) | QUIC | Игровые действия, обновления состояния мира, атака/перемещение |
| [notifications](notifications/) | WebSocket | Push-уведомления, подписка на каналы, фильтрация, ACK |

## Запуск

Все примеры используют Go Workspace (`go.work`) в корне проекта.

```bash
# Сборка
cd <корень проекта>
go build github.com/qmish/FProto/examples/iot-telemetry

# Тесты
go test github.com/qmish/FProto/examples/iot-telemetry -v

# Запуск сервера
./iot-telemetry --mode server --addr :4433

# Запуск клиента
./iot-telemetry --mode client --addr 127.0.0.1:4433 --device sensor-001
```

## Общие принципы

Каждый пример использует одинаковый паттерн:

1. **Генерация ключей** — `crypto.GenerateKeyPair()` для Noise XX
2. **Установка транспорта** — QUIC (`transport.ListenQUIC` / `transport.DialQUIC`) или WebSocket (`transport.UpgradeHTTP` / `transport.DialWS`)
3. **Noise-хэндшейк** — `crypto.ServerHandshake()` / `crypto.ClientHandshake()`
4. **Шифрованный обмен** — `session.Encrypt()` / `session.Decrypt()`
5. **Прикладные сообщения** — TLV-фрейм: `[1B type][4B length][payload]`
