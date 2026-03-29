# TypeScript SDK Quickstart

## Установка

```bash
npm install @fproto/sdk
```

## Быстрый старт

```typescript
import { FProtoClient } from '@fproto/sdk';

async function main() {
  // 1. Создание клиента
  const client = new FProtoClient('ws://localhost:8080/ws');

  // 2. Подключение
  await client.connect();

  // 3. Отправка сообщения
  client.send(new TextEncoder().encode('Привет, FProto!'));

  // 4. Обработка входящих сообщений
  client.onMessage((data: Uint8Array) => {
    console.log('Получено:', new TextDecoder().decode(data));
  });

  // 5. Отключение
  client.close();
}

main();
```

## Reconnect

```typescript
const client = new FProtoClient('ws://localhost:8080/ws', {
  reconnect: true,
  maxRetries: 10,
  baseDelay: 1000,
});
```

## Запуск тестов

```bash
cd sdk/typescript
npm install
npm test
```
