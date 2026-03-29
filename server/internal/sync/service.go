package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/qmish/FProto/server/internal/inbox"
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// Config holds SyncService configuration.
type Config struct {
	KafkaBrokers   []string
	GroupID        string
	RedisAddr      string
	PushServiceAddr string
}

// Service is the main SyncService that consumes from Kafka and delivers to Inbox + push.
type Service struct {
	cfg      Config
	handler  *Handler
	consumer sarama.ConsumerGroup
	topics   []string
}

// New creates a SyncService. It connects to Redis, Inbox store, and push service.
func New(ctx context.Context, cfg Config, inboxStore *inbox.Store) (*Service, error) {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	dedup := NewDedup(rdb)

	var pushClient pb.PushServiceClient
	if cfg.PushServiceAddr != "" {
		conn, err := grpc.NewClient(cfg.PushServiceAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("push service dial: %w", err)
		}
		pushClient = pb.NewPushServiceClient(conn)
	}

	handler := NewHandler(dedup, inboxStore, pushClient)

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version = sarama.V3_6_0_0
	kafkaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	kafkaCfg.Consumer.Offsets.AutoCommit.Enable = false

	consumer, err := sarama.NewConsumerGroup(cfg.KafkaBrokers, cfg.GroupID, kafkaCfg)
	if err != nil {
		return nil, fmt.Errorf("consumer group: %w", err)
	}

	return &Service{
		cfg:      cfg,
		handler:  handler,
		consumer: consumer,
		topics:   []string{}, // will be discovered dynamically
	}, nil
}

// DiscoverTopics fetches topic list from Kafka and filters user-* and group-* topics.
func (s *Service) DiscoverTopics(admin sarama.ClusterAdmin) error {
	topics, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("list topics: %w", err)
	}
	s.topics = nil
	for name := range topics {
		if strings.HasPrefix(name, "user-") || strings.HasPrefix(name, "group-") {
			s.topics = append(s.topics, name)
		}
	}
	log.Printf("sync service: discovered %d topics", len(s.topics))
	return nil
}

// Run starts the consumer loop. Blocks until ctx is cancelled.
func (s *Service) Run(ctx context.Context) error {
	if len(s.topics) == 0 {
		log.Println("sync service: no topics to consume, waiting for discovery...")
		time.Sleep(5 * time.Second)
		return nil
	}

	h := &syncGroupHandler{handler: s.handler}
	for {
		if err := s.consumer.Consume(ctx, s.topics, h); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("sync service consume error: %v", err)
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

// Close shuts down the service.
func (s *Service) Close() error {
	return s.consumer.Close()
}

type syncGroupHandler struct {
	handler *Handler
}

func (h *syncGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *syncGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *syncGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler.Handle(msg); err != nil {
			log.Printf("sync handler error: %v", err)
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
