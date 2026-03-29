# Proto Protocol -- Data Flows

**Версия:** 1.0
**Статус:** Draft

**Связанные документы:** [Roadmap](Roadmap.md) | [HLD](HLD.md) | [LLD](LLD.md) | [Network Architecture](Network-Architecture.md) | [Network Connectivity](Network-Connectivity.md) | [Infrastructure](Infrastructure.md)

---

## 1. Аутентификация и создание сессии

### 1.1 Первичное подключение (Noise Handshake + Session Creation)

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Transport Gateway
    participant SM as Session Manager
    participant CH as Crypto Handler
    participant R as Redis
    participant PG as PostgreSQL

    C->>GW: Transport connect (WS upgrade / QUIC / gRPC stream)
    GW->>SM: RegisterConnection(transport_type, remote_addr)
    SM->>R: CREATE session:{temp_id} (status=connecting)
    SM-->>GW: temp_session_id

    Note over C,CH: Noise XX Handshake (3 messages)

    C->>GW: HandshakeMessage(INIT, e_client_pub)
    GW->>CH: ProcessHandshake(INIT, e_client_pub)
    CH-->>GW: HandshakeMessage(RESPONSE, e_server_pub + Enc(s_server_pub))
    GW-->>C: HandshakeMessage(RESPONSE)

    C->>GW: HandshakeMessage(FINISH, Enc(s_client_pub) + auth_token)
    GW->>CH: ProcessHandshake(FINISH, payload)
    CH->>SM: ValidateAuthToken(token)
    SM->>PG: SELECT user by token
    PG-->>SM: user_id, device_id

    SM->>R: UPDATE session:{session_id} SET status=active, user_id, device_id, crypto_state
    SM->>R: SADD user:{user_id}:sessions session_id
    SM->>PG: INSERT INTO sessions (backup)
    SM-->>CH: session_id, user_id
    CH-->>GW: HandshakeComplete(session_id, send_key, recv_key)
    GW-->>C: Frame(session_id, seq=0, Enc(SessionEstablished))

    Note over C,GW: Session active -- encrypted communication
```

### 1.2 Reconnect (Session Resumption)

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Transport Gateway
    participant SM as Session Manager
    participant R as Redis

    C->>GW: Transport connect
    C->>GW: Frame(session_id, seq=last_sent+1, ack=last_received)
    GW->>SM: GetSession(session_id)
    SM->>R: HGETALL session:{session_id}
    R-->>SM: session state (crypto_state, status)

    alt Session exists and not expired
        SM->>R: HSET session:{session_id} status=active, last_active=now
        SM->>R: HSET session:{session_id}:connections {conn_id}={...}
        SM-->>GW: Session restored, crypto_state
        GW-->>C: Frame(session_id, seq=server_last+1, ack=client_seq)
        Note over C,GW: Session resumed -- no re-handshake needed
    else Session expired or not found
        GW-->>C: Error(SESSION_EXPIRED)
        Note over C: Client must perform full handshake
    end
```

---

## 2. Отправка личного сообщения

### 2.1 Полный путь сообщения (Sender -> Recipient)

```mermaid
sequenceDiagram
    participant Sender as Sender Client
    participant GW1 as Gateway (Sender)
    participant CH as Crypto Handler
    participant D as Dispatcher
    participant PG as PostgreSQL
    participant K as Kafka
    participant SS as SyncService
    participant SM as Session Manager
    participant GW2 as Gateway (Recipient)
    participant Recipient as Recipient Client

    Sender->>GW1: Frame(encrypted AppMessage: ChatMessage)
    GW1->>CH: Decrypt(session_id, encrypted_payload)
    CH-->>GW1: AppMessage(ChatMessage: text="Hello", recipient_id=Bob)

    GW1->>D: Dispatch(sender_id=Alice, AppMessage)

    Note over D,PG: Outbox Pattern
    D->>PG: INSERT INTO outbox (message_id, topic="user-Bob", payload)
    PG-->>D: OK
    D-->>GW1: Ack(message_id, status=RECEIVED)
    GW1->>CH: Encrypt(Ack)
    CH-->>GW1: encrypted Frame
    GW1-->>Sender: Frame(Ack: RECEIVED)

    Note over D,K: Async Relay
    D->>K: Produce("user-Bob", key=Bob_id, payload)
    D->>PG: UPDATE outbox SET status=published

    K->>SS: Consume("user-Bob", message)
    SS->>PG: INSERT INTO inbox (user_id=Bob, message_id, payload)

    SS->>SM: GetActiveSessions(user_id=Bob)
    SM-->>SS: [session_id_1, session_id_2]

    alt Recipient online
        SS->>SM: PushToUser(Bob, AppMessage)
        SM->>GW2: RouteToSession(session_id, encrypted Frame)
        GW2-->>Recipient: Frame(encrypted AppMessage: ChatMessage)
        Recipient-->>GW2: Frame(Ack: DELIVERED)
        GW2->>SS: AckDelivered(message_id)
        SS->>PG: UPDATE inbox SET delivered=true
    else Recipient offline
        Note over SS: Message stored in Inbox, awaiting sync
        SS->>K: Produce("notifications", push notification payload)
    end
```

