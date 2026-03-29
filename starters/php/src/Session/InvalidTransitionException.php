<?php

declare(strict_types=1);

namespace FProto\Session;

use RuntimeException;

final class InvalidTransitionException extends RuntimeException
{
    public function __construct(
        public readonly SessionStatus $status,
        public readonly SessionEvent $event,
    ) {
        parent::__construct(
            sprintf('invalid transition: %s + %s', $status->value, $event->value)
        );
    }
}
