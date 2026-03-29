# Changelog — FProto Protobuf Schemas

Все изменения Protobuf-схем документируются в этом файле.
Формат основан на [Keep a Changelog](https://keepachangelog.com).

## [1.0.0] — 2026-03-29

### Сообщения

#### Транспортный уровень
- `Frame` — транспортный фрейм с session_id, seq, ack, encrypted_payload
- `HandshakeMessage` — хэндшейк-сообщения Noise XX (INIT, RESPONSE, FINISH)

#### Прикладной уровень
- `AppMessage` — обёртка с oneof body для всех типов сообщений
- `ChatMessage` — текстовые сообщения (SEND, EDIT, DELETE, REACTION)
- `Attachment` — вложения к сообщениям
- `Command` — административные команды (CREATE_GROUP, ADD_MEMBER, REMOVE_MEMBER, UPDATE_GROUP)
- `MediaChunk` — фрагменты медиа-файлов
- `Ack` — подтверждения (RECEIVED, DELIVERED, READ, ERROR)
- `Ping` / `Pong` — heartbeat
- `SyncRequest` — запрос оффлайн-синхронизации
- `SenderKeyDistribution` — распространение групповых ключей
- `SignalingMessage` — WebRTC-сигнализация (SDP, ICE, StreamControl)

### gRPC-сервисы
- `SessionService` — управление сессиями (Create, Get, Update, Delete, ListByUser)
- `CryptoService` — криптографические операции (Encrypt, Decrypt, HandshakeInit, HandshakeProcess)
- `DispatcherService` — маршрутизация сообщений (Dispatch, DispatchBatch)
- `MediaService` — медиа-загрузки (InitUpload, CompleteUpload, GetDownloadURL)
- `PushService` — push-уведомления (Send, SendBatch)

### Конфигурация
- `buf.yaml` v2 с lint (STANDARD) и breaking (FILE) правилами
- `buf.gen.yaml` для генерации Go-кода
