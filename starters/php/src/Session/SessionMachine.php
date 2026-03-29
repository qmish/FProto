<?php

declare(strict_types=1);

namespace FProto\Session;

final class SessionMachine
{
    /** @var array<string, TransitionResult> */
    private static array $transitions = [];

    private static function init(): void
    {
        if (!empty(self::$transitions)) {
            return;
        }
        $t = static fn(SessionStatus $s, SessionEvent $e): string => $s->value . ':' . $e->value;

        self::$transitions[$t(SessionStatus::Connecting, SessionEvent::HandshakeOK)] =
            new TransitionResult(SessionStatus::Active, false);
        self::$transitions[$t(SessionStatus::Connecting, SessionEvent::HandshakeFail)] =
            new TransitionResult(SessionStatus::Expired, true);
        self::$transitions[$t(SessionStatus::Active, SessionEvent::PacketReceived)] =
            new TransitionResult(SessionStatus::Active, false);
        self::$transitions[$t(SessionStatus::Active, SessionEvent::IdleTimeout)] =
            new TransitionResult(SessionStatus::Sleeping, false);
        self::$transitions[$t(SessionStatus::Active, SessionEvent::ExplicitLogout)] =
            new TransitionResult(SessionStatus::Expired, false);
        self::$transitions[$t(SessionStatus::Sleeping, SessionEvent::PacketReceived)] =
            new TransitionResult(SessionStatus::Active, false);
        self::$transitions[$t(SessionStatus::Sleeping, SessionEvent::ExpiryTimeout)] =
            new TransitionResult(SessionStatus::Expired, false);
    }

    public static function transition(SessionStatus $current, SessionEvent $event): TransitionResult
    {
        self::init();
        $key = $current->value . ':' . $event->value;
        if (!isset(self::$transitions[$key])) {
            throw new InvalidTransitionException($current, $event);
        }
        return self::$transitions[$key];
    }
}
