// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"errors"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type mockAdmin struct {
	topics          map[string]sarama.TopicDetail
	metadata        map[string]*sarama.TopicMetadata
	configs         map[string][]sarama.ConfigEntry
	configErrors    map[string]error
	describeTopics  [][]string
	describeConfigs []sarama.ConfigResource
	closed          bool
}

func (m *mockAdmin) DescribeTopics(names []string) ([]*sarama.TopicMetadata, error) {
	m.describeTopics = append(m.describeTopics, append([]string(nil), names...))
	if len(names) == 0 {
		names = m.topicNames()
	}
	metadata := make([]*sarama.TopicMetadata, 0, len(names))
	for _, name := range names {
		if topicMetadata := m.metadata[name]; topicMetadata != nil {
			metadata = append(metadata, topicMetadata)
			continue
		}
		if detail, ok := m.topics[name]; ok {
			topicMetadata, err := topicMetadataFromDetail(name, detail)
			if err != nil {
				return nil, err
			}
			metadata = append(metadata, topicMetadata)
		}
	}
	return metadata, nil
}

func (m *mockAdmin) topicNames() []string {
	seen := map[string]struct{}{}
	names := make([]string, 0, len(m.topics)+len(m.metadata))
	for name := range m.topics {
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range m.metadata {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func topicMetadataFromDetail(name string, detail sarama.TopicDetail) (*sarama.TopicMetadata, error) {
	partitions, replicationFactor, err := topicShapeFromDetail(detail)
	if err != nil {
		return nil, err
	}
	partitionMetadata := []*sarama.PartitionMetadata{}
	if len(detail.ReplicaAssignment) > 0 {
		partitionIDs := make([]int32, 0, len(detail.ReplicaAssignment))
		for partitionID := range detail.ReplicaAssignment {
			partitionIDs = append(partitionIDs, partitionID)
		}
		sort.Slice(partitionIDs, func(i, j int) bool {
			return partitionIDs[i] < partitionIDs[j]
		})
		for _, partitionID := range partitionIDs {
			partitionMetadata = append(partitionMetadata, &sarama.PartitionMetadata{
				ID:       partitionID,
				Replicas: detail.ReplicaAssignment[partitionID],
			})
		}
		return &sarama.TopicMetadata{Name: name, Partitions: partitionMetadata}, nil
	}
	for partitionID := int32(0); partitionID < partitions; partitionID++ {
		replicas := []int32{}
		for replicaID := int16(0); replicaID < replicationFactor; replicaID++ {
			replicas = append(replicas, int32(replicaID)+1)
		}
		partitionMetadata = append(partitionMetadata, &sarama.PartitionMetadata{
			ID:       partitionID,
			Replicas: replicas,
		})
	}
	return &sarama.TopicMetadata{Name: name, Partitions: partitionMetadata}, nil
}

func (m *mockAdmin) DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error) {
	m.describeConfigs = append(m.describeConfigs, resource)
	if err := m.configErrors[resource.Name]; err != nil {
		return nil, err
	}
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

func TestTopicInitResourcesDoesNotRequireConfigsForMetadataListing(t *testing.T) {
	admin := &mockAdmin{
		metadata: map[string]*sarama.TopicMetadata{
			"orders": {
				Name: "orders",
				Partitions: []*sarama.PartitionMetadata{
					{ID: 0, Replicas: []int32{1, 2}},
					{ID: 1, Replicas: []int32{1, 2}},
				},
			},
		},
		configErrors: map[string]error{"orders": sarama.ErrTopicAuthorizationFailed},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(admin.describeTopics) != 1 || admin.describeTopics[0] != nil {
		t.Fatalf("DescribeTopics calls = %#v, want one all-topic metadata call", admin.describeTopics)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	resource := generator.Resources[0]
	if resource.InstanceState.ID != "orders" {
		t.Fatalf("resource ID = %q, want orders", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["partitions"]; got != "2" {
		t.Fatalf("partitions = %q, want 2", got)
	}
	if got := resource.InstanceState.Attributes["replication_factor"]; got != "2" {
		t.Fatalf("replication_factor = %q, want 2", got)
	}
	if _, ok := resource.InstanceState.Attributes["config.%"]; ok {
		t.Fatal("topic config was exported after authorization failure")
	}
}

func TestTopicInitResourcesSkipsUnauthorizedMetadataEntries(t *testing.T) {
	admin := &mockAdmin{
		metadata: map[string]*sarama.TopicMetadata{
			"orders": {
				Name: "orders",
				Partitions: []*sarama.PartitionMetadata{
					{ID: 0, Replicas: []int32{1, 2}},
					{ID: 1, Replicas: []int32{1, 2}},
				},
			},
			"private": {
				Name: "private",
				Err:  sarama.ErrTopicAuthorizationFailed,
			},
		},
		configs:      map[string][]sarama.ConfigEntry{"orders": nil},
		configErrors: map[string]error{"private": errors.New("unauthorized topic config should not be described")},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "orders" {
		t.Fatalf("resource ID = %q, want orders", generator.Resources[0].InstanceState.ID)
	}
	if len(admin.describeConfigs) != 1 || admin.describeConfigs[0].Name != "orders" {
		t.Fatalf("DescribeConfig calls = %#v, want only orders", admin.describeConfigs)
	}
}

func TestTopicInitResourcesSkipsInternalTopicsByDefault(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
			"__consumer_offsets": {
				NumPartitions:     50,
				ReplicationFactor: 3,
			},
		},
		configs: map[string][]sarama.ConfigEntry{
			"orders": nil,
		},
		configErrors: map[string]error{"__consumer_offsets": errors.New("internal topic config should not be described")},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "orders" {
		t.Fatalf("resource ID = %q, want orders", generator.Resources[0].InstanceState.ID)
	}
	if len(admin.describeConfigs) != 1 || admin.describeConfigs[0].Name != "orders" {
		t.Fatalf("DescribeConfig calls = %#v, want only orders", admin.describeConfigs)
	}
}

func TestTopicInitResourcesIncludesExplicitInternalTopicFilter(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
			"__consumer_offsets": {
				NumPartitions:     50,
				ReplicationFactor: 3,
			},
		},
		configs: map[string][]sarama.ConfigEntry{
			"orders":             nil,
			"__consumer_offsets": nil,
		},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.ParseFilters([]string{"topics=__consumer_offsets"})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if !reflect.DeepEqual(admin.describeTopics, [][]string{{"__consumer_offsets"}}) {
		t.Fatalf("DescribeTopics calls = %#v, want only __consumer_offsets", admin.describeTopics)
	}
	generator.InitialCleanup()
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "__consumer_offsets" {
		t.Fatalf("resource ID = %q, want __consumer_offsets", generator.Resources[0].InstanceState.ID)
	}
}

func TestTopicInitResourcesAppliesIDFilterBeforeConfigDescribe(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
			"payments": {
				NumPartitions:     6,
				ReplicationFactor: 3,
			},
		},
		configs:      map[string][]sarama.ConfigEntry{"orders": nil},
		configErrors: map[string]error{"payments": errors.New("unfiltered topic config should not be described")},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.ParseFilters([]string{"topics=orders"})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if !reflect.DeepEqual(admin.describeTopics, [][]string{{"orders"}}) {
		t.Fatalf("DescribeTopics calls = %#v, want only orders", admin.describeTopics)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "orders" {
		t.Fatalf("resource ID = %q, want orders", generator.Resources[0].InstanceState.ID)
	}
	if len(admin.describeConfigs) != 1 || admin.describeConfigs[0].Name != "orders" {
		t.Fatalf("DescribeConfig calls = %#v, want only orders", admin.describeConfigs)
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

func TestTopicConfigAuthorizationErrorIsSkippable(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
		},
		configErrors: map[string]error{"orders": sarama.ErrTopicAuthorizationFailed},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if _, ok := generator.Resources[0].InstanceState.Attributes["config.%"]; ok {
		t.Fatal("topic config was exported after authorization failure")
	}
	for key, want := range map[string]interface{}{
		"name":               "orders",
		"partitions":         3,
		"replication_factor": 2,
	} {
		if got := generator.Resources[0].AdditionalFields[key]; got != want {
			t.Fatalf("AdditionalFields[%q] = %#v, want %#v", key, got, want)
		}
	}
}

func TestTopicPreservesRequiredFieldsAfterImportFallback(t *testing.T) {
	resource := TopicGenerator{}.createResources([]Topic{{
		Name:              "orders",
		Partitions:        3,
		ReplicationFactor: 2,
	}})[0]

	resource.InstanceState.Attributes = map[string]string{"id": "orders"}
	parser := terraformutils.NewFlatmapParser(
		resource.InstanceState.Attributes,
		[]*regexp.Regexp{regexp.MustCompile("^id$")},
		nil,
	)
	if err := resource.ParseTFstate(parser, cty.Object(map[string]cty.Type{
		"id":                 cty.String,
		"name":               cty.String,
		"partitions":         cty.Number,
		"replication_factor": cty.Number,
	})); err != nil {
		t.Fatalf("ParseTFstate() error = %v", err)
	}

	for key, want := range map[string]interface{}{
		"name":               "orders",
		"partitions":         3,
		"replication_factor": 2,
	} {
		if got := resource.Item[key]; got != want {
			t.Fatalf("Item[%q] = %#v, want %#v", key, got, want)
		}
	}
	if _, ok := resource.Item["id"]; ok {
		t.Fatal("id attribute was not filtered from generated config item")
	}
}

func TestTopicConfigUnexpectedErrorFailsImport(t *testing.T) {
	admin := &mockAdmin{
		topics: map[string]sarama.TopicDetail{
			"orders": {
				NumPartitions:     3,
				ReplicationFactor: 2,
			},
		},
		configErrors: map[string]error{"orders": errors.New("broker connection reset")},
	}
	generator := &TopicGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	err := generator.InitResources()
	if err == nil {
		t.Fatal("expected unexpected topic config error")
	}
	if !strings.Contains(err.Error(), "describe config") {
		t.Fatalf("error = %q, want describe config context", err)
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
