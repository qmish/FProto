package outbox

import (
	"context"
	"log"
	"time"

	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
)

const (
	DefaultPollInterval = 500 * time.Millisecond
	DefaultBatchSize    = 100
	MaxRetries          = 5
)

// Relay polls the outbox table and publishes pending entries to Kafka.
type Relay struct {
	store    *Store
	producer *kafkapkg.Producer
	interval time.Duration
	batch    int
}

func NewRelay(store *Store, producer *kafkapkg.Producer) *Relay {
	return &Relay{
		store:    store,
		producer: producer,
		interval: DefaultPollInterval,
		batch:    DefaultBatchSize,
	}
}

// Run starts the relay loop. Blocks until ctx is cancelled.
func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.poll(ctx)
		}
	}
}

func (r *Relay) poll(ctx context.Context) {
	entries, err := r.store.FetchPending(ctx, r.batch)
	if err != nil {
		log.Printf("outbox relay: fetch pending: %v", err)
		return
	}

	for _, e := range entries {
		if err := r.publish(ctx, e); err != nil {
			log.Printf("outbox relay: publish id=%d topic=%s: %v", e.ID, e.Topic, err)
			_ = r.store.IncrementRetry(ctx, e.ID)
			if e.RetryCount+1 >= MaxRetries {
				r.sendToDeadLetter(ctx, e)
			}
		}
	}
}

func (r *Relay) publish(ctx context.Context, e Entry) error {
	_, _, err := r.producer.Send(e.Topic, e.PartitionKey, e.Payload)
	if err != nil {
		return err
	}
	return r.store.MarkPublished(ctx, e.ID)
}

func (r *Relay) sendToDeadLetter(ctx context.Context, e Entry) {
	_, _, err := r.producer.Send(kafkapkg.TopicDeadLetter, []byte(e.Topic), e.Payload)
	if err != nil {
		log.Printf("outbox relay: dead-letter failed for id=%d: %v", e.ID, err)
		return
	}
	if err := r.store.MarkFailed(ctx, e.ID); err != nil {
		log.Printf("outbox relay: mark failed for id=%d: %v", e.ID, err)
	}
}

// PublishOne is a convenience for testing: immediately publishes a single entry.
func (r *Relay) PublishOne(ctx context.Context, e Entry) error {
	return r.publish(ctx, e)
}
