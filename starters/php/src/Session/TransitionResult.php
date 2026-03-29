<?php

declare(strict_types=1);

namespace FProto\Session;

final readonly class TransitionResult
{
    public function __construct(
        public SessionStatus $next,
        public bool $destroyed,
    ) {}
}
