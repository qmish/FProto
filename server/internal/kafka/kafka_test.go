package kafka

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
)

func TestDefaultProducerConfig(t *testing.T) {
	cfg := DefaultProducerConfig()
	if cfg.Producer.RequiredAcks != sarama.WaitForAll {
		t.Fatal("expected acks=all")
	}
	if !cfg.Producer.Idempotent {
		t.Fatal("expected idempotent")
	}
	if cfg.Producer.Compression != sarama.CompressionLZ4 {
		t.Fatal("expected LZ4 compression")
	}
	if cfg.Net.MaxOpenRequests != 1 {
		t.Fatal("expected MaxOpenRequests=1 for idempotent")
	}
}

func TestDefaultConsumerConfig(t *testing.T) {
	cfg := DefaultConsumerConfig()
	if cfg.Consumer.Offsets.Initial != sarama.OffsetOldest {
		t.Fatal("expected OffsetOldest")
	}
	if cfg.Consumer.Offsets.AutoCommit.Enable {
		t.Fatal("expected manual commit")
	}
	if cfg.Consumer.IsolationLevel != sarama.ReadCommitted {
		t.Fatal("expected ReadCommitted")
	}
}

func TestMockProducer_Send(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	mp := mocks.NewSyncProducer(t, cfg)
	mp.ExpectSendMessageAndSucceed()

	p := &Producer{sp: mp}
	partition, offset, err := p.Send("test-topic", []byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if partition != 0 || offset != 0 {
		t.Logf("partition=%d, offset=%d", partition, offset)
	}
}

func TestMockProducer_SendError(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	mp := mocks.NewSyncProducer(t, cfg)
	mp.ExpectSendMessageAndFail(sarama.ErrBrokerNotAvailable)

	p := &Producer{sp: mp}
	_, _, err := p.Send("test-topic", nil, []byte("value"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockProducer_MultipleMessages(t *testing.T) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	mp := mocks.NewSyncProducer(t, cfg)

	for i := 0; i < 10; i++ {
		mp.ExpectSendMessageAndSucceed()
	}

	p := &Producer{sp: mp}
	for i := 0; i < 10; i++ {
		_, _, err := p.Send("topic", []byte("k"), []byte("v"))
		if err != nil {
			t.Fatalf("msg %d: %v", i, err)
		}
	}
}

func TestUserTopic(t *testing.T) {
	uid := []byte{0x01, 0x02, 0x03, 0x04}
	topic := UserTopic(uid)
	if topic != "user-01020304" {
		t.Fatalf("expected user-01020304, got %s", topic)
	}
}

func TestGroupTopic(t *testing.T) {
	gid := []byte{0xaa, 0xbb}
	topic := GroupTopic(gid)
	if topic != "group-aabb" {
		t.Fatalf("expected group-aabb, got %s", topic)
	}
}

func TestStaticTopics(t *testing.T) {
	specs := StaticTopics(1)
	if len(specs) != 4 {
		t.Fatalf("expected 4 static topics, got %d", len(specs))
	}
	names := map[string]bool{}
	for _, s := range specs {
		names[s.Name] = true
	}
	for _, name := range []string{TopicCommands, TopicEvents, TopicMediaEvents, TopicDeadLetter} {
		if !names[name] {
			t.Fatalf("missing topic: %s", name)
		}
	}
}
