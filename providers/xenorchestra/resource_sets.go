// SPDX-License-Identifier: Apache-2.0
package xenorchestra

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-xenorchestra/client"
)

type ResourceSetGenerator struct {
	XenorchestraService
}

func (g ResourceSetGenerator) createResources(acls []client.ResourceSet) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, acl := range acls {
		resourceName := acl.Id
		resources = append(resources, terraformutils.NewSimpleResource(
			acl.Id,
			resourceName,
			"xenorchestra_resource_set",
			"xenorchestra",
			[]string{}))
	}
	return resources
}

func (g *ResourceSetGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	acls, err := client.GetResourceSets()

	if err != nil {
		return err
	}
	g.Resources = g.createResources(acls)
	return nil
}
