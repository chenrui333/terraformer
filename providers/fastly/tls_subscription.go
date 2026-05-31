// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"context"
	"log"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/fastly/go-fastly/v15/fastly"
)

type TLSSubscriptionGenerator struct {
	FastlyService
}

func (g *TLSSubscriptionGenerator) loadTLSSubscriptions(client *fastly.Client) ([]*fastly.TLSSubscription, error) {
	subscriptions, err := client.ListTLSSubscriptions(context.Background(), &fastly.ListTLSSubscriptionsInput{})
	if err != nil {
		return nil, err
	}
	for _, subscription := range subscriptions {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			subscription.ID,
			subscription.ID,
			"fastly_tls_subscription",
			"fastly",
			[]string{}))
	}
	return subscriptions, nil
}

func (g *TLSSubscriptionGenerator) loadTLSActivations(client *fastly.Client) ([]*fastly.TLSActivation, error) {
	activations, err := client.ListTLSActivations(context.Background(), &fastly.ListTLSActivationsInput{})
	if err != nil {
		return nil, err
	}
	for _, activation := range activations {
		log.Println("certicate: ", activation.ID)

		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			activation.ID,
			activation.ID,
			"fastly_tls_activation",
			"fastly",
			[]string{},
		))
	}
	return activations, nil
}

func (g *TLSSubscriptionGenerator) InitResources() error {
	client, err := fastly.NewClient(g.Args["api_key"].(string))
	if err != nil {
		return err
	}

	if _, err := g.loadTLSSubscriptions(client); err != nil {
		return err
	}

	if _, err := g.loadTLSActivations(client); err != nil {
		return err
	}

	return nil
}
