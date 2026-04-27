// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// InstanceTemplateGenerator ...
type InstanceTemplateGenerator struct {
	IBMService
}

func (g InstanceTemplateGenerator) createInstanceTemplateResources(templateID, templateName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		templateID,
		normalizeResourceName(templateName, false),
		"ibm_is_instance_template",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *InstanceTemplateGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("no API key set")
	}

	isURL := GetVPCEndPoint(region)
	iamURL := GetAuthEndPoint()
	vpcoptions := &vpcv1.VpcV1Options{
		URL: isURL,
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
			URL:    iamURL,
		},
	}
	vpcclient, err := vpcv1.NewVpcV1(vpcoptions)
	if err != nil {
		return err
	}
	options := &vpcv1.ListInstanceTemplatesOptions{}
	templates, response, err := vpcclient.ListInstanceTemplates(options)
	if err != nil {
		return fmt.Errorf("error fetching Instance Templates %w\n%s", err, response)
	}

	for _, template := range templates.Templates {
		instemp := template.(*vpcv1.InstanceTemplate)
		g.Resources = append(g.Resources, g.createInstanceTemplateResources(*instemp.ID, *instemp.Name))
	}
	return nil
}
