<?php

declare(strict_types=1);

namespace FProto\Transport;

enum TransportType: string
{
    case WebSocket = 'websocket';
    case Quic = 'quic';
    case Grpc = 'grpc';
}
