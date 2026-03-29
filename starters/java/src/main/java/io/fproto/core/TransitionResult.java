package io.fproto.core;

public record TransitionResult(SessionStatus next, boolean destroyed) {}
