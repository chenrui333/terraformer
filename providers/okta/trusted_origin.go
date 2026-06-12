// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type TrustedOriginGenerator struct {
	OktaService
}

func (g TrustedOriginGenerator) createResources(trustedOriginList []okta.TrustedOrigin) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, trustedOrigin := range trustedOriginList {
		resources = append(resources, terraformutils.NewSimpleResource(
			trustedOrigin.GetId(),
			"trusted_origin_"+trustedOrigin.GetId(),
			"okta_trusted_origin",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *TrustedOriginGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.TrustedOriginAPI.ListTrustedOrigins(ctx).Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextTrustedOriginSet []okta.TrustedOrigin
		resp, err = resp.Next(&nextTrustedOriginSet)
		if err != nil {
			return err
		}
		output = append(output, nextTrustedOriginSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
