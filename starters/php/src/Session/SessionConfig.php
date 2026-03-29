<?php

declare(strict_types=1);

namespace FProto\Session;

final readonly class SessionConfig
{
    public function __construct(
        public int $idleTimeoutSecs = 300,
        public int $expiryTimeoutSecs = 86400,
        public int $handshakeTimeoutSecs = 30,
        public int $maxConnections = 10000,
    ) {}
}
