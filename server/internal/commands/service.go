package commands

import (
	"context"
	"log"

	"github.com/IBM/sarama"
	kafkapkg "github.com/qmish/FProto/server/internal/kafka"
)

// Service consumes from the commands Kafka topic and delegates to Handler.
type Service struct {
	consumer sarama.ConsumerGroup
	handler  *Handler
}

func NewService(brokers []string, groupID string, handler *Handler) (*Service, error) {
	cfg := kafkapkg.DefaultConsumerConfig()
	consumer, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}
	return &Service{consumer: consumer, handler: handler}, nil
}

func (s *Service) Run(ctx context.Context) error {
	h := &commandGroupHandler{handler: s.handler}
	for {
		if err := s.consumer.Consume(ctx, []string{kafkapkg.TopicCommands}, h); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("command service consume error: %v", err)
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

func (s *Service) Close() error {
	return s.consumer.Close()
}

type commandGroupHandler struct {
	handler *Handler
}

func (h *commandGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *commandGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *commandGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.handler.Handle(msg); err != nil {
			log.Printf("command handler error: %v", err)
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
