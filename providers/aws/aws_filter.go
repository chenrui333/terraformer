// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
)

func awsTypedIDFilterValues(filters []terraformutils.ResourceFilter, serviceName string) map[string]bool {
	serviceName = strings.TrimPrefix(serviceName, "aws_")
	values := map[string]bool{}
	for _, filter := range filters {
		if filter.FieldPath != "id" || filter.ServiceName != serviceName || len(filter.AcceptableValues) == 0 {
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

func awsIDFilterAllows(values map[string]bool, value string) bool {
	return len(values) == 0 || values[value]
}
