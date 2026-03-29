package io.fproto.core;

import javax.crypto.Cipher;
import javax.crypto.Mac;
import javax.crypto.spec.IvParameterSpec;
import javax.crypto.spec.SecretKeySpec;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.util.Arrays;

public final class CryptoUtils {

    private CryptoUtils() {}

    public static KeyPair generateKeyPair() throws Exception {
        KeyPairGenerator kpg = KeyPairGenerator.getInstance("X25519");
        return kpg.generateKeyPair();
    }

    public static byte[] buildNonce(long seq) {
        byte[] nonce = new byte[12];
        ByteBuffer.wrap(nonce, 4, 8).order(ByteOrder.LITTLE_ENDIAN).putLong(seq);
        return nonce;
    }

    public static byte[] buildAad(byte[] sessionId, long seq) {
        byte[] aad = new byte[sessionId.length + 8];
        System.arraycopy(sessionId, 0, aad, 0, sessionId.length);
        ByteBuffer.wrap(aad, sessionId.length, 8).order(ByteOrder.LITTLE_ENDIAN).putLong(seq);
        return aad;
    }

    public static byte[] encryptAead(byte[] key, byte[] sessionId, long seq, byte[] plaintext) throws Exception {
        if (key.length != 32) {
            throw new IllegalArgumentException("invalid key size: expected 32, got " + key.length);
        }
        byte[] nonce = buildNonce(seq);
        byte[] aad = buildAad(sessionId, seq);

        Cipher cipher = Cipher.getInstance("ChaCha20-Poly1305");
        SecretKeySpec keySpec = new SecretKeySpec(key, "ChaCha20");
        IvParameterSpec ivSpec = new IvParameterSpec(nonce);
        cipher.init(Cipher.ENCRYPT_MODE, keySpec, ivSpec);
        cipher.updateAAD(aad);
        return cipher.doFinal(plaintext);
    }

    public static byte[] decryptAead(byte[] key, byte[] sessionId, long seq, byte[] ciphertext) throws Exception {
        if (key.length != 32) {
            throw new IllegalArgumentException("invalid key size: expected 32, got " + key.length);
        }
        byte[] nonce = buildNonce(seq);
        byte[] aad = buildAad(sessionId, seq);

        Cipher cipher = Cipher.getInstance("ChaCha20-Poly1305");
        SecretKeySpec keySpec = new SecretKeySpec(key, "ChaCha20");
        IvParameterSpec ivSpec = new IvParameterSpec(nonce);
        cipher.init(Cipher.DECRYPT_MODE, keySpec, ivSpec);
        cipher.updateAAD(aad);
        return cipher.doFinal(ciphertext);
    }

    public static byte[] rekey(byte[] currentKey) throws Exception {
        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(new byte[0], "HmacSHA256"));
        byte[] prk = mac.doFinal(currentKey);

        Mac mac2 = Mac.getInstance("HmacSHA256");
        mac2.init(new SecretKeySpec(prk, "HmacSHA256"));
        mac2.update("fproto-rekey".getBytes());
        mac2.update((byte) 0x01);
        byte[] okm = mac2.doFinal();
        return Arrays.copyOf(okm, 32);
    }

    public static void zeroize(byte[] buf) {
        Arrays.fill(buf, (byte) 0);
    }
}
