// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"reflect"
	"strings"
	"testing"

	"github.com/IBM/sarama"
)

type mockAdmin struct {
	topics          map[string]sarama.TopicDetail
	metadata        map[string]*sarama.TopicMetadata
	configs         map[string][]sarama.ConfigEntry
	describeConfigs []sarama.ConfigResource
	closed          bool
}

func (m *mockAdmin) ListTopics() (map[string]sarama.TopicDetail, error) {
	return m.topics, nil
}

func (m *mockAdmin) DescribeTopics(names []string) ([]*sarama.TopicMetadata, error) {
	metadata := make([]*sarama.TopicMetadata, 0, len(names))
	for _, name := range names {
		if topicMetadata := m.metadata[name]; topicMetadata != nil {
			metadata = append(metadata, topicMetadata)
		}
	}
	return metadata, nil
}

func (m *mockAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	m.describeConfigs = append(m.describeConfigs, resource)
	return m.configs[resource.Name], nil
}

func (m *mockAdmin) Close() error {
	m.closed = true
	return nil
}

func TestTopicInitResourcesConstructsResources(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"payments.events": {
				NumPartitions:     12,
				ReplicationFactor: 3,
			},
		},
		configs: map[string][]sarama.ConfigEntry{
			"payments.events": {
				{Name: "retention.ms", Value: "604800000", Source: sarama.SourceTopic},
				{Name: "cleanup.policy", Value: "delete", Source: sarama.SourceDefault, Default: true},
				{Name: "sasl.password", Value: "secret", Source: sarama.SourceTopic},
			},
		},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if !admin.closed {
		t.Fatal("admin client was not closed")
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}

	resource := generator.Resources[0]
	if resource.InstanceState.ID != "payments.events" {
		t.Fatalf("resource ID = %q, want topic name", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != "kafka_topic" {
		t.Fatalf("resource type = %q, want kafka_topic", resource.InstanceInfo.Type)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "payments.events" {
		t.Fatalf("name = %q, want payments.events", got)
	}
	if got := resource.InstanceState.Attributes["partitions"]; got != "12" {
		t.Fatalf("partitions = %q, want 12", got)
	}
	if got := resource.InstanceState.Attributes["replication_factor"]; got != "3" {
		t.Fatalf("replication_factor = %q, want 3", got)
	}
	if got := resource.InstanceState.Attributes["config.retention.ms"]; got != "604800000" {
		t.Fatalf("config.retention.ms = %q, want 604800000", got)
	}
	if _, ok := resource.InstanceState.Attributes["config.sasl.password"]; ok {
		t.Fatal("secret-looking topic config was exported")
	}
	if _, ok := resource.InstanceState.Attributes["config.cleanup.policy"]; ok {
		t.Fatal("default topic config was exported")
	}
	if resource.ResourceName == "tfer--topic_payments.events" {
		t.Fatalf("resource name was not normalized for punctuation: %q", resource.ResourceName)
	}
}

func TestTopicInitResourcesEmptyList(t *testing.T) {
	admin := &mockAdmin{topics: map[string]sarama.TopicDetail{}}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("resources len = %d, want 0", len(generator.Resources))
	}
}

func TestTopicFallsBackToMetadataShape(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     -1,
				ReplicationFactor: -1,
			},
		},
		metadata: map[string]*sarama.TopicMetadata{
			"orders": {
				Name: "orders",
				Partitions: []*sarama.PartitionMetadata{
					{ID: 0, Replicas: []int32{1, 2}},
					{ID: 1, Replicas: []int32{2, 3}},
				},
			},
		},
		configs: map[string][]sarama.ConfigEntry{"orders": nil},
	}
	generator := &TopicGenerator{}
	topic, err := generator.topicFromDetail(admin, "orders", admin.topics["orders"], Config{
		BootstrapServers: []string{"broker1.example.com:9092"},
	})
	if err != nil {
		t.Fatalf("topicFromDetail() error = %v", err)
	}
	if topic.Partitions != 2 {
		t.Fatalf("partitions = %d, want 2", topic.Partitions)
	}
	if topic.ReplicationFactor != 2 {
		t.Fatalf("replication factor = %d, want 2", topic.ReplicationFactor)
	}
}

func TestTopicIDFilterAliases(t *testing.T) {
	generator := &TopicGenerator{}
	generator.ParseFilters([]string{"topics=orders:payments.events"})
	wantValues := []string{"orders", "payments.events"}
	if len(generator.Filter) != 1 {
		t.Fatalf("filter len = %d, want 1", len(generator.Filter))
	}
	if generator.Filter[0].ServiceName != "topic" {
		t.Fatalf("filter service = %q, want topic", generator.Filter[0].ServiceName)
	}
	if generator.Filter[0].FieldPath != "id" {
		t.Fatalf("filter field = %q, want id", generator.Filter[0].FieldPath)
	}
	if !reflect.DeepEqual(generator.Filter[0].AcceptableValues, wantValues) {
		t.Fatalf("filter values = %#v, want %#v", generator.Filter[0].AcceptableValues, wantValues)
	}

	generator.Resources = generator.createResources([]Topic{
		{Name: "orders", Partitions: 3, ReplicationFactor: 2},
		{Name: "audit", Partitions: 1, ReplicationFactor: 1},
	})
	generator.InitialCleanup()
	if len(generator.Resources) != 1 {
		t.Fatalf("filtered resources len = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "orders" {
		t.Fatalf("remaining resource ID = %q, want orders", generator.Resources[0].InstanceState.ID)
	}
}

func TestKafkaTopicResourceNameIsStableForPunctuation(t *testing.T) {
	first := kafkaTopicResourceName("a.b_c-d/slash")
	second := kafkaTopicResourceName("a.b_c-d/slash")
	if first != second {
		t.Fatalf("resource names are not stable: %q != %q", first, second)
	}
	for _, disallowed := range []string{".", "/"} {
		if strings.Contains(first, disallowed) {
			t.Fatalf("resource name = %q contains %q", first, disallowed)
		}
	}
}

func TestTopicConfigSkipsMSKServerlessSegmentBytes(t *testing.T) {
	admin := &mockAdmin{
		configs: map[string][]sarama.ConfigEntry{
			"orders": {
				{Name: "segment.bytes", Value: "104857600", Source: sarama.SourceTopic},
				{Name: "retention.ms", Value: "604800000", Source: sarama.SourceTopic},
			},
		},
	}
	config, err := topicConfig(admin, "orders", Config{
		BootstrapServers: []string{"boot-abcd.c1.kafka-serverless.us-east-1.amazonaws.com:9098"},
	})
	if err != nil {
		t.Fatalf("topicConfig() error = %v", err)
	}
	if _, ok := config["segment.bytes"]; ok {
		t.Fatal("MSK Serverless segment.bytes config was exported")
	}
	if got := config["retention.ms"]; got != "604800000" {
		t.Fatalf("retention.ms = %q, want 604800000", got)
	}
}

func TestTopicConfigKeepsSegmentBytesOutsideMSKServerless(t *testing.T) {
	admin := &mockAdmin{
		configs: map[string][]sarama.ConfigEntry{
			"orders": {
				{Name: "segment.bytes", Value: "104857600", Source: sarama.SourceTopic},
			},
		},
	}
	config, err := topicConfig(admin, "orders", Config{
		BootstrapServers: []string{"broker1.example.com:9092"},
	})
	if err != nil {
		t.Fatalf("topicConfig() error = %v", err)
	}
	if got := config["segment.bytes"]; got != "104857600" {
		t.Fatalf("segment.bytes = %q, want 104857600", got)
	}
}
