// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	DomainAllowlistAllowEmptyValues = []string{}
)

type DomainAllowlistGenerator struct {
	DatadogService
}

func (g *DomainAllowlistGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewDomainAllowlistApi(datadogClient)

	resp, httpResp, err := api.GetDomainAllowlist(auth)
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
	if err != nil {
		return err
	}

	data := resp.GetData()
	id := data.GetId()
	if id == "" {
		return nil
	}

	g.Resources = []terraformutils.Resource{
		terraformutils.NewSimpleResource(
			id,
			"domain_allowlist",
			"datadog_domain_allowlist",
			"datadog",
			DomainAllowlistAllowEmptyValues,
		),
	}
	return nil
}
