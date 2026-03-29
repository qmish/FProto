package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
)

// ProducerHeaderCarrier adapts sarama ProducerMessage headers for OTel propagation.
type ProducerHeaderCarrier struct {
	Headers *[]sarama.RecordHeader
}

func (c ProducerHeaderCarrier) Get(key string) string {
	for _, h := range *c.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c ProducerHeaderCarrier) Set(key string, value string) {
	for i, h := range *c.Headers {
		if string(h.Key) == key {
			(*c.Headers)[i].Value = []byte(value)
			return
		}
	}
	*c.Headers = append(*c.Headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c ProducerHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.Headers))
	for i, h := range *c.Headers {
		keys[i] = string(h.Key)
	}
	return keys
}

// ConsumerHeaderCarrier adapts sarama ConsumerMessage headers for OTel extraction.
type ConsumerHeaderCarrier struct {
	Headers []*sarama.RecordHeader
}

func (c ConsumerHeaderCarrier) Get(key string) string {
	for _, h := range c.Headers {
		if h != nil && string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c ConsumerHeaderCarrier) Set(key string, value string) {}

func (c ConsumerHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Headers))
	for _, h := range c.Headers {
		if h != nil {
			keys = append(keys, string(h.Key))
		}
	}
	return keys
}

// InjectTraceContext injects the current span context into Sarama producer message headers.
func InjectTraceContext(ctx context.Context, headers *[]sarama.RecordHeader) {
	otel.GetTextMapPropagator().Inject(ctx, ProducerHeaderCarrier{Headers: headers})
}

// ExtractTraceContext extracts span context from Sarama consumer message headers.
func ExtractTraceContext(ctx context.Context, headers []*sarama.RecordHeader) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, ConsumerHeaderCarrier{Headers: headers})
}
