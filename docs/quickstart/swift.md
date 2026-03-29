# Swift SDK Quickstart (iOS)

## Установка

Добавьте в `Package.swift`:

```swift
dependencies: [
    .package(url: "https://github.com/qmish/FProto.git", from: "0.5.0")
]
```

## Быстрый старт

```swift
import FProtoSDK

// 1. Создание клиента
let client = FProtoClient(url: "ws://localhost:8080/ws")

// 2. Подключение
try await client.connect()

// 3. Отправка сообщения
try await client.send(Data("Привет, FProto!".utf8))

// 4. Обработка входящих
client.onMessage { data in
    print("Получено: \(String(data: data, encoding: .utf8) ?? "")")
}

// 5. Отключение
client.disconnect()
```

## Reconnect

```swift
let client = FProtoClient(
    url: "ws://localhost:8080/ws",
    reconnect: true,
    maxRetries: 10,
    baseDelay: 1.0
)
```

## Запуск тестов

```bash
cd sdk/swift
swift test
```
