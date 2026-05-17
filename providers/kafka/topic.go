// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
)

type TopicGenerator struct {
	Service
}

type Topic struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
	Config            map[string]string
}

var TopicAllowEmptyValues = []string{}

func (g *TopicGenerator) InitResources() error {
	config := g.Args["config"].(Config)
	admin, err := g.admin(config)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := admin.Close(); closeErr != nil {
			log.Printf("kafka: close admin client: %v", closeErr)
		}
	}()

	topics, err := g.listTopics(admin, config)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(topics)
	return nil
}

func (g *TopicGenerator) ParseFilter(rawFilter string) []terraformutils.ResourceFilter {
	normalized := rawFilter
	for _, prefix := range []string{"kafka_topic=", "topics="} {
		if strings.HasPrefix(rawFilter, prefix) {
			normalized = "topic=" + strings.TrimPrefix(rawFilter, prefix)
			break
		}
	}
	return g.Service.ParseFilter(normalized)
}

func (g *TopicGenerator) ParseFilters(rawFilters []string) {
	g.Filter = []terraformutils.ResourceFilter{}
	for _, rawFilter := range rawFilters {
		g.Filter = append(g.Filter, g.ParseFilter(rawFilter)...)
	}
}

func (g *TopicGenerator) listTopics(admin adminClient, config Config) ([]Topic, error) {
	details, err := admin.ListTopics()
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(details))
	for name := range details {
		names = append(names, name)
	}
	sort.Strings(names)

	topics := make([]Topic, 0, len(names))
	for _, name := range names {
		if isInternalTopic(name) && !g.isExplicitlyRequestedTopic(name) {
			continue
		}
		topic, err := g.topicFromDetail(admin, name, details[name], config)
		if err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, nil
}

func isInternalTopic(name string) bool {
	return strings.HasPrefix(name, "__")
}

func (g *TopicGenerator) isExplicitlyRequestedTopic(name string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("topic") {
			continue
		}
		for _, value := range filter.AcceptableValues {
			if value == name {
				return true
			}
		}
	}
	return false
}

func (g *TopicGenerator) topicFromDetail(admin adminClient, name string, detail sarama.TopicDetail, providerConfig Config) (Topic, error) {
	partitions, replicationFactor, err := topicShapeFromDetail(detail)
	if err != nil {
		return Topic{}, fmt.Errorf("kafka topic %q: %w", name, err)
	}
	if partitions == 0 || replicationFactor == 0 {
		metadata, err := admin.DescribeTopics([]string{name})
		if err != nil {
			return Topic{}, fmt.Errorf("kafka topic %q: describe metadata: %w", name, err)
		}
		partitions, replicationFactor, err = topicShapeFromMetadata(name, metadata)
		if err != nil {
			return Topic{}, err
		}
	}
	config, err := topicConfig(admin, name, providerConfig)
	if err != nil {
		log.Printf("kafka: skipping topic config for %q: %v", name, err)
		config = map[string]string{}
	}
	return Topic{
		Name:              name,
		Partitions:        partitions,
		ReplicationFactor: replicationFactor,
		Config:            config,
	}, nil
}

func topicShapeFromDetail(detail sarama.TopicDetail) (int32, int16, error) {
	partitions := detail.NumPartitions
	replicationFactor := detail.ReplicationFactor
	if partitions <= 0 && len(detail.ReplicaAssignment) > 0 {
		converted, err := safePartitionCount(len(detail.ReplicaAssignment))
		if err != nil {
			return 0, 0, err
		}
		partitions = converted
	}
	if replicationFactor <= 0 && len(detail.ReplicaAssignment) > 0 {
		for _, replicas := range detail.ReplicaAssignment {
			if len(replicas) == 0 {
				continue
			}
			if replicationFactor == 0 || replicationFactor == -1 {
				converted, err := safeReplicationFactor(len(replicas))
				if err != nil {
					return 0, 0, err
				}
				replicationFactor = converted
				continue
			}
			converted, err := safeReplicationFactor(len(replicas))
			if err != nil {
				return 0, 0, err
			}
			if replicationFactor != converted {
				return 0, 0, fmt.Errorf("inconsistent replication factor %d != %d", replicationFactor, len(replicas))
			}
		}
	}
	if partitions < 0 {
		partitions = 0
	}
	if replicationFactor < 0 {
		replicationFactor = 0
	}
	return partitions, replicationFactor, nil
}

