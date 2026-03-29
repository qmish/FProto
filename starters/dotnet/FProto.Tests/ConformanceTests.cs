using FProto.Core;
using Xunit;

namespace FProto.Tests;

public class CryptoTests
{
    private static byte[] TestKey()
    {
        var key = new byte[32];
        for (int i = 0; i < 32; i++) key[i] = (byte)i;
        return key;
    }

    [Fact]
    public void AeadEncryptDecrypt()
    {
        var key = TestKey();
        var sid = "test-session"u8.ToArray();
        var plaintext = "secret data"u8.ToArray();

        var ct = CryptoUtils.EncryptAead(key, sid, 1, plaintext);
        var pt = CryptoUtils.DecryptAead(key, sid, 1, ct);
        Assert.Equal(plaintext, pt);
    }

    [Fact]
    public void AeadWrongSeqFails()
    {
        var key = TestKey();
        var sid = "session"u8.ToArray();
        var ct = CryptoUtils.EncryptAead(key, sid, 1, "data"u8.ToArray());
        Assert.ThrowsAny<Exception>(() => CryptoUtils.DecryptAead(key, sid, 2, ct));
    }

    [Fact]
    public void AeadWrongSessionFails()
    {
        var key = TestKey();
        var ct = CryptoUtils.EncryptAead(key, "session-a"u8.ToArray(), 1, "data"u8.ToArray());
        Assert.ThrowsAny<Exception>(() => CryptoUtils.DecryptAead(key, "session-b"u8.ToArray(), 1, ct));
    }

    [Fact]
    public void AeadTamperDetection()
    {
        var key = TestKey();
        var sid = "session"u8.ToArray();
        var ct = CryptoUtils.EncryptAead(key, sid, 1, "data"u8.ToArray());
        ct[0] ^= 0xFF;
        Assert.ThrowsAny<Exception>(() => CryptoUtils.DecryptAead(key, sid, 1, ct));
    }

    [Fact]
    public void LargePayloadAead()
    {
        var key = TestKey();
        var sid = "large"u8.ToArray();
        var payload = new byte[32768];
        for (int i = 0; i < payload.Length; i++) payload[i] = (byte)(i % 256);

        var ct = CryptoUtils.EncryptAead(key, sid, 1, payload);
        var pt = CryptoUtils.DecryptAead(key, sid, 1, ct);
        Assert.Equal(payload, pt);
    }

    [Fact]
    public void SequentialAead()
    {
        var key = TestKey();
        var sid = "seq-session"u8.ToArray();

        for (ulong seq = 0; seq < 100; seq++)
        {
            var msg = System.Text.Encoding.UTF8.GetBytes($"message-{seq}");
            var ct = CryptoUtils.EncryptAead(key, sid, seq, msg);
            var pt = CryptoUtils.DecryptAead(key, sid, seq, ct);
            Assert.Equal(msg, pt);
        }
    }

    [Fact]
    public void InvalidKeySize()
    {
        Assert.Throws<ArgumentException>(() =>
            CryptoUtils.EncryptAead(new byte[16], "s"u8.ToArray(), 0, "d"u8.ToArray()));
    }

    [Fact]
    public void NonceUniqueness()
    {
        var seen = new HashSet<string>();
        for (ulong i = 0; i < 1000; i++)
        {
            var n = Convert.ToHexString(CryptoUtils.BuildNonce(i));
            Assert.True(seen.Add(n), $"duplicate nonce at seq {i}");
        }
    }

    [Fact]
    public void NonceStructure()
    {
        var n0 = CryptoUtils.BuildNonce(0);
        Assert.Equal(new byte[12], n0);
    }

    [Fact]
    public void BuildAadStructure()
    {
        var sid = "session-1"u8.ToArray();
        var aad = CryptoUtils.BuildAad(sid, 42);
        Assert.Equal(sid.Length + 8, aad.Length);
        Assert.Equal(sid, aad[..sid.Length]);
    }

    [Fact]
    public void RekeyDeterministic()
    {
        var key = TestKey();
        var k1 = CryptoUtils.Rekey(key);
        var k2 = CryptoUtils.Rekey(key);
        Assert.Equal(k1, k2);
        Assert.NotEqual(key, k1);
        Assert.Equal(32, k1.Length);
    }

