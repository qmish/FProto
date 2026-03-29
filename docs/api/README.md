# FProto API Reference

## Protobuf схемы

Все `.proto` файлы находятся в `proto/fproto/v1/`.

---

## Транспортный уровень

### Frame (`frame.proto`)

Базовый транспортный пакет. Все транспорты (WebSocket, QUIC, gRPC) передают одинаковые Frame.

| Поле | Тип | Описание |
|------|-----|----------|
| `session_id` | `bytes` | UUID сессии (16 байт, UUIDv4/v7) |
| `seq` | `uint64` | Монотонный номер пакета отправителя |
| `ack` | `uint64` | Подтверждение последнего принятого seq |
| `encrypted_payload` | `bytes` | Зашифрованный AppMessage (ChaCha20-Poly1305) |

### HandshakeMessage (`frame.proto`)

Сообщение Noise XX handshake (INIT → RESPONSE → FINISH).

---

## Прикладной уровень

### AppMessage (`app_message.proto`)

Основное прикладное сообщение. Содержит `message_id` (UUIDv7), `timestamp` и `oneof body`.

| Тип body | Proto | Описание |
|----------|-------|----------|
| `chat` | `ChatMessage` | Текстовое сообщение (1-to-1, группа) |
| `command` | `Command` | Административная команда |
| `media` | `MediaChunk` | Фрагмент медиафайла |
| `ack` | `Ack` | Подтверждение (RECEIVED/DELIVERED/READ) |
| `ping` | `Ping` | Проверка активности |
| `pong` | `Pong` | Ответ на Ping |
| `sender_key` | `SenderKeyDistribution` | Распространение группового ключа |
| `sync_request` | `SyncRequest` | Запрос offline синхронизации |
| `signaling` | `SignalingMessage` | WebRTC сигнализация |

### ChatMessage (`chat.proto`)

| Поле | Тип | Описание |
|------|-----|----------|
| `conversation_id` | `bytes` | ID чата |
| `sender_id` | `bytes` | ID отправителя |
| `recipient_id` | `bytes` | ID получателя (1-to-1) |
| `group_id` | `bytes` | ID группы |
| `action` | `ChatAction` | SEND / EDIT / DELETE / REACTION |
| `text` | `string` | Текст сообщения |
| `reply_to` | `bytes` | message_id цитируемого сообщения |
| `attachments` | `Attachment[]` | Вложения |

### Command (`command.proto`)

| Тип | Описание |
|-----|----------|
| `CREATE_GROUP` | Создание группы |
| `ADD_MEMBER` | Добавление участника |
| `REMOVE_MEMBER` | Удаление участника |
| `UPDATE_GROUP_SETTINGS` | Обновление настроек |
| `BAN_MEMBER` | Бан участника |
| `DELETE_MESSAGE` | Удаление сообщения |

### SignalingMessage (`signaling.proto`)

WebRTC сигнализация: SDP offer/answer, ICE candidates, stream control.

---

## gRPC сервисы

### SessionService (`session_service.proto`)

Управление сессиями пользователей. Redis + PostgreSQL.

| Метод | Описание |
|-------|----------|
| `CreateSession` | Создание сессии (user_id, device_id) → session_id |
| `GetSession` | Получение состояния сессии |
| `UpdateSessionStatus` | Обновление статуса (CONNECTING → ACTIVE → SLEEPING → EXPIRED) |
| `RegisterConnection` | Регистрация транспортного соединения |
| `UnregisterConnection` | Отключение соединения |
| `RouteToSession` | Маршрутизация payload к сессии |

### CryptoService (`crypto_service.proto`)

Криптографические операции. Noise_XX_25519_ChaChaPoly_BLAKE2s.

| Метод | Описание |
|-------|----------|
| `InitiateHandshake` | Инициализация Noise XX handshake |
| `ProcessHandshake` | Обработка шага handshake |
| `Encrypt` | Шифрование (AEAD, seq-based nonce) |
| `Decrypt` | Дешифрование с проверкой AEAD тега |
| `Rekey` | Ротация ключей (HKDF-SHA256) |

### DispatcherService (`dispatcher_service.proto`)

Маршрутизация сообщений в Kafka через Outbox.

| Метод | Описание |
|-------|----------|
| `Dispatch` | Маршрутизация AppMessage → Kafka topic |

### PushService (`push_service.proto`)

Доставка сообщений онлайн-клиентам.

| Метод | Описание |
|-------|----------|
| `PushToUser` | Доставка всем сессиям пользователя |
| `PushToSession` | Доставка конкретной сессии |

### MediaService (`media_service.proto`)

Управление медиафайлами. Presigned URL + S3/MinIO.

| Метод | Описание |
|-------|----------|
| `InitUpload` | Инициализация загрузки → presigned URL |
| `CompleteUpload` | Подтверждение завершения |
| `GetDownloadURL` | Presigned URL для скачивания |

---

## Kafka Topics

| Topic | Описание | Partition key |
|-------|----------|---------------|
| `user-<hex_id>` | Сообщения конкретному пользователю | message_id |
| `group-<hex_id>` | Сообщения в группу | message_id |
| `commands` | Административные команды | group_id |
| `events` | Системные события | message_id |
| `media-events` | Медиа события | stream_id |
| `dead-letter` | Сообщения с исчерпанными retry | original_topic |
