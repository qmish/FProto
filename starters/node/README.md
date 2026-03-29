# FProto Node.js/TypeScript Starter Pack

TypeScript SDK для протокола FProto — пакет `@fproto/core`.

## Установка

```bash
npm install
```

## Сборка

```bash
npm run build
```

## Тесты

```bash
npm test
```

## Echo-пример

```bash
# Сервер
npx ts-node examples/echo.ts server 127.0.0.1:9400

# Клиент
npx ts-node examples/echo.ts client 127.0.0.1:9400
```

## Структура

```
src/
  index.ts        # Публичный API
  transport.ts    # WebSocket Conn (ws)
  crypto.ts       # AEAD (Node crypto), Rekey, Zeroize
  session.ts      # Конечный автомат состояний
examples/
  echo.ts         # Echo server + client
tests/
  conformance.test.ts  # Vitest conformance тесты
```

## Зависимости

- `ws` — WebSocket для Node.js
- Node.js `crypto` — ChaCha20-Poly1305 (встроен с Node 18+)
- TypeScript — типизация
