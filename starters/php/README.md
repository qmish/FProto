# FProto PHP Starter Pack

PHP SDK для протокола FProto — пакет `fproto/core`.

## Требования

- PHP 8.2+
- ext-sodium

## Установка

```bash
composer install
```

## Тесты

```bash
vendor/bin/phpunit
```

## Структура

```
src/
  Transport/
    TransportType.php       # Типы транспорта (enum)
    ConnInterface.php       # Интерфейс соединения
  Crypto/
    CryptoUtils.php         # AEAD (sodium), Rekey (HMAC-SHA256), Zeroize
  Session/
    SessionStatus.php       # Состояния сессии (enum)
    SessionEvent.php        # События сессии (enum)
    SessionMachine.php      # Конечный автомат
    SessionConfig.php       # Конфигурация
    TransitionResult.php    # Результат перехода
    InvalidTransitionException.php

tests/
  ConformanceTest.php       # PHPUnit conformance тесты
```

## Зависимости

- `ext-sodium` — ChaCha20-Poly1305 AEAD + zeroize
- PHPUnit — тесты

## Примечание по Noise XX

Для полного Noise XX handshake на PHP рекомендуется использовать FFI-обёртку
над C-библиотекой `noise-c`. Текущий стартер-пак содержит AEAD-слой и конечный
автомат сессий. Noise XX будет добавлен после подготовки FFI-биндинга.
