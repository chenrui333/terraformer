// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// ServiceGenerator ...
type ServiceGenerator struct {
	MackerelService
}

func (g *ServiceGenerator) createResources(services []*mackerel.Service) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, service := range services {
		resources = append(resources, g.createResource(service.Name))
	}
	return resources
}

func (g *ServiceGenerator) createResource(serviceName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		serviceName,
		fmt.Sprintf("service_%s", serviceName),
		"mackerel_service",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each service create 1 TerraformResource.
// Need Service Name as ID for terraform resource
func (g *ServiceGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	services, err := client.FindServices()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(services)...)
	return nil
}
