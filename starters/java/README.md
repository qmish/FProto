# FProto Java Starter Pack

Java SDK для протокола FProto — артефакт `io.fproto:fproto-core`.

## Требования

- Java 17+
- Maven 3.8+

## Сборка

```bash
mvn compile
```

## Тесты

```bash
mvn test
```

## Структура

```
src/main/java/io/fproto/core/
  Conn.java                    # Интерфейс транспортного соединения
  TransportType.java           # Типы транспорта
  CryptoUtils.java             # AEAD, Rekey, Zeroize, nonce/AAD
  SessionStatus.java           # Состояния сессии
  SessionEvent.java            # События сессии
  SessionMachine.java          # Конечный автомат
  SessionConfig.java           # Конфигурация сессии
  TransitionResult.java        # Результат перехода
  InvalidTransitionException.java

src/test/java/io/fproto/core/
  ConformanceTest.java         # JUnit 5 conformance тесты
```

## Зависимости

- JCA `ChaCha20-Poly1305` (Java 11+)
- `Java-WebSocket` — WebSocket транспорт
- JUnit 5 — тесты
