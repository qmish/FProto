# Go SDK Quickstart

## Установка

```bash
go get github.com/qmish/FProto/sdk/go
```

## Быстрый старт

```go
package main

import (
    "context"
    "fmt"
    "log"

    fproto "github.com/qmish/FProto/sdk/go"
)

func main() {
    // 1. Создание клиента
    client := fproto.NewClient("ws://localhost:8080/ws")

    // 2. Подключение (Noise XX handshake)
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // 3. Отправка сообщения
    if err := client.Send([]byte("Привет, FProto!")); err != nil {
        log.Fatal(err)
    }

    // 4. Получение ответа
    resp, err := client.Receive()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Ответ: %s\n", resp)
}
```

## Reconnect с экспоненциальным backoff

```go
client := fproto.NewClient("ws://localhost:8080/ws",
    fproto.WithReconnect(true),
    fproto.WithMaxRetries(10),
)
```

## QUIC транспорт

```go
client := fproto.NewClient("quic://localhost:8443")
```

## Запуск тестов

```bash
cd sdk/go
go test -v ./...
```
