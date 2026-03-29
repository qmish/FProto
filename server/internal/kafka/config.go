package kafka

import (
	"time"

	"github.com/IBM/sarama"
)

// DefaultProducerConfig returns a sarama config tuned for reliable delivery:
// acks=all, idempotent, LZ4 compression, retries.
func DefaultProducerConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0

	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Idempotent = true
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	cfg.Producer.Retry.Max = 5
	cfg.Producer.Retry.Backoff = 100 * time.Millisecond
	cfg.Producer.Compression = sarama.CompressionLZ4
	cfg.Producer.MaxMessageBytes = 4 * 1024 * 1024 // 4 MB

	cfg.Net.MaxOpenRequests = 1 // required for idempotent producer

	return cfg
}

// DefaultConsumerConfig returns a sarama config for consumer groups
// with manual commit and tuned fetch settings.
func DefaultConsumerConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0

	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Offsets.AutoCommit.Enable = false
	cfg.Consumer.Fetch.Min = 1
	cfg.Consumer.Fetch.Default = 1024 * 1024
	cfg.Consumer.MaxWaitTime = 500 * time.Millisecond
	cfg.Consumer.IsolationLevel = sarama.ReadCommitted

	return cfg
}