    [Fact]
    public void RekeyChain()
    {
        var key = TestKey();
        for (int i = 0; i < 10; i++)
        {
            var newKey = CryptoUtils.Rekey(key);
            Assert.NotEqual(key, newKey);
            Assert.Equal(32, newKey.Length);
            key = newKey;
        }
    }

    [Fact]
    public void Zeroize()
    {
        var buf = new byte[] { 0xAA, 0xBB, 0xCC };
        CryptoUtils.Zeroize(buf);
        Assert.All(buf, b => Assert.Equal(0, b));
    }
}

public class SessionTests
{
    [Fact]
    public void ConnectingHandshakeOk()
    {
        var r = SessionMachine.Transition(SessionStatus.Connecting, SessionEvent.HandshakeOK);
        Assert.Equal(SessionStatus.Active, r.Next);
        Assert.False(r.Destroyed);
    }

    [Fact]
    public void ConnectingHandshakeFail()
    {
        var r = SessionMachine.Transition(SessionStatus.Connecting, SessionEvent.HandshakeFail);
        Assert.True(r.Destroyed);
    }

    [Fact]
    public void ActiveIdleTimeout()
    {
        var r = SessionMachine.Transition(SessionStatus.Active, SessionEvent.IdleTimeout);
        Assert.Equal(SessionStatus.Sleeping, r.Next);
        Assert.False(r.Destroyed);
    }

    [Fact]
    public void ActivePacketReceived()
    {
        var r = SessionMachine.Transition(SessionStatus.Active, SessionEvent.PacketReceived);
        Assert.Equal(SessionStatus.Active, r.Next);
    }

    [Fact]
    public void ActiveExplicitLogout()
    {
        var r = SessionMachine.Transition(SessionStatus.Active, SessionEvent.ExplicitLogout);
        Assert.Equal(SessionStatus.Expired, r.Next);
    }

    [Fact]
    public void SleepingPacketReceived()
    {
        var r = SessionMachine.Transition(SessionStatus.Sleeping, SessionEvent.PacketReceived);
        Assert.Equal(SessionStatus.Active, r.Next);
    }

    [Fact]
    public void SleepingExpiryTimeout()
    {
        var r = SessionMachine.Transition(SessionStatus.Sleeping, SessionEvent.ExpiryTimeout);
        Assert.Equal(SessionStatus.Expired, r.Next);
        Assert.False(r.Destroyed);
    }

    [Fact]
    public void InvalidTransitions()
    {
        Assert.Throws<InvalidTransitionException>(
            () => SessionMachine.Transition(SessionStatus.Expired, SessionEvent.PacketReceived));
        Assert.Throws<InvalidTransitionException>(
            () => SessionMachine.Transition(SessionStatus.Connecting, SessionEvent.IdleTimeout));
        Assert.Throws<InvalidTransitionException>(
            () => SessionMachine.Transition(SessionStatus.Sleeping, SessionEvent.HandshakeOK));
    }

    [Fact]
    public void FullLifecycle()
    {
        var r = SessionMachine.Transition(SessionStatus.Connecting, SessionEvent.HandshakeOK);
        Assert.Equal(SessionStatus.Active, r.Next);

        r = SessionMachine.Transition(r.Next, SessionEvent.PacketReceived);
        Assert.Equal(SessionStatus.Active, r.Next);

        r = SessionMachine.Transition(r.Next, SessionEvent.IdleTimeout);
        Assert.Equal(SessionStatus.Sleeping, r.Next);

        r = SessionMachine.Transition(r.Next, SessionEvent.PacketReceived);
        Assert.Equal(SessionStatus.Active, r.Next);

        r = SessionMachine.Transition(r.Next, SessionEvent.IdleTimeout);
        r = SessionMachine.Transition(r.Next, SessionEvent.ExpiryTimeout);
        Assert.Equal(SessionStatus.Expired, r.Next);
    }

    [Fact]
    public void DefaultConfig()
    {
        var cfg = new SessionConfig();
        Assert.Equal(300, cfg.IdleTimeoutSecs);
        Assert.Equal(86400, cfg.ExpiryTimeoutSecs);
        Assert.Equal(30, cfg.HandshakeTimeoutSecs);
        Assert.Equal(10000, cfg.MaxConnections);
    }
}
