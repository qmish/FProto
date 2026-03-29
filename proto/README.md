# FProto Protocol Buffers Schemas

Официальные Protobuf-схемы протокола FProto.

## Структура

```
proto/
├── buf.yaml              # Конфигурация Buf (lint, breaking changes)
├── buf.gen.yaml          # Генерация Go-кода
├── buf.gen.docs.yaml     # Генерация документации
├── CHANGELOG.md          # История изменений схем
└── fproto/v1/            # Схемы версии v1
    ├── frame.proto           # Транспортный фрейм, HandshakeMessage
    ├── app_message.proto     # AppMessage (oneof body)
    ├── chat.proto            # ChatMessage, Attachment
    ├── command.proto         # Command (CREATE_GROUP, ADD_MEMBER, ...)
    ├── media.proto           # MediaChunk
    ├── ack.proto             # Ack, Ping, Pong
    ├── sync.proto            # SyncRequest, SenderKeyDistribution
    ├── signaling.proto       # SignalingMessage (WebRTC)
    ├── session_service.proto # gRPC SessionService
    ├── crypto_service.proto  # gRPC CryptoService
    ├── dispatcher_service.proto # gRPC DispatcherService
    ├── media_service.proto   # gRPC MediaService
    └── push_service.proto    # gRPC PushService
```

## Версионирование

Схемы следуют [Semantic Versioning](https://semver.org) и [Buf Breaking Change Detection](https://buf.build/docs/breaking/overview):

| Версия пакета | Описание |
|---------------|----------|
| `fproto.v1`   | Текущая стабильная версия |

### Политика совместимости

- **Patch/Minor**: Добавление новых полей, сообщений, enum-значений. Обратно совместимые изменения.
- **Major (v2)**: Удаление полей, переименование, изменение типов. Создаётся новый пакет `fproto.v2`.

### Проверка обратной совместимости

```bash
# Проверка на breaking changes относительно предыдущей версии
buf breaking --against '.git#branch=main'

# Lint-проверка
buf lint
```

## Использование

### Buf Schema Registry

```bash
# Установка зависимости
buf dep update

# В buf.yaml вашего проекта:
# deps:
#   - buf.build/qmish/fproto
```

### Генерация кода

#### Go

```bash
buf generate proto
```

Результат: `server/internal/protocol/gen/fproto/v1/*.pb.go`

#### Другие языки

Настройте `buf.gen.yaml` с нужными плагинами:

```yaml
version: v2
plugins:
  # TypeScript
  - remote: buf.build/bufbuild/es
    out: sdk/typescript/src/gen
    opt: target=ts

  # Swift
  - remote: buf.build/apple/swift
    out: sdk/swift/Sources/Proto
    opt: Visibility=Public

  # Kotlin
  - remote: buf.build/grpc/kotlin
    out: sdk/kotlin/src/main/kotlin
```

## Интеграция

### Go

```go
import pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"

frame := &pb.Frame{
    SessionId:        sessionID,
    Seq:              seq,
    EncryptedPayload: ciphertext,
}
```

### Прямой импорт .proto

```protobuf
syntax = "proto3";
import "fproto/v1/frame.proto";
import "fproto/v1/app_message.proto";
```
