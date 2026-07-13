// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
)

func (s *AWSService) ParseFilters(rawFilters []string) error {
	if err := s.Service.ParseFilters(rawFilters); err != nil {
		return err
	}
	normalizeAWSResourceFilters(s.Filter)
	return nil
}

func (s *AWSService) ParseFilter(rawFilter string) ([]terraformutils.ResourceFilter, error) {
	filters, err := s.Service.ParseFilter(rawFilter)
	if err != nil {
		return nil, err
	}
	normalizeAWSResourceFilters(filters)
	return filters, nil
}

func shouldLoadAWSResourceForTypedFilters(filters []terraformutils.ResourceFilter, serviceNames ...string) bool {
	hasTypedFilter := false
	for _, filter := range filters {
		if filter.ServiceName == "" {
			continue
		}
		hasTypedFilter = true
		filterServiceName := normalizeAWSFilterServiceName(filter.ServiceName)
		for _, serviceName := range serviceNames {
			if filterServiceName == normalizeAWSFilterServiceName(serviceName) {
				return true
			}
		}
	}
	return !hasTypedFilter
}

func normalizeAWSResourceFilters(filters []terraformutils.ResourceFilter) {
	for i := range filters {
		filters[i].ServiceName = normalizeAWSFilterServiceName(filters[i].ServiceName)
	}
}

func normalizeAWSFilterServiceName(serviceName string) string {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	if serviceName == "transit_gateway" {
		return "ec2_transit_gateway"
	}
	return serviceName
}
