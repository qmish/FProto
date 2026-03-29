# proto-core API Reference

## Обзор

`proto-core` — переиспользуемое ядро протокола FProto, содержащее транспортный, сессионный, криптографический и observability слои.

Модуль: `github.com/qmish/FProto/proto-core`

---

## Пакет `transport`

### Интерфейсы

#### `Conn`
```go
type Conn interface {
    ReadMessage() ([]byte, error)
    WriteMessage(data []byte) error
    Close() error
    Type() Type
    RemoteAddr() string
}
```

### Типы

- `Type` — `"websocket"`, `"quic"`, `"grpc"`
- `WSConn` — реализация Conn поверх gorilla/websocket
- `QUICConn` — реализация Conn поверх quic-go
- `GRPCStreamConn` — серверная реализация Conn для gRPC streaming
- `GRPCClientConn` — клиентская реализация Conn для gRPC streaming
- `ConnPool` — пул соединений для одной сессии

### Функции

| Функция | Описание |
|---------|----------|
| `UpgradeHTTP(w, r)` | WebSocket upgrade для HTTP-запроса |
| `DialWS(url)` | Подключение к WebSocket-серверу |
| `ListenQUIC(addr, tlsConf)` | Запуск QUIC-листенера |
| `AcceptQUIC(ctx, listener)` | Приём QUIC-соединения |
| `DialQUIC(ctx, addr, tlsConf)` | Подключение к QUIC-серверу |
| `ListenGRPC(addr)` | Запуск gRPC-сервера |
| `GenerateSelfSignedTLS()` | Генерация TLS для тестирования |
| `NewConnPool(maxConns)` | Создание пула соединений |
| `ExtractRemoteAddr(ctx)` | Адрес из gRPC peer/metadata |

---

## Пакет `crypto`

### Типы

#### `KeyPair`
```go
type KeyPair struct {
    Private []byte
    Public  []byte
}
```

#### `NoiseSession`
```go
type NoiseSession struct {
    Send *noise.CipherState
    Recv *noise.CipherState
}
```

### Функции

| Функция | Описание |
|---------|----------|
| `GenerateKeyPair()` | X25519 ключевая пара |
| `ServerHandshake(key, read, write)` | Серверный Noise XX хэндшейк |
| `ClientHandshake(key, read, write)` | Клиентский Noise XX хэндшейк |
| `NewAEAD(key)` | ChaCha20-Poly1305 AEAD из 32-байт ключа |
| `EncryptAEAD(aead, sessionID, seq, pt)` | Шифрование с seq-nonce и AAD |
| `DecryptAEAD(aead, sessionID, seq, ct)` | Расшифровка с seq-nonce и AAD |
| `BuildNonce(seq)` | 96-bit nonce: 4 zero bytes + LE64(seq) |
| `BuildAAD(sessionID, seq)` | AAD: session_id \|\| LE64(seq) |
| `Rekey(currentKey)` | HKDF-SHA256 ротация ключа |
| `Zeroize(b)` | Зануление ключевого материала |

### Методы NoiseSession

| Метод | Описание |
|-------|----------|
| `Encrypt(plaintext)` | Шифрование исходящим ключом |
| `Decrypt(ciphertext)` | Расшифровка входящим ключом |

---

## Пакет `session`

### Типы

#### `Session`
```go
type Session struct {
    SessionID, UserID, DeviceID []byte
    Status                      Status
    CryptoState                 []byte
    CreatedAt, LastActive, ExpiresAt time.Time
    Connections                 []Connection
}
```

#### `Status`
`"connecting"` → `"active"` → `"sleeping"` → `"expired"`

#### `Event`
| Событие | Описание |
|---------|----------|
| `EventHandshakeOK` | Успешный хэндшейк |
| `EventHandshakeFail` | Ошибка хэндшейка (уничтожает сессию) |
| `EventPacketReceived` | Получен пакет данных |
| `EventIdleTimeout` | Таймаут бездействия |
| `EventExpiryTimeout` | Истечение срока сессии |
| `EventExplicitLogout` | Явный выход |

