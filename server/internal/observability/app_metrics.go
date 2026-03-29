package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all application-level metrics.
type Metrics struct {
	MessagesSent     metric.Int64Counter
	MessagesReceived metric.Int64Counter

	HandshakeDuration metric.Float64Histogram

	ActiveSessions    metric.Int64UpDownCounter
	ActiveConnections metric.Int64UpDownCounter

	KafkaConsumerLag metric.Int64Gauge

	OutboxPending     metric.Int64Gauge
	OutboxPublished   metric.Int64Counter
	OutboxDeadLetter  metric.Int64Counter

	DispatchDuration     metric.Float64Histogram
	MediaUploadDuration  metric.Float64Histogram
}

// NewMetrics registers all application metrics with the global OTel meter.
func NewMetrics(serviceName string) (*Metrics, error) {
	meter := otel.Meter(serviceName)
	m := &Metrics{}
	var err error

	m.MessagesSent, err = meter.Int64Counter("fproto.messages_sent_total",
		metric.WithDescription("Total messages sent"))
	if err != nil {
		return nil, err
	}

	m.MessagesReceived, err = meter.Int64Counter("fproto.messages_received_total",
		metric.WithDescription("Total messages received"))
	if err != nil {
		return nil, err
	}

	m.HandshakeDuration, err = meter.Float64Histogram("fproto.handshake_duration_seconds",
		metric.WithDescription("Duration of Noise handshake"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	m.ActiveSessions, err = meter.Int64UpDownCounter("fproto.active_sessions",
		metric.WithDescription("Number of active sessions"))
	if err != nil {
		return nil, err
	}

	m.ActiveConnections, err = meter.Int64UpDownCounter("fproto.active_connections",
		metric.WithDescription("Number of active transport connections"))
	if err != nil {
		return nil, err
	}

	m.KafkaConsumerLag, err = meter.Int64Gauge("fproto.kafka_consumer_lag",
		metric.WithDescription("Kafka consumer group lag"))
	if err != nil {
		return nil, err
	}

	m.OutboxPending, err = meter.Int64Gauge("fproto.outbox_pending",
		metric.WithDescription("Number of pending outbox entries"))
	if err != nil {
		return nil, err
	}

	m.OutboxPublished, err = meter.Int64Counter("fproto.outbox_published_total",
		metric.WithDescription("Total outbox entries published"))
	if err != nil {
		return nil, err
	}

	m.OutboxDeadLetter, err = meter.Int64Counter("fproto.outbox_dead_lettered_total",
		metric.WithDescription("Total outbox entries sent to dead letter"))
	if err != nil {
		return nil, err
	}

	m.DispatchDuration, err = meter.Float64Histogram("fproto.dispatch_duration_seconds",
		metric.WithDescription("Duration of message dispatch"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	m.MediaUploadDuration, err = meter.Float64Histogram("fproto.media_upload_duration_seconds",
		metric.WithDescription("Duration of media upload initiation"),
		metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}

	return m, nil
}