### 2.2 Ack Flow (подтверждение доставки)

```mermaid
sequenceDiagram
    participant S as Sender
    participant Server as Proto Server
    participant R as Recipient

    S->>Server: ChatMessage(id=msg1)
    Server-->>S: Ack(msg1, RECEIVED)
    Note over Server: Сообщение принято сервером

    Server->>R: ChatMessage(id=msg1)
    R-->>Server: Ack(msg1, DELIVERED)
    Server-->>S: Ack(msg1, DELIVERED)
    Note over S: Сообщение доставлено

    Note over R: Пользователь открыл чат
    R-->>Server: Ack(msg1, READ)
    Server-->>S: Ack(msg1, READ)
    Note over S: Сообщение прочитано
```

---

## 3. Групповое сообщение с Sender Keys

```mermaid
sequenceDiagram
    participant Alice as Alice (Sender)
    participant GW as Gateway
    participant D as Dispatcher
    participant K as Kafka
    participant SS as SyncService
    participant Bob as Bob (Member)
    participant Carol as Carol (Member)

    Note over Alice: Шифрует сообщение своим sender_key для группы
    Alice->>GW: Frame(Enc(AppMessage: ChatMessage, group_id=G1))
    GW->>D: Dispatch(sender=Alice, group_id=G1, encrypted_body)

    D->>K: Produce("group-G1", key=G1, payload)

    K->>SS: Consume("group-G1")

    Note over SS: Определяет участников группы
    SS->>SS: GetGroupMembers(G1) -> [Alice, Bob, Carol]

    par Deliver to Bob
        SS->>Bob: Push(AppMessage with sender_key_encrypted body)
        Note over Bob: Расшифровывает sender_key Alice
    and Deliver to Carol
        SS->>Carol: Push(AppMessage with sender_key_encrypted body)
        Note over Carol: Расшифровывает sender_key Alice
    end
```

### 3.1 Распространение Sender Key при вступлении

```mermaid
sequenceDiagram
    participant Admin as Admin
    participant Server as Server
    participant New as New Member (Dave)
    participant Bob as Bob (Existing)
    participant Carol as Carol (Existing)

    Admin->>Server: Command(ADD_MEMBER, group=G1, user=Dave)
    Server->>Server: Process command

    par Existing members send their keys to Dave
        Bob->>Dave: SenderKeyDistribution(group=G1, sender=Bob, key)
        Carol->>Dave: SenderKeyDistribution(group=G1, sender=Carol, key)
        Admin->>Dave: SenderKeyDistribution(group=G1, sender=Admin, key)
    end

    Dave->>Bob: SenderKeyDistribution(group=G1, sender=Dave, key)
    Dave->>Carol: SenderKeyDistribution(group=G1, sender=Dave, key)
    Dave->>Admin: SenderKeyDistribution(group=G1, sender=Dave, key)

    Note over Bob,Carol: Dave теперь может отправлять/получать в группе
```

---

## 4. Офлайн-синхронизация

```mermaid
sequenceDiagram
    participant C as Client (was offline 2h)
    participant GW as Gateway
    participant SM as Session Manager
    participant SS as SyncService
    participant K as Kafka
    participant PG as PostgreSQL

    C->>GW: Connect + Noise Handshake (or Resume)
    C->>GW: SyncRequest(last_seen_message_id=msg_42, max_messages=100)

    GW->>SM: GetSession / CreateSession
    GW->>SS: SyncRequest(user_id, last_seen=msg_42, limit=100)

    SS->>PG: SELECT FROM inbox WHERE user_id=? AND message_id > msg_42 ORDER BY message_id LIMIT 100

    PG-->>SS: [msg_43, msg_44, ..., msg_87] (45 messages)

    loop For each missed message
        SS->>GW: Push(AppMessage)
        GW-->>C: Frame(encrypted AppMessage)
    end

    SS-->>GW: SyncComplete(last_message_id=msg_87, has_more=false)
    GW-->>C: Frame(SyncComplete)

    C-->>GW: Ack(batch: msg_43..msg_87)
    Note over C: Client is now up to date
```

