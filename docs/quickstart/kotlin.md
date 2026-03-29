# Kotlin SDK Quickstart (Android)

## Установка

Добавьте в `build.gradle.kts`:

```kotlin
dependencies {
    implementation("com.fproto:sdk:0.5.0")
}
```

## Быстрый старт

```kotlin
import com.fproto.sdk.FProtoClient

fun main() {
    // 1. Создание клиента
    val client = FProtoClient("ws://localhost:8080/ws")

    // 2. Подключение
    client.connect()

    // 3. Обработка сообщений
    client.onMessage { data ->
        println("Получено: ${String(data)}")
    }

    // 4. Отправка
    client.send("Привет, FProto!".toByteArray())

    // 5. Отключение
    client.close()
}
```

## Reconnect

```kotlin
val client = FProtoClient(
    url = "ws://localhost:8080/ws",
    reconnect = true,
    maxRetries = 10
)
```

## Запуск тестов

```bash
cd sdk/kotlin
./gradlew test
```
