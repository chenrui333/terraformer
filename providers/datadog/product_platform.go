// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

func newDatadogIDResource(serviceName, id string, allowEmptyValues []string) (terraformutils.Resource, error) {
	if id == "" {
		return terraformutils.Resource{}, fmt.Errorf("%s missing id", serviceName)
	}

	return terraformutils.NewSimpleResource(
		id,
		fmt.Sprintf("%s_%s", serviceName, id),
		fmt.Sprintf("datadog_%s", serviceName),
		"datadog",
		allowEmptyValues,
	), nil
}

func datadogIDResources(serviceName string, ids []string, allowEmptyValues []string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, id := range ids {
		resource, err := newDatadogIDResource(serviceName, id, allowEmptyValues)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}