### 4.1 Пагинация при длительном офлайне

```mermaid
sequenceDiagram
    participant C as Client
    participant Server as Server

    C->>Server: SyncRequest(last_seen=msg_100, max=100)
    Server-->>C: [msg_101..msg_200], has_more=true
    C->>Server: Ack(msg_101..msg_200)

    C->>Server: SyncRequest(last_seen=msg_200, max=100)
    Server-->>C: [msg_201..msg_250], has_more=false
    C->>Server: Ack(msg_201..msg_250)

    Note over C: Синхронизация завершена (150 сообщений)
```

---

## 5. Медиа-загрузка

### 5.1 Chunked Upload Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway (media stream)
    participant MA as Media Aggregator
    participant S3 as S3 / MinIO
    participant D as Dispatcher
    participant K as Kafka

    Note over C: Файл: photo.jpg, 256KB -> 4 chunks x 64KB

    C->>GW: MediaChunk(stream_id=S1, idx=0, total=4, mime="image/jpeg", size=262144, data, hash)
    GW->>MA: StoreChunk(S1, 0, data, hash)
    MA->>MA: Validate hash, store to temp

    C->>GW: MediaChunk(stream_id=S1, idx=1, data, hash)
    GW->>MA: StoreChunk(S1, 1, data, hash)

    C->>GW: MediaChunk(stream_id=S1, idx=2, data, hash)
    GW->>MA: StoreChunk(S1, 2, data, hash)

    C->>GW: MediaChunk(stream_id=S1, idx=3, data, hash)
    GW->>MA: StoreChunk(S1, 3, data, hash)

    Note over MA: All chunks received, assembling file

    MA->>MA: Assemble file, verify total hash
    MA->>S3: PutObject(key="media/S1/photo.jpg", body=assembled_file)
    S3-->>MA: OK, url=https://cdn.example.com/media/S1/photo.jpg

    MA->>D: Dispatch(MediaComplete: stream_id=S1, url, mime, size)
    D->>K: Produce(topic, MediaMetaMessage)

    MA-->>GW: MediaUploadComplete(stream_id=S1, url)
    GW-->>C: Ack(stream_id=S1, url=...)

    Note over C: Клиент может отправить ChatMessage с Attachment(url)
```

### 5.2 Resume Upload (при потере соединения)

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant MA as Media Aggregator

    Note over C: Отправлено 2 из 4 chunks, соединение потеряно

    C->>GW: Reconnect
    C->>GW: MediaChunk(stream_id=S1, idx=0, total=4, ...)
    GW->>MA: GetUploadStatus(S1)
    MA-->>GW: Received chunks: [0, 1]
    GW-->>C: Ack(S1, received_chunks=[0, 1])

    Note over C: Клиент пропускает chunks 0,1 и продолжает с 2

    C->>GW: MediaChunk(stream_id=S1, idx=2, data, hash)
    C->>GW: MediaChunk(stream_id=S1, idx=3, data, hash)

    Note over MA: Assembly continues...
```

---

## 6. Ротация ключей

### 6.1 Scheduled Rekey

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant SM as Session Manager
    participant R as Redis

    Note over C: Порог: 10000 сообщений или 4 часа

    C->>GW: Frame(seq=10001, AppMessage: RekeyRequest)
    GW->>SM: InitiateRekey(session_id)

    SM->>SM: new_key = HKDF(current_key, "proto-rekey", 32)
    SM->>R: HSET session:{id} crypto_state=new_state
    SM-->>GW: RekeyAccepted(new_nonce_base=0)
    GW-->>C: Frame(seq=N, AppMessage: RekeyAccepted)

    Note over C: Вычисляет new_key = HKDF(current_key, "proto-rekey", 32)
    Note over C: Сбрасывает nonce counter = 0
    Note over C: Zeroize old key

    C->>GW: Frame(seq=0, new encryption)
    Note over C,GW: Дальнейшие пакеты шифруются новым ключом
