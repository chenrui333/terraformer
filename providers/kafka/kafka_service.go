// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"errors"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
)

type adminClient interface {
	DescribeTopics(topics []string) ([]*sarama.TopicMetadata, error)
	DescribeConfig(resource sarama.ConfigResource) ([]sarama.ConfigEntry, error)
	ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error)
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
	admin, err := sarama.NewClusterAdmin(config.BootstrapServers, saramaConfig)
	if err != nil {
		return nil, err
	}
	return saramaAdminClient{
		ClusterAdmin: admin,
		config:       saramaConfig,
	}, nil
}

type saramaAdminClient struct {
	sarama.ClusterAdmin
	config *sarama.Config
}

func (c saramaAdminClient) ListAcls(filter sarama.AclFilter) ([]sarama.ResourceAcls, error) {
	request := &sarama.DescribeAclsRequest{AclFilter: filter}
	if c.config != nil && c.config.Version.IsAtLeast(sarama.V2_0_0_0) {
		request.Version = 1
	}

	broker, err := c.Controller()
	if err != nil {
		return nil, err
	}
	response, err := broker.DescribeAcls(request)
	if err != nil {
		return nil, err
	}
	return resourceACLsFromDescribeResponse(response)
}

func resourceACLsFromDescribeResponse(response *sarama.DescribeAclsResponse) ([]sarama.ResourceAcls, error) {
	if response == nil {
		return nil, errors.New("kafka acl: empty describe ACLs response")
	}
	if response.Err != sarama.ErrNoError {
		if response.ErrMsg != nil && *response.ErrMsg != "" {
			return nil, fmt.Errorf("kafka acl: describe ACLs failed: %w: %s", response.Err, *response.ErrMsg)
		}
		return nil, fmt.Errorf("kafka acl: describe ACLs failed: %w", response.Err)
	}

	resources := make([]sarama.ResourceAcls, 0, len(response.ResourceAcls))
	for _, resourceACLs := range response.ResourceAcls {
		if resourceACLs == nil {
			continue
		}
		resources = append(resources, *resourceACLs)
	}
	return resources, nil
}

func (s *Service) admin(config Config) (adminClient, error) {
	if s.newAdmin != nil {
		return s.newAdmin(config)
	}
	return defaultAdminFactory(config)
}
