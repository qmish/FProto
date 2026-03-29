package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
)

// Producer wraps sarama.SyncProducer with convenience methods.
type Producer struct {
	sp     sarama.SyncProducer
	mu     sync.Mutex
	closed bool
}

// NewProducer creates a synchronous Kafka producer.
func NewProducer(brokers []string, cfg *sarama.Config) (*Producer, error) {
	if cfg == nil {
		cfg = DefaultProducerConfig()
	}
	sp, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("new sync producer: %w", err)
	}
	return &Producer{sp: sp}, nil
}

// Send publishes a message to Kafka and returns partition+offset.
// It injects trace context from ctx into message headers.
func (p *Producer) Send(topic string, key, value []byte) (partition int32, offset int64, err error) {
	return p.SendWithContext(context.Background(), topic, key, value)
}

// SendWithContext publishes a message with trace context propagation.
func (p *Producer) SendWithContext(ctx context.Context, topic string, key, value []byte) (partition int32, offset int64, err error) {
	var headers []sarama.RecordHeader
	InjectTraceContext(ctx, &headers)

	msg := &sarama.ProducerMessage{
		Topic:   topic,
		Value:   sarama.ByteEncoder(value),
		Headers: headers,
	}
	if key != nil {
		msg.Key = sarama.ByteEncoder(key)
	}
	return p.sp.SendMessage(msg)
}

// Close shuts down the producer.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	return p.sp.Close()
}
