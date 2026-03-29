<?php

declare(strict_types=1);

namespace FProto\Transport;

interface ConnInterface
{
    public function readMessage(): string;
    public function writeMessage(string $data): void;
    public function close(): void;
    public function transportType(): TransportType;
    public function remoteAddr(): string;
}
