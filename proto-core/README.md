# proto-core

Транспортно-агностичный, криптографически защищённый фреймворк для построения систем реального времени.

## Установка

```bash
go get github.com/qmish/FProto/proto-core
```

## Пакеты

| Пакет | Описание |
|-------|----------|
| `transport` | Абстракция соединений: WebSocket, QUIC, gRPC streaming, пул соединений |
| `session` | Управление сессиями: state machine, Redis/PostgreSQL хранилища |
| `crypto` | Noise_XX_25519_ChaChaPoly_BLAKE2s хэндшейк, AEAD, ротация ключей |
| `observability` | OpenTelemetry tracing, Prometheus метрики, zap логирование |

## Быстрый старт

### WebSocket echo-сервер с Noise шифрованием

```go
package main

import (
    "fmt"
    "net/http"

    "github.com/qmish/FProto/proto-core/crypto"
    "github.com/qmish/FProto/proto-core/transport"
)

func main() {
    serverKey, _ := crypto.GenerateKeyPair()

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        conn, _ := transport.UpgradeHTTP(w, r)
        defer conn.Close()

        session, _ := crypto.ServerHandshake(serverKey, conn.ReadMessage, conn.WriteMessage)

        for {
            ct, err := conn.ReadMessage()
            if err != nil {
                return
            }
            pt, _ := session.Decrypt(ct)
            resp, _ := session.Encrypt(fmt.Appendf(nil, "echo: %s", pt))
            conn.WriteMessage(resp)
        }
    })

    http.ListenAndServe(":8080", nil)
}
```

### Клиент

```go
conn, _ := transport.DialWS("ws://localhost:8080/ws")
defer conn.Close()

clientKey, _ := crypto.GenerateKeyPair()
session, _ := crypto.ClientHandshake(clientKey, conn.ReadMessage, conn.WriteMessage)

ct, _ := session.Encrypt([]byte("Hello!"))
conn.WriteMessage(ct)

respCT, _ := conn.ReadMessage()
pt, _ := session.Decrypt(respCT)
fmt.Println(string(pt)) // echo: Hello!
```

### QUIC транспорт

```go
tlsConf := transport.GenerateSelfSignedTLS()
listener, _ := transport.ListenQUIC(":8443", tlsConf)

conn, _ := transport.AcceptQUIC(ctx, listener)
// conn реализует тот же интерфейс transport.Conn
```

### Управление сессиями

```go
redisStore := session.NewRedisStore(redisClient, 24*time.Hour)
pgStore := session.NewPGStore(pgPool)
mgr := session.NewManager(redisStore, pgStore, session.DefaultConfig())

s, _ := mgr.CreateSession(ctx, userID, deviceID)
mgr.TransitionSession(ctx, s.SessionID, session.EventHandshakeOK)
```

### Observability

```go
shutdown, _ := observability.InitTracer(ctx, "my-service", "1.0.0", "localhost:4317")
defer shutdown(ctx)

mp, metricsHandler, _ := observability.InitMeter()
http.Handle("/metrics", metricsHandler)

logger, _ := observability.NewLogger("my-service")
logger.Info("started")
```

## Хуки

proto-core предоставляет интерфейсы для интеграции с бизнес-логикой:

- `MessageHandler` — вызывается при получении расшифрованного сообщения
- `SessionHook` — колбэки жизненного цикла сессий (OnCreate, OnActive, OnExpire)
- `TransportHook` — колбэки транспортных соединений (OnConnect, OnDisconnect)

## Зависимости

proto-core имеет минимальный набор зависимостей — **без** Kafka, MinIO или messenger-специфичных пакетов:

- `github.com/flynn/noise` — Noise Protocol
- `github.com/gorilla/websocket` — WebSocket
- `github.com/quic-go/quic-go` — QUIC
- `github.com/redis/go-redis/v9` — Redis
- `github.com/jackc/pgx/v5` — PostgreSQL
- `go.opentelemetry.io/otel` — OpenTelemetry
- `go.uber.org/zap` — структурированное логирование

## Лицензия

MIT
