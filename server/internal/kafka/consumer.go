package kafka

import (
	"context"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// MessageHandler processes a single Kafka message.
// Return nil to commit, non-nil to retry.
type MessageHandler func(msg *sarama.ConsumerMessage) error

// ConsumerGroup wraps sarama.ConsumerGroup with a handler loop.
type ConsumerGroup struct {
	group   sarama.ConsumerGroup
	handler MessageHandler
	topics  []string
	ready   chan struct{}
}

// NewConsumerGroup creates a consumer group that processes messages via handler.
func NewConsumerGroup(brokers []string, groupID string, topics []string, cfg *sarama.Config, handler MessageHandler) (*ConsumerGroup, error) {
	if cfg == nil {
		cfg = DefaultConsumerConfig()
	}
	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, fmt.Errorf("new consumer group: %w", err)
	}
	return &ConsumerGroup{
		group:   group,
		handler: handler,
		topics:  topics,
		ready:   make(chan struct{}),
	}, nil
}

// Run starts consuming in a loop until context is cancelled.
func (cg *ConsumerGroup) Run(ctx context.Context) error {
	h := &consumerGroupHandler{
		handler: cg.handler,
		ready:   cg.ready,
	}

	for {
		if err := cg.group.Consume(ctx, cg.topics, h); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("consumer group error: %v", err)
		}
		if ctx.Err() != nil {
			return nil
		}
		cg.ready = make(chan struct{})
		h.ready = cg.ready
	}
}

// Ready returns a channel that is closed when the consumer is ready.
func (cg *ConsumerGroup) Ready() <-chan struct{} {
	return cg.ready
}

// Close shuts down the consumer group.
func (cg *ConsumerGroup) Close() error {
	return cg.group.Close()
}

type consumerGroupHandler struct {
	handler MessageHandler
	ready   chan struct{}
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := ExtractTraceContext(sess.Context(), msg.Headers)

		tracer := otel.Tracer("kafka.consumer")
		_, span := tracer.Start(ctx, "kafka.consume",
			oteltrace.WithAttributes(
				attribute.String("kafka.topic", msg.Topic),
				attribute.Int64("kafka.partition", int64(msg.Partition)),
				attribute.Int64("kafka.offset", msg.Offset),
			),
		)

		if err := h.handler(msg); err != nil {
			span.RecordError(err)
			span.SetStatus(otelcodes.Error, err.Error())
			log.Printf("message handler error (topic=%s, partition=%d, offset=%d): %v",
				msg.Topic, msg.Partition, msg.Offset, err)
			span.End()
			continue
		}
		span.End()
		sess.MarkMessage(msg, "")
	}
	return nil
}