### Интерфейсы

#### `Store`
```go
type Store interface {
    Create(ctx, session) error
    Get(ctx, sessionID) (*Session, error)
    UpdateStatus(ctx, sessionID, status) error
    UpdateCryptoState(ctx, sessionID, cryptoState) error
    UpdateLastActive(ctx, sessionID) error
    Delete(ctx, sessionID) error
    AddConnection(ctx, sessionID, conn) error
    RemoveConnection(ctx, sessionID, connID) error
    GetConnections(ctx, sessionID) ([]Connection, error)
    GetUserSessions(ctx, userID) ([][]byte, error)
}
```

### Реализации Store

| Тип | Хранилище | Назначение |
|-----|-----------|------------|
| `RedisStore` | Redis | Горячий путь (low latency) |
| `PGStore` | PostgreSQL | Восстановление (durability) |

### Manager

| Метод | Описание |
|-------|----------|
| `CreateSession(ctx, userID, deviceID)` | Создание сессии |
| `GetSession(ctx, sessionID)` | Получение сессии |
| `TransitionSession(ctx, sessionID, event)` | Переход state machine |
| `UpdateCryptoState(ctx, sessionID, state)` | Обновление крипто-состояния |
| `RegisterConnection(ctx, sessionID, conn)` | Регистрация соединения |
| `UnregisterConnection(ctx, sessionID, connID)` | Удаление соединения |

### Функции

| Функция | Описание |
|---------|----------|
| `Transition(status, event)` | Переход state machine (чистая функция) |
| `DefaultConfig()` | Конфигурация по умолчанию |

---

## Пакет `observability`

### Функции

| Функция | Описание |
|---------|----------|
| `InitTracer(ctx, name, version, endpoint)` | Настройка OTLP трейсинга |
| `Tracer(name)` | Получение именованного трейсера |
| `InitMeter()` | Prometheus MeterProvider + HTTP handler |
| `NewLogger(serviceName)` | zap.Logger с JSON-энкодингом |
| `LoggerWithTrace(logger, ctx)` | Добавление trace_id/span_id |
| `GRPCServerOptions(logger)` | gRPC серверные опции с OTel |
| `GRPCDialOptions()` | gRPC клиентские опции с OTel |
| `NewMetrics(serviceName)` | Регистрация всех метрик |

### Метрики (`Metrics`)

| Поле | Тип | Описание |
|------|-----|----------|
| `MessagesSent` | Counter | Отправленные сообщения |
| `MessagesReceived` | Counter | Полученные сообщения |
| `HandshakeDuration` | Histogram | Длительность хэндшейка |
| `ActiveSessions` | UpDownCounter | Активные сессии |
| `ActiveConnections` | UpDownCounter | Активные соединения |
| `KafkaConsumerLag` | Gauge | Лаг Kafka consumer |
| `OutboxPending` | Gauge | Ожидающие записи outbox |
| `OutboxPublished` | Counter | Опубликованные записи |
| `OutboxDeadLetter` | Counter | Dead letter записи |
| `DispatchDuration` | Histogram | Длительность диспетчеризации |
| `MediaUploadDuration` | Histogram | Длительность загрузки медиа |

---

## Пакет `core` (верхнеуровневый)

### Хуки

| Интерфейс | Методы |
|-----------|--------|
| `MessageHandler` | `HandleMessage(ctx, sessionID, payload)` |
| `SessionHook` | `OnCreate`, `OnActive`, `OnExpire` |
| `TransportHook` | `OnConnect`, `OnDisconnect` |

### Конфигурация

```go
cfg := core.DefaultConfig()
cfg.WebSocketAddr = ":8080"
cfg.QUICAddr = ":8443"
cfg.RedisAddr = "redis:6379"
cfg.PgDSN = "postgres://..."
cfg.OTLPEndpoint = "jaeger:4317"
```
