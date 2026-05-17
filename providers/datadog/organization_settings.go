// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	OrganizationSettingsAllowEmptyValues = []string{"settings."}
)

type OrganizationSettingsGenerator struct {
	DatadogService
}

func (g *OrganizationSettingsGenerator) createResource(publicID, name string) terraformutils.Resource {
	resourceName := fmt.Sprintf("organization_settings_%s", publicID)
	if name != "" {
		resourceName = fmt.Sprintf("organization_settings_%s", name)
	}

	return terraformutils.NewSimpleResource(
		publicID,
		resourceName,
		"datadog_organization_settings",
		"datadog",
		OrganizationSettingsAllowEmptyValues,
	)
}

func (g *OrganizationSettingsGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV1.NewOrganizationsApi(datadogClient)

	resp, httpResp, err := api.ListOrgs(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	resources := []terraformutils.Resource{}
	for _, org := range resp.GetOrgs() {
		publicID := org.GetPublicId()
		if publicID == "" {
			continue
		}
		resources = append(resources, g.createResource(publicID, org.GetName()))
	}
	g.Resources = resources
	return nil
}
