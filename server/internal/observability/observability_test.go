package observability

import (
	"context"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger, err := NewLogger("test-service")
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	logger.Info("test message")
	_ = logger.Sync()
}

func TestLoggerWithTrace(t *testing.T) {
	logger, err := NewLogger("test-service")
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	enriched := LoggerWithTrace(logger, context.Background())
	enriched.Info("no trace context")
}

func TestNewMetrics(t *testing.T) {
	m, err := NewMetrics("test-service")
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	if m.MessagesSent == nil {
		t.Fatal("MessagesSent is nil")
	}
	if m.MessagesReceived == nil {
		t.Fatal("MessagesReceived is nil")
	}
	if m.HandshakeDuration == nil {
		t.Fatal("HandshakeDuration is nil")
	}
	if m.ActiveSessions == nil {
		t.Fatal("ActiveSessions is nil")
	}
	if m.ActiveConnections == nil {
		t.Fatal("ActiveConnections is nil")
	}
	if m.KafkaConsumerLag == nil {
		t.Fatal("KafkaConsumerLag is nil")
	}
	if m.OutboxPending == nil {
		t.Fatal("OutboxPending is nil")
	}
	if m.OutboxPublished == nil {
		t.Fatal("OutboxPublished is nil")
	}
	if m.OutboxDeadLetter == nil {
		t.Fatal("OutboxDeadLetter is nil")
	}
	if m.DispatchDuration == nil {
		t.Fatal("DispatchDuration is nil")
	}
	if m.MediaUploadDuration == nil {
		t.Fatal("MediaUploadDuration is nil")
	}
}

func TestInitMeter(t *testing.T) {
	mp, handler, err := InitMeter()
	if err != nil {
		t.Fatalf("InitMeter: %v", err)
	}
	if mp == nil {
		t.Fatal("MeterProvider is nil")
	}
	if handler == nil {
		t.Fatal("HTTP handler is nil")
	}
}

func TestTracer(t *testing.T) {
	tr := Tracer("test")
	if tr == nil {
		t.Fatal("Tracer is nil")
	}
}
