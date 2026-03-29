package kafka

import (
	"encoding/hex"
	"fmt"

	"github.com/IBM/sarama"
)

// Default partition counts per topic type.
const (
	UserPartitions       int32 = 128
	GroupPartitions      int32 = 64
	CommandPartitions    int32 = 32
	EventPartitions      int32 = 64
	MediaEventPartitions int32 = 32
	DeadLetterParts      int32 = 8
)

const (
	TopicCommands    = "commands"
	TopicEvents      = "events"
	TopicMediaEvents = "media-events"
	TopicDeadLetter  = "dead-letter"
)

// UserTopic returns the topic name for a specific user.
func UserTopic(userID []byte) string {
	return "user-" + hex.EncodeToString(userID)
}

// GroupTopic returns the topic name for a specific group.
func GroupTopic(groupID []byte) string {
	return "group-" + hex.EncodeToString(groupID)
}

// TopicSpec defines a topic to create.
type TopicSpec struct {
	Name       string
	Partitions int32
	Replicas   int16
	Retention  string // e.g. "2592000000" for 30 days in ms
}

// StaticTopics returns the fixed topics that should always exist.
func StaticTopics(replicas int16) []TopicSpec {
	return []TopicSpec{
		{Name: TopicCommands, Partitions: CommandPartitions, Replicas: replicas, Retention: "604800000"},
		{Name: TopicEvents, Partitions: EventPartitions, Replicas: replicas, Retention: "7776000000"},
		{Name: TopicMediaEvents, Partitions: MediaEventPartitions, Replicas: replicas, Retention: "2592000000"},
		{Name: TopicDeadLetter, Partitions: DeadLetterParts, Replicas: replicas, Retention: "7776000000"},
	}
}

// EnsureTopics creates topics if they don't exist using the cluster admin API.
func EnsureTopics(admin sarama.ClusterAdmin, specs []TopicSpec) error {
	existing, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("list topics: %w", err)
	}

	for _, spec := range specs {
		if _, ok := existing[spec.Name]; ok {
			continue
		}
		detail := &sarama.TopicDetail{
			NumPartitions:     spec.Partitions,
			ReplicationFactor: spec.Replicas,
			ConfigEntries:     map[string]*string{},
		}
		if spec.Retention != "" {
			detail.ConfigEntries["retention.ms"] = &spec.Retention
		}
		if err := admin.CreateTopic(spec.Name, detail, false); err != nil {
			return fmt.Errorf("create topic %s: %w", spec.Name, err)
		}
	}
	return nil
}

// EnsureDynamicTopic creates a user- or group- topic on-demand if it doesn't exist.
func EnsureDynamicTopic(admin sarama.ClusterAdmin, name string, partitions int32, replicas int16) error {
	existing, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("list topics: %w", err)
	}
	if _, ok := existing[name]; ok {
		return nil
	}
	retention := "2592000000" // 30 days
	detail := &sarama.TopicDetail{
		NumPartitions:     partitions,
		ReplicationFactor: replicas,
		ConfigEntries: map[string]*string{
			"retention.ms": &retention,
		},
	}
	return admin.CreateTopic(name, detail, false)
}
