// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

type GCPService struct { //nolint
	terraformutils.Service
}

func (s *GCPService) applyCustomProviderType(resources []terraformutils.Resource, providerName string) []terraformutils.Resource {
	editedResources := []terraformutils.Resource{}
	for _, r := range resources {
		r.Item["provider"] = providerName
		editedResources = append(editedResources, r)
	}
	return editedResources
}
