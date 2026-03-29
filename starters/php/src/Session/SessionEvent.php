<?php

declare(strict_types=1);

namespace FProto\Session;

enum SessionEvent: string
{
    case HandshakeOK = 'handshake_ok';
    case HandshakeFail = 'handshake_fail';
    case PacketReceived = 'packet_received';
    case IdleTimeout = 'idle_timeout';
    case ExpiryTimeout = 'expiry_timeout';
    case ExplicitLogout = 'explicit_logout';
}
