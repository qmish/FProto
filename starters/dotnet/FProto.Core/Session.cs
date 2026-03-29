namespace FProto.Core;

public enum SessionStatus
{
    Connecting,
    Active,
    Sleeping,
    Expired
}

public enum SessionEvent
{
    HandshakeOK,
    HandshakeFail,
    PacketReceived,
    IdleTimeout,
    ExpiryTimeout,
    ExplicitLogout
}

public record TransitionResult(SessionStatus Next, bool Destroyed);

public class InvalidTransitionException : Exception
{
    public SessionStatus Status { get; }
    public SessionEvent Event { get; }

    public InvalidTransitionException(SessionStatus status, SessionEvent ev)
        : base($"invalid transition: {status} + {ev}")
    {
        Status = status;
        Event = ev;
    }
}

public static class SessionMachine
{
    private static readonly Dictionary<(SessionStatus, SessionEvent), TransitionResult> Transitions = new()
    {
        [(SessionStatus.Connecting, SessionEvent.HandshakeOK)] = new(SessionStatus.Active, false),
        [(SessionStatus.Connecting, SessionEvent.HandshakeFail)] = new(SessionStatus.Expired, true),
        [(SessionStatus.Active, SessionEvent.PacketReceived)] = new(SessionStatus.Active, false),
        [(SessionStatus.Active, SessionEvent.IdleTimeout)] = new(SessionStatus.Sleeping, false),
        [(SessionStatus.Active, SessionEvent.ExplicitLogout)] = new(SessionStatus.Expired, false),
        [(SessionStatus.Sleeping, SessionEvent.PacketReceived)] = new(SessionStatus.Active, false),
        [(SessionStatus.Sleeping, SessionEvent.ExpiryTimeout)] = new(SessionStatus.Expired, false),
    };

    public static TransitionResult Transition(SessionStatus current, SessionEvent ev)
    {
        if (Transitions.TryGetValue((current, ev), out var result))
            return result;
        throw new InvalidTransitionException(current, ev);
    }
}

public record SessionConfig(
    int IdleTimeoutSecs = 300,
    int ExpiryTimeoutSecs = 86400,
    int HandshakeTimeoutSecs = 30,
    int MaxConnections = 10000
);
