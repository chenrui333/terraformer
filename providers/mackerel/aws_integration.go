// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// AWSIntegrationGenerator ...
type AWSIntegrationGenerator struct {
	MackerelService
}

func (g *AWSIntegrationGenerator) createResources(awsIntegrations []*mackerel.AWSIntegration) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, awsIntegration := range awsIntegrations {
		resources = append(resources, g.createResource(awsIntegration.ID))
	}
	return resources
}

func (g *AWSIntegrationGenerator) createResource(awsIntegrationID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		awsIntegrationID,
		fmt.Sprintf("aws_integration_%s", awsIntegrationID),
		"mackerel_aws_integration",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each aws integration create 1 TerraformResource.
// Need AWS Integration ID as ID for terraform resource
func (g *AWSIntegrationGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	awsIntegrations, err := client.FindAWSIntegrations()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(awsIntegrations)...)
	return nil
}
