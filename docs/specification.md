# Proto Protocol Specification v1.0

**Статус:** Draft  
**Дата:** 2026-03-29  
**Авторы:** FProto Team  
**Связанные документы:** [HLD](HLD.md) | [LLD](LLD.md) | [Network Architecture](Network-Architecture.md) | [Data Flows](Data-Flows.md) | [proto-core API](proto-core-api.md)

---

## Содержание

1. [Введение](#1-введение)
2. [Термины и определения](#2-термины-и-определения)
3. [Обзор архитектуры](#3-обзор-архитектуры)
4. [Транспортный уровень](#4-транспортный-уровень)
5. [Фрейминг](#5-фрейминг)
6. [Криптографический уровень](#6-криптографический-уровень)
7. [Сессионный уровень](#7-сессионный-уровень)
8. [Прикладной уровень](#8-прикладной-уровень)
9. [Групповое шифрование](#9-групповое-шифрование-sender-keys)
10. [Механизм подтверждений](#10-механизм-подтверждений)
11. [Оффлайн-синхронизация](#11-оффлайн-синхронизация)
12. [WebRTC-сигнализация](#12-webrtc-сигнализация)
13. [Коды ошибок](#13-коды-ошибок)
14. [Требования безопасности](#14-требования-безопасности)
15. [Требования к реализациям](#15-требования-к-реализациям)
16. [IANA-подобные реестры](#16-iana-подобные-реестры)

---

## 1. Введение

### 1.1 Назначение

Данный документ определяет формальную спецификацию протокола Proto (FProto) — транспортно-независимого, криптографически защищённого протокола обмена сообщениями. Протокол обеспечивает конфиденциальность, целостность и аутентификацию при передаче данных поверх произвольных транспортных каналов.

### 1.2 Область применения

Протокол Proto предназначен для:

- Систем мгновенного обмена сообщениями
- IoT-платформ (телеметрия, управляющие команды)
- Финансовых систем (ордера с цифровыми подписями)
- Игровых серверов (состояние мира, действия игроков)
- Систем push-уведомлений

### 1.3 Ключевые слова

Ключевые слова "MUST" (ДОЛЖЕН), "MUST NOT" (НЕ ДОЛЖЕН), "REQUIRED" (ОБЯЗАТЕЛЕН), "SHALL" (СЛЕДУЕТ), "SHOULD" (РЕКОМЕНДУЕТСЯ), "MAY" (МОЖЕТ) интерпретируются согласно [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### 1.4 Нотация

- Все числовые значения передаются в сетевом порядке байтов (big-endian), если не указано иное.
- Длины указаны в байтах.
- Строки кодируются в UTF-8.
- Бинарные данные кодируются согласно определениям Protocol Buffers v3.

---

## 2. Термины и определения

| Термин | Определение |
|--------|-------------|
| Frame | Минимальная единица передачи на транспортном уровне. Содержит метаданные сессии и зашифрованную полезную нагрузку. |
| Session | Логическая связь между клиентом и сервером, привязанная к криптографическому состоянию (CipherState). Одна сессия может использовать несколько транспортных соединений. |
| Connection | Физическое соединение на транспортном уровне (WebSocket, QUIC stream, gRPC stream). |
| Noise Handshake | Протокол аутентифицированного обмена ключами по шаблону Noise_XX. |
| CipherState | Пара состояний шифрования (Send/Recv), полученных после завершения Noise-хэндшейка. |
| AppMessage | Прикладное сообщение, сериализованное через Protocol Buffers. |
| Sender Key | Ключевой материал для группового E2E-шифрования на основе протокола Signal Sender Keys. |
| Outbox | Таблица исходящих сообщений в PostgreSQL для гарантии at-least-once доставки в Kafka. |
| Inbox | Таблица входящих сообщений для дедупликации и оффлайн-хранения. |

---

## 3. Обзор архитектуры

### 3.1 Слоистая модель

Протокол Proto организован в четыре уровня:

```
┌─────────────────────────────────────┐
│  Application Layer (Protobuf)       │  ChatMessage, Command, Media, Ack...
├─────────────────────────────────────┤
│  Crypto Layer (Noise XX)            │  Encrypt/Decrypt, Rekey, Zeroize
├─────────────────────────────────────┤
│  Session Layer                      │  State machine, Multiplexing
├─────────────────────────────────────┤
│  Transport Layer                    │  WebSocket, QUIC, gRPC
└─────────────────────────────────────┘
```

Каждый уровень взаимодействует только со смежными уровнями. Реализации MUST поддерживать эту модель разделения.

### 3.2 Инварианты протокола

1. Все данные, передаваемые после завершения хэндшейка, MUST быть зашифрованы.
2. Каждый Frame MUST содержать монотонно возрастающий порядковый номер `seq`.
3. Реализации MUST поддерживать как минимум один транспорт.
4. Криптографический уровень MUST использовать Noise_XX_25519_ChaChaPoly_BLAKE2s.

---

## 4. Транспортный уровень

### 4.1 Поддерживаемые транспорты

| Транспорт | Идентификатор | Порт | Протокол |
|-----------|---------------|------|----------|
| WebSocket | `websocket` | 443 (WSS) | RFC 6455 over TLS 1.3 |
| QUIC | `quic` | 443 (UDP) | RFC 9000, ALPN: `proto/1` |
| gRPC | `grpc` | 9000 | HTTP/2 + TLS, bidirectional streaming |

### 4.2 WebSocket

- Реализация MUST использовать subprotocol `proto.v1`.
- Максимальный размер фрейма: 65536 байт.
- Keep-alive: Ping каждые 30 секунд, Pong timeout 10 секунд.
- Реализации SHOULD поддерживать per-message deflate (RFC 7692).

### 4.3 QUIC

- ALPN: `proto/1`.
- Максимальный размер сообщения: 65536 байт.
- Idle timeout: 120 секунд.
- Фрейминг: 4-байтный length prefix (big-endian uint32) + payload.

```
+--------+--------+--------+--------+--...--+
|     Length (4 bytes, BE)          | Payload |
+--------+--------+--------+--------+--...--+
```

### 4.4 gRPC

- Сервис: bidirectional streaming RPC.
- Сообщения передаются как `bytes` в stream.
- Реализации SHOULD использовать OpenTelemetry stats handler для трассировки.

### 4.5 Интерфейс абстракции транспорта

Все транспорты MUST реализовывать следующий интерфейс:

```
interface Conn {
    ReadMessage() -> (bytes, error)
    WriteMessage(data: bytes) -> error
    Close() -> error
    Type() -> TransportType
    RemoteAddr() -> string
}
```

### 4.6 Connection Pool

Одна сессия MAY использовать несколько транспортных соединений одновременно. Реализация MUST:

- Маршрутизировать сообщения по лучшему доступному соединению.
- Обеспечивать упорядочение сообщений через `seq` в Frame.
- Приоритет транспортов (по умолчанию): QUIC > gRPC > WebSocket.

---

## 5. Фрейминг

### 5.1 Формат Frame

Все сообщения после хэндшейка передаются как сериализованные Protocol Buffers сообщения типа `Frame`:

```protobuf
message Frame {
    bytes  session_id       = 1;  // 16 bytes, UUIDv4 или UUIDv7
    uint64 seq              = 2;  // Монотонный номер пакета
    uint64 ack              = 3;  // Последний принятый seq от удалённой стороны
    bytes  encrypted_payload = 4; // AEAD-зашифрованный AppMessage
}
```

### 5.2 Требования

1. `session_id` MUST быть установлен после завершения хэндшейка.
2. `seq` MUST монотонно увеличиваться для каждого отправленного фрейма.
3. `ack` SHOULD отражать последний успешно обработанный `seq` от удалённой стороны.
4. `encrypted_payload` MUST быть зашифрован алгоритмом из текущего CipherState.

### 5.3 Хэндшейк-фреймы

До завершения хэндшейка передаются сообщения типа `HandshakeMessage`:

```protobuf
message HandshakeMessage {
    HandshakeType type    = 1;
    bytes         payload = 2;
    
    enum HandshakeType {
        HANDSHAKE_TYPE_UNSPECIFIED = 0;
        INIT     = 1;  // -> e
        RESPONSE = 2;  // -> e, ee, s, es
        FINISH   = 3;  // -> s, se
    }
}
```

---

## 6. Криптографический уровень

### 6.1 Cipher Suite

Протокол Proto использует фиксированный cipher suite:

**Noise_XX_25519_ChaChaPoly_BLAKE2s**

| Компонент | Алгоритм | Спецификация |
|-----------|----------|--------------|
| DH | X25519 | RFC 7748 |
| Cipher | ChaCha20-Poly1305 | RFC 8439 |
| Hash | BLAKE2s | RFC 7693 |
| Pattern | XX | Noise Protocol Framework, rev 34 |

### 6.2 Noise XX Handshake

Хэндшейк выполняется в три сообщения:

```
Initiator (Client)                  Responder (Server)
      |                                    |
      |---- INIT: -> e ------------------>|   Msg 1
      |                                    |
      |<--- RESPONSE: -> e, ee, s, es ----|   Msg 2
      |                                    |
      |---- FINISH: -> s, se ------------>|   Msg 3
      |                                    |
      |====== Encrypted Channel ===========|
```

1. **Msg 1 (INIT):** Инициатор отправляет эфемерный публичный ключ `e`.
2. **Msg 2 (RESPONSE):** Ответчик отправляет свой эфемерный ключ `e`, выполняет DH-обмены `ee`, `s` (статический ключ сервера), `es`.
3. **Msg 3 (FINISH):** Инициатор отправляет свой статический ключ `s`, выполняет DH-обмен `se`.

После Msg 3 обе стороны получают пару `CipherState` (Send, Recv) для дальнейшего шифрования.

### 6.3 AEAD-шифрование

После хэндшейка каждое сообщение шифруется через CipherState:

```
ciphertext = CipherState.Encrypt(ad=nil, plaintext)
plaintext  = CipherState.Decrypt(ad=nil, ciphertext)
```

Nonce управляется автоматически CipherState (внутренний счётчик).

### 6.4 Дополнительное AEAD-шифрование (уровень приложения)

Для случаев, когда требуется дополнительное шифрование поверх Noise (например, групповые сообщения):

**Алгоритм:** ChaCha20-Poly1305  
**Nonce:** Seq-based, 12 байт:

```
nonce[0..3]  = 0x00000000
nonce[4..11] = BigEndian(seq)
```

**AAD (Associated Authenticated Data):**

```
aad = session_id || BigEndian(seq)
```

### 6.5 Ротация ключей (Rekey)

Реализации MUST поддерживать ротацию ключей через HKDF-SHA256:

```
new_key = HKDF-Expand(
    PRK  = current_key,
    info = "proto-rekey",
    L    = 32
)
```

Ротация SHOULD выполняться:
- После каждых 2^32 сообщений.
- По запросу любой из сторон.
- При изменении состава группы.

### 6.6 Зероизация

Реализации MUST обнулять ключевой материал в памяти после использования:

```
for i := range key {
    key[i] = 0
}
```

Компилятор не должен оптимизировать эту операцию (использовать volatile-аналоги или barrier).

---

## 7. Сессионный уровень

### 7.1 Состояния сессии

Сессия проходит через следующие состояния:

```
          ┌──────────────────────────────────────┐
          │                                      │
CREATED ──┤── HandshakeComplete ──► ACTIVE ──────┤── Timeout ──► EXPIRED
          │                        │    ▲        │
          │                        │    │        │
          │                        ▼    │        │
          │                    SUSPENDED ┘        │
          │                                      │
          └── HandshakeFailed ──► EXPIRED ────────┘
```

| Состояние | Описание |
|-----------|----------|
| CREATED | Сессия создана, ожидание хэндшейка. |
| ACTIVE | Хэндшейк завершён, обмен сообщениями. |
| SUSPENDED | Все соединения потеряны, ожидание реконнекта. |
| EXPIRED | Сессия завершена, ресурсы освобождены. |

### 7.2 Конфигурация

| Параметр | Значение по умолчанию | Описание |
|----------|----------------------|----------|
| HandshakeTimeout | 30s | Максимальное время на хэндшейк. |
| SessionTTL | 24h | Время жизни активной сессии. |
| SuspendTTL | 5m | Время ожидания реконнекта в SUSPENDED. |
| MaxConnections | 5 | Максимум соединений на сессию. |

### 7.3 Хранение состояния

Реализации MUST поддерживать персистентное хранение:

- **Redis:** Горячее состояние (активные сессии, быстрый поиск).
- **PostgreSQL:** Холодное хранение (история, аудит, восстановление).

### 7.4 Мультиплексирование

Одна сессия MAY иметь несколько транспортных соединений. Реализация MUST:

1. Назначать каждому соединению уникальный `connection_id`.
2. Выбирать оптимальное соединение для отправки (по приоритету транспорта).
3. Обеспечивать graceful fallback при потере соединения.

---

## 8. Прикладной уровень

### 8.1 AppMessage

Все прикладные сообщения сериализуются в `AppMessage`:

```protobuf
message AppMessage {
    bytes                     message_id = 1;  // UUIDv7 (16 bytes)
    google.protobuf.Timestamp timestamp  = 2;
    oneof body {
        ChatMessage            chat          = 3;
        Command                cmd           = 4;
        MediaChunk             media         = 5;
        Ack                    ack           = 6;
        Ping                   ping          = 7;
        Pong                   pong          = 8;
        SenderKeyDistribution  sender_key    = 9;
        SyncRequest            sync_request  = 10;
        SignalingMessage       signaling     = 11;
    }
}
```

### 8.2 ChatMessage

```protobuf
message ChatMessage {
    bytes     conversation_id = 1;
    bytes     sender_id       = 2;
    bytes     recipient_id    = 3;  // 1-to-1
    bytes     group_id        = 4;  // Групповые
    ChatAction action         = 5;
    string    text            = 6;
    bytes     reply_to        = 7;
    repeated Attachment attachments = 8;
    
    enum ChatAction {
        CHAT_ACTION_UNSPECIFIED = 0;
        SEND     = 1;
        EDIT     = 2;
        DELETE   = 3;
        REACTION = 4;
    }
}
```

### 8.3 Command

```protobuf
message Command {
    CommandType type    = 1;
    bytes       payload = 2;
    
    enum CommandType {
        COMMAND_TYPE_UNSPECIFIED = 0;
        CREATE_GROUP  = 1;
        ADD_MEMBER    = 2;
        REMOVE_MEMBER = 3;
        UPDATE_GROUP  = 4;
    }
}
```

### 8.4 MediaChunk

```protobuf
message MediaChunk {
    bytes  upload_id   = 1;  // UUID загрузки
    uint32 chunk_index = 2;  // Индекс чанка
    uint32 total       = 3;  // Всего чанков
    bytes  data        = 4;  // Данные чанка
    string mime_type   = 5;  // MIME-тип файла
}
```

### 8.5 Ack, Ping, Pong

```protobuf
message Ack {
    bytes     ref_message_id = 1;
    AckStatus status         = 2;
    
    enum AckStatus {
        ACK_STATUS_UNSPECIFIED = 0;
        RECEIVED  = 1;
        DELIVERED = 2;
        READ      = 3;
        ERROR     = 4;
    }
}

message Ping {
    int64 timestamp_ms = 1;
}

message Pong {
    int64 ping_timestamp_ms = 1;
    int64 pong_timestamp_ms = 2;
}
```

### 8.6 SyncRequest

```protobuf
message SyncRequest {
    bytes  last_message_id  = 1;
    uint64 last_seq         = 2;
    uint32 max_messages     = 3;
}
```

### 8.7 SignalingMessage

```protobuf
message SignalingMessage {
    string call_id    = 1;
    bytes  sender_id  = 2;
    bytes  target_id  = 3;
    oneof payload {
        string sdp_offer      = 4;
        string sdp_answer     = 5;
        string ice_candidate  = 6;
    }
    StreamControl stream_control = 7;
    
    enum StreamControl {
        STREAM_CONTROL_UNSPECIFIED = 0;
        MUTE_AUDIO   = 1;
        UNMUTE_AUDIO = 2;
        MUTE_VIDEO   = 3;
        UNMUTE_VIDEO = 4;
        END_CALL     = 5;
    }
}
```

---

## 9. Групповое шифрование (Sender Keys)

### 9.1 Обзор

Групповое E2E-шифрование основано на протоколе Signal Sender Keys:

1. Каждый участник группы генерирует пару Sender Key (Ed25519 signing key + 32-byte chain key).
2. Sender Key распространяется всем участникам группы через зашифрованные 1-to-1 каналы.
3. Сообщения шифруются производным ключом и подписываются Ed25519.

### 9.2 Генерация ключей

```
signing_key     = Ed25519.GenerateKey()       // 64 bytes private, 32 bytes public
chain_key       = CSPRNG(32)                   // 32 random bytes
sender_key      = (signing_key, chain_key)
```

### 9.3 Ratchet (цепная деривация)

Каждое сообщение использует производный message key:

```
message_key = HMAC-SHA256(chain_key, 0x01)[0:32]
next_chain  = HMAC-SHA256(chain_key, 0x02)[0:32]
chain_key   = next_chain
```

### 9.4 Шифрование сообщения

```
plaintext   = original_message
message_key = derive_from_chain()
ciphertext  = ChaCha20-Poly1305.Encrypt(key=message_key, nonce=random(12), ad=nil, plaintext)
signature   = Ed25519.Sign(signing_private_key, ciphertext)
output      = signature || ciphertext
```

### 9.5 Расшифровка сообщения

```
signature  = output[0:64]
ciphertext = output[64:]
valid      = Ed25519.Verify(sender_public_key, ciphertext, signature)
if !valid: REJECT
message_key = derive_from_sender_chain()
plaintext   = ChaCha20-Poly1305.Decrypt(key=message_key, ciphertext)
```

### 9.6 Распространение ключей

```protobuf
message SenderKeyDistribution {
    bytes group_id       = 1;
    bytes sender_id      = 2;
    bytes chain_key      = 3;  // 32 bytes
    bytes signing_key    = 4;  // Ed25519 public key, 32 bytes
    uint32 iteration     = 5;  // Текущий номер итерации
}
```

Реализации MUST:
- Распространять новый Sender Key при добавлении участника.
- Выполнять полную ротацию (full rotation) при удалении участника.

### 9.7 Ротация

| Событие | Действие |
|---------|---------|
| Новый участник | Текущие участники отправляют свои Sender Keys новому члену. |
| Удаление участника | Все оставшиеся участники генерируют новые Sender Keys и распространяют их. |
| Периодическая | SHOULD выполнять ротацию каждые 2^16 сообщений. |

### 9.8 Кэширование

Sender Keys SHOULD кэшироваться в Redis с TTL:

```
Key:   senderkey:{group_id}:{sender_id}
Value: JSON(SenderKey)
TTL:   24h
```

---

## 10. Механизм подтверждений

### 10.1 Потоковая модель

```
Sender                              Receiver
  |                                    |
  |---- Frame(seq=N, AppMessage) ----->|
  |                                    |
  |<--- Frame(ack=N, Ack{RECEIVED}) ---|
  |                                    |
  |<--- Frame(ack=N, Ack{DELIVERED}) --|  (отображено пользователю)
  |                                    |
  |<--- Frame(ack=N, Ack{READ}) ------|  (прочитано пользователем)
```

### 10.2 Уровни подтверждения

| Уровень | Описание | Обязательность |
|---------|----------|----------------|
| RECEIVED | Сообщение получено сервером. | MUST |
| DELIVERED | Сообщение доставлено получателю. | SHOULD |
| READ | Сообщение прочитано получателем. | MAY |

### 10.3 Дедупликация

Реализации MUST обеспечивать идемпотентную обработку через `message_id`:

- Redis SETNX с TTL для проверки дубликатов.
- PostgreSQL INSERT ... ON CONFLICT DO NOTHING для Inbox.

---

## 11. Оффлайн-синхронизация

### 11.1 Механизм

1. Клиент отправляет `SyncRequest` с `last_message_id` и `last_seq`.
2. Сервер выбирает пропущенные сообщения из Inbox.
3. Сообщения отправляются пакетами (страницами).

### 11.2 Параметры

| Параметр | Значение по умолчанию | Описание |
|----------|----------------------|----------|
| max_messages | 100 | Максимум сообщений в одном ответе. |
| page_size | 50 | Размер страницы при пагинации. |

### 11.3 Outbox/Inbox паттерн

```
Sender → Outbox (PostgreSQL) → Relay → Kafka → Consumer → Inbox (PostgreSQL) → Receiver
```

Outbox MUST гарантировать at-least-once доставку в Kafka через polling relay.

---

## 12. WebRTC-сигнализация

### 12.1 Поток

WebRTC-сигнализация передаётся как `SignalingMessage` внутри зашифрованного канала Proto:

```
Caller                    Server                    Callee
  |                         |                         |
  |-- SignalingMessage ----->|                         |
  |   (sdp_offer)           |-- SignalingMessage ----->|
  |                         |   (sdp_offer)           |
  |                         |                         |
  |                         |<-- SignalingMessage -----|
  |<-- SignalingMessage -----|   (sdp_answer)          |
  |   (sdp_answer)          |                         |
  |                         |                         |
  |-- ice_candidate ------->|-- ice_candidate -------->|
  |<-- ice_candidate -------|<-- ice_candidate --------|
  |                         |                         |
  |========== Direct P2P Media Stream ================|
```

### 12.2 Управление потоком

`StreamControl` позволяет управлять медиа-потоками без пересогласования SDP:

- `MUTE_AUDIO` / `UNMUTE_AUDIO`
- `MUTE_VIDEO` / `UNMUTE_VIDEO`
- `END_CALL`

---

## 13. Коды ошибок

| Код | Имя | Описание |
|-----|-----|----------|
| 0 | OK | Успех. |
| 1 | HANDSHAKE_FAILED | Ошибка при Noise-хэндшейке. |
| 2 | DECRYPT_FAILED | Ошибка расшифровки (повреждённые данные или неверный ключ). |
| 3 | SESSION_EXPIRED | Сессия истекла. |
| 4 | SESSION_NOT_FOUND | Сессия не найдена. |
| 5 | INVALID_FRAME | Некорректный формат фрейма. |
| 6 | SEQ_OUT_OF_ORDER | Нарушение порядка seq (потеря или дубликат). |
| 7 | PAYLOAD_TOO_LARGE | Полезная нагрузка превышает максимальный размер. |
| 8 | SIGNATURE_INVALID | Неверная Ed25519-подпись (группы). |
| 9 | SENDER_KEY_MISSING | Отсутствует Sender Key для расшифровки группового сообщения. |
| 10 | RATE_LIMITED | Превышен лимит запросов. |
| 11 | INTERNAL_ERROR | Внутренняя ошибка сервера. |

---

## 14. Требования безопасности

### 14.1 Обязательные

1. Реализации MUST использовать TLS 1.3 или выше для WebSocket-транспорта.
2. Реализации MUST обнулять ключевой материал после использования.
3. Реализации MUST отклонять хэндшейк-сообщения с некорректной длиной.
4. Реализации MUST использовать CSPRNG для генерации всех ключей и nonce.
5. Реализации MUST NOT передавать открытый текст после начала хэндшейка.
6. Реализации MUST проверять Ed25519-подписи для групповых сообщений.
7. Реализации MUST NOT повторно использовать nonce.

### 14.2 Рекомендуемые

1. Реализации SHOULD ротировать ключи каждые 2^32 сообщений.
2. Реализации SHOULD использовать UUIDv7 для `message_id` (сортируемость по времени).
3. Реализации SHOULD логировать аномалии (неудачные хэндшейки, повторные nonce) для аудита.
4. Реализации SHOULD использовать rate limiting на транспортном уровне.
5. Реализации SHOULD хранить Sender Keys в защищённом хранилище с ограниченным TTL.

---

## 15. Требования к реализациям

### 15.1 Conformance Levels

| Уровень | Описание |
|---------|----------|
| **Minimal** | Transport + Noise handshake + Frame encrypt/decrypt. |
| **Standard** | Minimal + Session management + AppMessage types + Ack. |
| **Full** | Standard + Group encryption + Offline sync + WebRTC signaling. |

### 15.2 Обязательные тесты

Реализации, заявляющие соответствие, MUST проходить conformance test suite:

1. **Handshake:** Успешный 3-way Noise XX handshake.
2. **Encrypt/Decrypt:** Round-trip шифрования/расшифровки.
3. **Rekey:** Ротация ключей с сохранением связи.
4. **Seq ordering:** Монотонность seq.
5. **Frame encoding:** Корректная Protobuf-сериализация Frame.
6. **Session state machine:** Переходы CREATED → ACTIVE → SUSPENDED → ACTIVE → EXPIRED.

### 15.3 Векторы тестирования

Референсные реализации (Go) находятся в:

- `proto-core/crypto/` — криптографические тесты
- `proto-core/transport/` — транспортные тесты
- `proto-core/session/` — сессионные тесты

---

## 16. IANA-подобные реестры

### 16.1 Типы транспорта

| Значение | Имя | Спецификация |
|----------|-----|--------------|
| 0 | websocket | Секция 4.2 |
| 1 | quic | Секция 4.3 |
| 2 | grpc | Секция 4.4 |

### 16.2 Типы хэндшейк-сообщений

| Значение | Имя | Описание |
|----------|-----|----------|
| 0 | UNSPECIFIED | Не определён. |
| 1 | INIT | Инициализация (-> e). |
| 2 | RESPONSE | Ответ (-> e, ee, s, es). |
| 3 | FINISH | Завершение (-> s, se). |

### 16.3 Типы прикладных сообщений

| Field number | Тип | Описание |
|-------------|-----|----------|
| 3 | ChatMessage | Текстовое сообщение. |
| 4 | Command | Административная команда. |
| 5 | MediaChunk | Фрагмент медиа-файла. |
| 6 | Ack | Подтверждение. |
| 7 | Ping | Heartbeat запрос. |
| 8 | Pong | Heartbeat ответ. |
| 9 | SenderKeyDistribution | Распределение группового ключа. |
| 10 | SyncRequest | Запрос синхронизации. |
| 11 | SignalingMessage | WebRTC-сигнализация. |

### 16.4 Статусы подтверждения (Ack)

| Значение | Имя | Описание |
|----------|-----|----------|
| 0 | UNSPECIFIED | Не определён. |
| 1 | RECEIVED | Получено сервером. |
| 2 | DELIVERED | Доставлено клиенту. |
| 3 | READ | Прочитано пользователем. |
| 4 | ERROR | Ошибка обработки. |

### 16.5 Типы команд

| Значение | Имя | Описание |
|----------|-----|----------|
| 0 | UNSPECIFIED | Не определён. |
| 1 | CREATE_GROUP | Создание группы. |
| 2 | ADD_MEMBER | Добавление участника. |
| 3 | REMOVE_MEMBER | Удаление участника. |
| 4 | UPDATE_GROUP | Обновление метаданных группы. |

---

## Приложение A: Kafka-топики

| Топик | Назначение | Ключ партиции |
|-------|-----------|---------------|
| `user-{id}` | Персональные сообщения | message_id |
| `group-{id}` | Групповые сообщения | message_id |
| `commands` | Административные команды | command_type |
| `events` | Системные события | event_type |
| `media-events` | События медиа-загрузки | upload_id |
| `dead-letter` | Необработанные сообщения | original_topic |

## Приложение B: Версионирование

Протокол Proto следует [Semantic Versioning 2.0.0](https://semver.org):

- **Major:** Несовместимые изменения в фрейминге, криптографии или хэндшейке.
- **Minor:** Новые типы AppMessage, новые транспорты, расширения.
- **Patch:** Уточнения формулировок, исправления опечаток.

Текущая версия: **1.0.0**

## Приложение C: Ссылки

1. [Noise Protocol Framework, rev 34](https://noiseprotocol.org/noise.html)
2. [RFC 7748 - Elliptic Curves for Security (X25519)](https://www.rfc-editor.org/rfc/rfc7748)
3. [RFC 8439 - ChaCha20 and Poly1305](https://www.rfc-editor.org/rfc/rfc8439)
4. [RFC 7693 - BLAKE2](https://www.rfc-editor.org/rfc/rfc7693)
5. [RFC 6455 - WebSocket Protocol](https://www.rfc-editor.org/rfc/rfc6455)
6. [RFC 9000 - QUIC Transport Protocol](https://www.rfc-editor.org/rfc/rfc9000)
7. [RFC 2119 - Key Words for RFCs](https://www.rfc-editor.org/rfc/rfc2119)
8. [Protocol Buffers v3](https://protobuf.dev)
9. [Signal Sender Keys](https://signal.org/docs/specifications/group-v2/)
10. [Semantic Versioning 2.0.0](https://semver.org)
