# FProto .NET Starter Pack

.NET 8 SDK для протокола FProto — пакет `FProto.Core`.

## Требования

- .NET 8 SDK

## Сборка

```bash
dotnet build
```

## Тесты

```bash
dotnet test
```

## Структура

```
FProto.Core/
  TransportType.cs    # IConn интерфейс, типы транспорта
  CryptoUtils.cs      # AEAD (ChaCha20-Poly1305), Rekey, Zeroize
  Session.cs          # Конечный автомат, конфигурация

FProto.Tests/
  ConformanceTests.cs # xUnit conformance тесты
```

## Зависимости

- `System.Security.Cryptography.ChaCha20Poly1305` (.NET 8+)
- `System.Net.WebSockets` (встроен)
- xUnit — тесты
