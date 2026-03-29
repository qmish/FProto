<?php

declare(strict_types=1);

namespace FProto\Crypto;

use InvalidArgumentException;
use RuntimeException;

final class CryptoUtils
{
    public static function buildNonce(int $seq): string
    {
        return str_repeat("\0", 4) . pack('P', $seq);
    }

    public static function buildAad(string $sessionId, int $seq): string
    {
        return $sessionId . pack('P', $seq);
    }

    public static function encryptAead(string $key, string $sessionId, int $seq, string $plaintext): string
    {
        if (strlen($key) !== 32) {
            throw new InvalidArgumentException(
                sprintf('invalid key size: expected 32, got %d', strlen($key))
            );
        }

        $nonce = self::buildNonce($seq);
        $aad = self::buildAad($sessionId, $seq);

        $ciphertext = sodium_crypto_aead_chacha20poly1305_ietf_encrypt(
            $plaintext,
            $aad,
            $nonce,
            $key
        );

        if ($ciphertext === false) {
            throw new RuntimeException('AEAD encryption failed');
        }

        return $ciphertext;
    }

    public static function decryptAead(string $key, string $sessionId, int $seq, string $ciphertext): string
    {
        if (strlen($key) !== 32) {
            throw new InvalidArgumentException(
                sprintf('invalid key size: expected 32, got %d', strlen($key))
            );
        }

        $nonce = self::buildNonce($seq);
        $aad = self::buildAad($sessionId, $seq);

        $plaintext = sodium_crypto_aead_chacha20poly1305_ietf_decrypt(
            $ciphertext,
            $aad,
            $nonce,
            $key
        );

        if ($plaintext === false) {
            throw new RuntimeException('AEAD decryption failed');
        }

        return $plaintext;
    }

    public static function rekey(string $currentKey): string
    {
        $prk = hash_hmac('sha256', $currentKey, '', true);
        $info = 'fproto-rekey';
        $okm = hash_hmac('sha256', $info . "\x01", $prk, true);
        return substr($okm, 0, 32);
    }

    public static function zeroize(string &$buf): void
    {
        sodium_memzero($buf);
    }
}
