// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type SubscriptionGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *SubscriptionGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	subscriptions, err := client.Project().Subscriptions().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, subscription := range subscriptions.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			subscription.ID,
			stringValue(subscription.Key),
			"commercetools_subscription",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