```

### 6.2 Sender Key Rotation (Group)

```mermaid
sequenceDiagram
    participant Admin as Admin
    participant Server as Server
    participant Alice as Alice
    participant Bob as Bob
    participant Removed as Charlie (removed)

    Admin->>Server: Command(REMOVE_MEMBER, group=G1, user=Charlie)
    Server->>Server: Remove Charlie from group

    Note over Server: Trigger sender key rotation for all remaining members

    par Each member generates new sender key
        Alice->>Alice: Generate new sender_key for G1
        Alice->>Bob: SenderKeyDistribution(new key)
        Alice->>Admin: SenderKeyDistribution(new key)
    and
        Bob->>Bob: Generate new sender_key for G1
        Bob->>Alice: SenderKeyDistribution(new key)
        Bob->>Admin: SenderKeyDistribution(new key)
    and
        Admin->>Admin: Generate new sender_key for G1
        Admin->>Alice: SenderKeyDistribution(new key)
        Admin->>Bob: SenderKeyDistribution(new key)
    end

    Note over Removed: Charlie не получает новые ключи
    Note over Alice,Bob: Forward secrecy обеспечена
```

---

## 7. Ping/Pong Keepalive

```mermaid
sequenceDiagram
    participant C as Client
    participant GW as Gateway
    participant SM as Session Manager
    participant R as Redis

    loop Every 30s
        C->>GW: Frame(Ping: timestamp_ms=1700000000000)
        GW->>SM: UpdateLastActive(session_id)
        SM->>R: HSET session:{id} last_active=now
        GW-->>C: Frame(Pong: echo_ts=1700000000000, server_ts=1700000000050)
        Note over C: RTT = server_ts - echo_ts (approx)
    end

    Note over GW: Если Ping не получен за 90s:
    GW->>SM: UpdateStatus(session_id, sleeping)
    SM->>R: HSET session:{id} status=sleeping
```

---

## 8. Outbox Relay (асинхронная публикация в Kafka)

```mermaid
sequenceDiagram
    participant Relay as Outbox Relay Worker
    participant PG as PostgreSQL
    participant K as Kafka

    loop Every 100ms (or on notification)
        Relay->>PG: SELECT * FROM outbox WHERE status='pending' ORDER BY created_at LIMIT 100 FOR UPDATE SKIP LOCKED
        PG-->>Relay: [msg_1, msg_2, ..., msg_N]

        loop For each message
            Relay->>K: Produce(topic, key, payload)
            alt Kafka success
                Relay->>PG: UPDATE outbox SET status='published', published_at=now WHERE id=?
            else Kafka failure
                Relay->>PG: UPDATE outbox SET retry_count=retry_count+1 WHERE id=?
                Note over Relay: Exponential backoff on next attempt
            end
        end
    end

    Note over Relay: Messages with retry_count > 10 -> dead_letter topic
```

---

## 9. Сводная карта потоков данных

```mermaid
graph LR
    subgraph clientSide[Client Side]
        App["Application"]
        Crypto["Crypto (Noise)"]
        Transport["Transport"]
    end

    subgraph serverSide[Server Side]
        GW["Gateway"]
        SM["Session Mgr"]
        CH["Crypto Handler"]
        DISP["Dispatcher"]
    end

    subgraph messaging[Messaging]
        Outbox["Outbox (PG)"]
        KafkaNode["Kafka"]
        Sync["SyncService"]
        Notif["NotifService"]
    end

    subgraph storage[Storage]
        Redis["Redis"]
        PG["PostgreSQL"]
        ObjectStore["S3"]
    end

    App -->|"1. AppMessage"| Crypto
    Crypto -->|"2. Encrypted Frame"| Transport
    Transport -->|"3. Network"| GW
    GW -->|"4. Decrypt"| CH
    CH -->|"5. Route"| DISP
    DISP -->|"6. Write"| Outbox
    Outbox -->|"7. Relay"| KafkaNode
    KafkaNode -->|"8a. Consume"| Sync
    KafkaNode -->|"8b. Consume"| Notif
    Sync -->|"9. Store"| PG
    Sync -->|"10. Push"| SM
    SM -->|"11. Route"| GW
    SM --> Redis
    DISP --> PG
    GW -->|"12. Deliver"| Transport
```
