// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/cloudfunctions/v2"
	"google.golang.org/api/compute/v1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudFunctionsAllowEmptyValues = []string{""}

var cloudFunctionsAdditionalFields = map[string]interface{}{}

type CloudFunctionsGenerator struct {
	GCPService
}

// Run on CloudFunctionsList and create for each TerraformResource
func (g CloudFunctionsGenerator) createCloudFunctionsResources(ctx context.Context, functionsList *cloudfunctions.ProjectsLocationsFunctionsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := functionsList.Pages(ctx, func(page *cloudfunctions.ListFunctionsResponse) error {
		for _, functions := range page.Functions {
			t := strings.Split(functions.Name, "/")
			if functions.Environment == "GEN_1" {
				name := t[len(t)-1]
				resources = append(resources, terraformutils.NewResource(
					g.GetArgs()["project"].(string)+"/"+g.GetArgs()["region"].(compute.Region).Name+"/"+name,
					g.GetArgs()["region"].(compute.Region).Name+"_"+name,
					"google_cloudfunctions_function",
					g.ProviderName,
					map[string]string{
						"name":     name,
						"project":  g.GetArgs()["project"].(string),
						"location": g.GetArgs()["region"].(compute.Region).Name,
					},
					cloudFunctionsAllowEmptyValues,
					cloudFunctionsAdditionalFields,
				))
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list cloud functions gen1: %w", err)
	}
	return resources, nil
}

func (g CloudFunctionsGenerator) createCloudFunctions2ndGenResources(ctx context.Context, functionsList *cloudfunctions.ProjectsLocationsFunctionsListCall) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := functionsList.Pages(ctx, func(page *cloudfunctions.ListFunctionsResponse) error {
		for _, functions := range page.Functions {
			t := strings.Split(functions.Name, "/")
			if functions.Environment == "GEN_2" {
				name := t[len(t)-1]
				resources = append(resources, terraformutils.NewResource(
					g.GetArgs()["project"].(string)+"/"+g.GetArgs()["region"].(compute.Region).Name+"/"+name,
					g.GetArgs()["region"].(compute.Region).Name+"_"+name,
					"google_cloudfunctions2_function",
					g.ProviderName,
					map[string]string{
						"name":     name,
						"project":  g.GetArgs()["project"].(string),
						"location": g.GetArgs()["region"].(compute.Region).Name,
					},
					cloudFunctionsAllowEmptyValues,
					cloudFunctionsAdditionalFields,
				))
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list cloud functions gen2: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each CloudFunctions create 1 TerraformResource
// Need CloudFunctions name as ID for terraform resource
func (g *CloudFunctionsGenerator) InitResources() error {
	ctx := context.Background()
	cloudfunctionsService, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return err
	}

	functionsList := cloudfunctionsService.Projects.Locations.Functions.List("projects/" + g.GetArgs()["project"].(string) + "/locations/" + g.GetArgs()["region"].(compute.Region).Name)

	functionResources, err := g.createCloudFunctionsResources(ctx, functionsList)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, functionResources...)

	functionsList = cloudfunctionsService.Projects.Locations.Functions.List("projects/" + g.GetArgs()["project"].(string) + "/locations/" + g.GetArgs()["region"].(compute.Region).Name)
	function2Resources, err := g.createCloudFunctions2ndGenResources(ctx, functionsList)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, function2Resources...)

	return nil
}