func topicShapeFromMetadata(name string, topics []*sarama.TopicMetadata) (int32, int16, error) {
	for _, metadata := range topics {
		if metadata == nil || metadata.Name != name {
			continue
		}
		partitions, err := safePartitionCount(len(metadata.Partitions))
		if err != nil {
			return 0, 0, fmt.Errorf("kafka topic %q: %w", name, err)
		}
		var replicationFactor int16
		for _, partition := range metadata.Partitions {
			if partition == nil || len(partition.Replicas) == 0 {
				continue
			}
			if replicationFactor == 0 {
				converted, err := safeReplicationFactor(len(partition.Replicas))
				if err != nil {
					return 0, 0, fmt.Errorf("kafka topic %q: %w", name, err)
				}
				replicationFactor = converted
				continue
			}
			converted, err := safeReplicationFactor(len(partition.Replicas))
			if err != nil {
				return 0, 0, fmt.Errorf("kafka topic %q: %w", name, err)
			}
			if replicationFactor != converted {
				return 0, 0, fmt.Errorf("kafka topic %q: inconsistent replication factor %d != %d", name, replicationFactor, len(partition.Replicas))
			}
		}
		if partitions == 0 || replicationFactor == 0 {
			return 0, 0, fmt.Errorf("kafka topic %q: could not determine partitions and replication factor", name)
		}
		return partitions, replicationFactor, nil
	}
	return 0, 0, fmt.Errorf("kafka topic %q: metadata not found", name)
}

func safePartitionCount(value int) (int32, error) {
	const maxInt32 = 2147483647
	if value > maxInt32 {
		return 0, fmt.Errorf("partition count %d exceeds int32 max", value)
	}
	converted := int32(value) //nolint:gosec // value is bounds checked above.
	return converted, nil
}

func safeReplicationFactor(value int) (int16, error) {
	const maxInt16 = 32767
	if value > maxInt16 {
		return 0, fmt.Errorf("replication factor %d exceeds int16 max", value)
	}
	converted := int16(value) //nolint:gosec // value is bounds checked above.
	return converted, nil
}

func topicConfig(admin adminClient, name string, providerConfig Config) (map[string]string, error) {
	entries, err := admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: name,
	})
	if err != nil {
		return nil, err
	}
	config := map[string]string{}
	for _, entry := range entries {
		if entry.Sensitive || isDefaultConfig(entry) || isForbiddenConfigName(entry.Name) || isUnsupportedTopicConfigName(entry.Name, providerConfig) {
			continue
		}
		config[entry.Name] = entry.Value
	}
	return config, nil
}

func isDefaultConfig(entry sarama.ConfigEntry) bool {
	return entry.Default ||
		entry.Source == sarama.SourceDefault ||
		entry.Source == sarama.SourceStaticBroker ||
		entry.Source == sarama.SourceDynamicDefaultBroker
}

func isForbiddenConfigName(name string) bool {
	lower := strings.ToLower(name)
	for _, needle := range []string{
		"password",
		"private.key",
		"access.key",
		"secret.key",
		"session.token",
		"oauth.token",
		"scram.password",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func isUnsupportedTopicConfigName(name string, providerConfig Config) bool {
	return strings.EqualFold(name, "segment.bytes") && isMSKServerless(providerConfig.BootstrapServers)
}

func isMSKServerless(bootstrapServers []string) bool {
	for _, server := range bootstrapServers {
		if strings.Contains(strings.ToLower(server), "kafka-serverless") {
			return true
		}
	}
	return false
}

func (g TopicGenerator) createResources(topics []Topic) []terraformutils.Resource {
	resources := make([]terraformutils.Resource, 0, len(topics))
	for _, topic := range topics {
		attributes := map[string]string{
			"name":               topic.Name,
			"partitions":         strconv.Itoa(int(topic.Partitions)),
			"replication_factor": strconv.Itoa(int(topic.ReplicationFactor)),
		}
		additionalFields := map[string]interface{}{}
		if len(topic.Config) > 0 {
			attributes["config.%"] = strconv.Itoa(len(topic.Config))
			config := map[string]interface{}{}
			for key, value := range topic.Config {
				attributes["config."+key] = value
				config[key] = value
			}
			additionalFields["config"] = config
		}
		resources = append(resources, terraformutils.NewResource(
			topic.Name,
			kafkaTopicResourceName(topic.Name),
			"kafka_topic",
			"kafka",
			attributes,
			TopicAllowEmptyValues,
			additionalFields,
		))
	}
	return resources
}

func kafkaTopicResourceName(topicName string) string {
	hash := sha256.Sum256([]byte(topicName))
	return "topic_" + normalizeTopicResourceName(topicName) + "_" + hex.EncodeToString(hash[:4])
}

func normalizeTopicResourceName(topicName string) string {
	var builder strings.Builder
	for _, r := range topicName {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			builder.WriteRune(r)
		case unicode.IsSpace(r):
			builder.WriteByte('_')
		default:
			fmt.Fprintf(&builder, "_x%04X_", r)
		}
	}
	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return "topic"
	}
	return name
}
