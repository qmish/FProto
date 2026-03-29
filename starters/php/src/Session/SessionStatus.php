<?php

declare(strict_types=1);

namespace FProto\Session;

enum SessionStatus: string
{
    case Connecting = 'connecting';
    case Active = 'active';
    case Sleeping = 'sleeping';
    case Expired = 'expired';
}
