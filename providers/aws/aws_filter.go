// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
)

func awsTypedIDFilterValues(filters []terraformutils.ResourceFilter, serviceName string) map[string]bool {
	return awsTypedFilterValues(filters, serviceName, "id")
}

func awsTypedFilterValues(filters []terraformutils.ResourceFilter, serviceName, fieldPath string) map[string]bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	values := map[string]bool{}
	for _, filter := range filters {
		if filter.FieldPath != fieldPath || filter.ServiceName != serviceName || len(filter.AcceptableValues) == 0 {
			continue
		}
		for _, value := range filter.AcceptableValues {
			if value != "" {
				values[value] = true
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func awsHasTypedFilter(filters []terraformutils.ResourceFilter, serviceName string) bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	for _, filter := range filters {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func awsHasTypedNonIDFilter(filters []terraformutils.ResourceFilter, serviceName string) bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	for _, filter := range filters {
		if filter.ServiceName == serviceName && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func awsHasApplicableNonIDFilter(filters []terraformutils.ResourceFilter, serviceName string) bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	for _, filter := range filters {
		if filter.FieldPath != "id" && filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func awsHasApplicableFilter(filters []terraformutils.ResourceFilter, serviceName string) bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	for _, filter := range filters {
		if filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func awsIDFilterAllows(values map[string]bool, value string) bool {
	return len(values) == 0 || values[value]
}

func awsMergeIDFilterValues(filters ...map[string]bool) map[string]bool {
	values := map[string]bool{}
	for _, filter := range filters {
		for value := range filter {
			values[value] = true
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
