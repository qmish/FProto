using System.Buffers.Binary;
using System.Security.Cryptography;

namespace FProto.Core;

public static class CryptoUtils
{
    public static byte[] BuildNonce(ulong seq)
    {
        var nonce = new byte[12];
        BinaryPrimitives.WriteUInt64LittleEndian(nonce.AsSpan(4), seq);
        return nonce;
    }

    public static byte[] BuildAad(byte[] sessionId, ulong seq)
    {
        var aad = new byte[sessionId.Length + 8];
        sessionId.CopyTo(aad, 0);
        BinaryPrimitives.WriteUInt64LittleEndian(aad.AsSpan(sessionId.Length), seq);
        return aad;
    }

    public static byte[] EncryptAead(byte[] key, byte[] sessionId, ulong seq, byte[] plaintext)
    {
        if (key.Length != 32)
            throw new ArgumentException($"invalid key size: expected 32, got {key.Length}");

        var nonce = BuildNonce(seq);
        var aad = BuildAad(sessionId, seq);

        using var cipher = new ChaCha20Poly1305(key);
        var ciphertext = new byte[plaintext.Length];
        var tag = new byte[16];
        cipher.Encrypt(nonce, plaintext, ciphertext, tag, aad);

        var result = new byte[ciphertext.Length + tag.Length];
        ciphertext.CopyTo(result, 0);
        tag.CopyTo(result, ciphertext.Length);
        return result;
    }

    public static byte[] DecryptAead(byte[] key, byte[] sessionId, ulong seq, byte[] ciphertextWithTag)
    {
        if (key.Length != 32)
            throw new ArgumentException($"invalid key size: expected 32, got {key.Length}");

        var nonce = BuildNonce(seq);
        var aad = BuildAad(sessionId, seq);

        var ciphertext = ciphertextWithTag.AsSpan(0, ciphertextWithTag.Length - 16);
        var tag = ciphertextWithTag.AsSpan(ciphertextWithTag.Length - 16);

        using var cipher = new ChaCha20Poly1305(key);
        var plaintext = new byte[ciphertext.Length];
        cipher.Decrypt(nonce, ciphertext, tag, plaintext, aad);
        return plaintext;
    }

    public static byte[] Rekey(byte[] currentKey)
    {
        var prk = HMACSHA256.HashData(Array.Empty<byte>(), currentKey);
        var info = "fproto-rekey"u8.ToArray();
        var input = new byte[info.Length + 1];
        info.CopyTo(input, 0);
        input[^1] = 0x01;
        var okm = HMACSHA256.HashData(prk, input);
        return okm[..32];
    }

    public static void Zeroize(byte[] buf)
    {
        CryptographicOperations.ZeroMemory(buf);
    }
}
