// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
)

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

func normalizeAWSFilterServiceName(serviceName string) string {
	return strings.TrimPrefix(serviceName, "aws_")
}
