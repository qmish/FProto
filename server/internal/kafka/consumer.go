package kafka

import (
	"context"
	"fmt"
	"log"

	"github.com/IBM/sarama"
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

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler(msg); err != nil {
			log.Printf("message handler error (topic=%s, partition=%d, offset=%d): %v",
				msg.Topic, msg.Partition, msg.Offset, err)
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
