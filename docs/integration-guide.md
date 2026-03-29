# Руководство по интеграции FProto

## Обзор

FProto -- транспортно-агностичный, криптографически защищённый протокол обмена сообщениями.
Данное руководство описывает процесс интеграции клиентского приложения с серверной инфраструктурой FProto.

---

## Архитектура подключения

```
Клиент -> [WebSocket / QUIC / gRPC] -> Gateway -> Dispatcher -> Kafka -> SyncService -> Push
```

1. Клиент устанавливает транспортное соединение с **Gateway**
2. Выполняется **Noise XX handshake** для установления зашифрованного канала
3. Все последующие сообщения передаются в **Frame** → **AppMessage** формате
4. Сервер маршрутизирует сообщения через **Kafka** получателям

---

## Шаг 1: Установка соединения

### WebSocket

```
ws://gateway-host:8080/ws
```

### QUIC

```
gateway-host:8443 (ALPN: fproto)
```

---

## Шаг 2: Noise XX Handshake

FProto использует `Noise_XX_25519_ChaChaPoly_BLAKE2s`:

1. **Клиент → Сервер**: Ephemeral public key (32 байта)
2. **Сервер → Клиент**: `e, ee, s, es` (ephemeral key + encrypted static key)
3. **Клиент → Сервер**: `s, se` (encrypted static key, завершение)

После handshake обе стороны получают `CipherState` для шифрования/дешифрования.

### Реализация (Go SDK)

```go
import "github.com/qmish/FProto/sdk/go"

client := fproto.NewClient("ws://localhost:8080/ws")
err := client.Connect(context.Background())
```

---

## Шаг 3: Формат сообщений

### Frame (транспортный уровень)

```protobuf
message Frame {
  bytes  session_id        = 1;  // UUID сессии (16 байт)
  uint64 seq               = 2;  // Монотонный номер пакета
  uint64 ack               = 3;  // Подтверждение последнего принятого seq
  bytes  encrypted_payload = 4;  // Зашифрованный AppMessage
}
```

### AppMessage (прикладной уровень)

```protobuf
message AppMessage {
  bytes                     message_id = 1;  // UUIDv7
  google.protobuf.Timestamp timestamp  = 2;
  oneof body {
    ChatMessage              chat         = 3;
    Command                  command      = 4;
    MediaChunk               media        = 5;
    Ack                      ack          = 6;
    Ping                     ping         = 7;
    Pong                     pong         = 8;
    SenderKeyDistribution    sender_key   = 9;
    SyncRequest              sync_request = 10;
    SignalingMessage          signaling    = 11;
  }
}
```

---

## Шаг 4: Отправка текстового сообщения

1. Создайте `ChatMessage` с `recipient_id` или `group_id`
2. Оберните в `AppMessage` с `message_id` (UUIDv7) и `timestamp`
3. Сериализуйте в `Frame.encrypted_payload` (Protobuf → Encrypt → Frame)
4. Отправьте через транспорт

### Пример (Go)

```go
chat := &pb.ChatMessage{
    RecipientId: recipientID,
    Action:      pb.ChatMessage_SEND,
    Text:        "Привет!",
}
msg := &pb.AppMessage{
    MessageId: generateUUIDv7(),
    Body:      &pb.AppMessage_Chat{Chat: chat},
}
err := client.Send(msg)
```

---

## Шаг 5: Получение подтверждений (ACK)

Сервер отвечает `Ack` с `ack_message_id` и статусом:

| Статус | Описание |
|--------|----------|
| `RECEIVED` | Сервер принял сообщение |
| `DELIVERED` | Доставлено получателю |
| `READ` | Прочитано получателем |
| `FAILED` | Ошибка обработки |

---

## Шаг 6: Медиафайлы

1. Вызовите `MediaService.InitUpload` для получения presigned URL
2. Загрузите файл напрямую в S3/MinIO по presigned URL
3. Вызовите `MediaService.CompleteUpload` для подтверждения
4. Отправьте `MediaChunk` другим участникам с `download_url`

---

## Шаг 7: Групповое шифрование (Sender Keys)

Для групповых чатов используется протокол Sender Keys:

1. Каждый участник генерирует `SenderKey` (Ed25519 + chain key)
2. Распространяет через `SenderKeyDistribution` всем членам группы
3. Шифрование: HMAC ratchet → ChaCha20-Poly1305 + Ed25519 подпись
4. При изменении состава группы выполняется ротация ключей

---

## Шаг 8: Offline синхронизация

При повторном подключении клиент отправляет `SyncRequest`:

```protobuf
message SyncRequest {
  bytes  last_seen_message_id = 1;  // Последний полученный UUIDv7
  uint32 max_messages         = 2;  // Лимит для пагинации
}
```

Сервер возвращает пропущенные сообщения из Inbox (PostgreSQL).

---

## Шаг 9: WebRTC сигнализация

Для аудио/видео звонков используйте `SignalingMessage`:

1. Отправьте `offer` (SDP) получателю
2. Получатель отвечает `answer` (SDP)
3. Обмен ICE candidates через `ice_candidate`
4. Управление потоком: `stream_control` (START/END/JOIN)

---

## Конфигурация

### Переменные окружения

| Переменная | Описание | По умолчанию |
|-----------|----------|-------------|
| `GATEWAY_URL` | WebSocket адрес Gateway | `ws://localhost:8080/ws` |
| `QUIC_ADDR` | QUIC адрес Gateway | `localhost:8443` |
| `OTLP_ENDPOINT` | OpenTelemetry OTLP endpoint | `localhost:4317` |

---

## Мониторинг

Все сервисы экспортируют:
- **Метрики**: Prometheus `/metrics` endpoint
- **Трейсы**: OpenTelemetry → Jaeger (OTLP gRPC :4317)
- **Логи**: JSON (stdout) → Promtail → Loki

Grafana dashboards доступны на порту `:3000` (admin/admin).
