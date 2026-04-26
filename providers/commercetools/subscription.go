// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
