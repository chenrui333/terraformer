// SPDX-License-Identifier: Apache-2.0
package xenorchestra

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-xenorchestra/client"
)

type AclGenerator struct { //nolint
	XenorchestraService
}

func (g AclGenerator) createResources(acls []client.Acl) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, acl := range acls {
		resourceName := acl.Id
		resources = append(resources, terraformutils.NewSimpleResource(
			acl.Id,
			resourceName,
			"xenorchestra_acl",
			"xenorchestra",
			[]string{}))
	}
	return resources
}

func (g *AclGenerator) InitResources() error {
	client := g.generateClient()
	acls, err := client.GetAcls()

	if err != nil {
		return err
	}
	g.Resources = g.createResources(acls)
	return nil
}
