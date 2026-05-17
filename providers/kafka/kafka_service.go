// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
)

type adminClient interface {
	ListTopics() (map[string]sarama.TopicDetail, error)
	DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error)
	DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error)
	Close() error
}

type adminFactory func(Config) (adminClient, error)

type Service struct {
	terraformutils.Service
	newAdmin adminFactory
}

func defaultAdminFactory(config Config) (adminClient, error) {
	saramaConfig, err := config.newSaramaConfig()
	if err != nil {
		return nil, err
	}
	return sarama.NewClusterAdmin(config.BootstrapServers, saramaConfig)
}

func (s *Service) admin(config Config) (adminClient, error) {
	if s.newAdmin != nil {
		return s.newAdmin(config)
	}
	return defaultAdminFactory(config)
}
