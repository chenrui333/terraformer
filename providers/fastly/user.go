// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/fastly/go-fastly/v7/fastly"
)

type UserGenerator struct {
	FastlyService
}

func (g *UserGenerator) loadUsers(client *fastly.Client, customerID string) error {
	users, err := client.ListCustomerUsers(&fastly.ListCustomerUsersInput{CustomerID: customerID})
	if err != nil {
		return err
	}
	for _, user := range users {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			user.ID,
			user.ID,
			"fastly_user_v1",
			"fastly",
			[]string{}))
	}
	return nil
}

func (g *UserGenerator) InitResources() error {
	client, err := fastly.NewClient(g.Args["api_key"].(string))
	if err != nil {
		return err
	}

	if err := g.loadUsers(client, g.Args["customer_id"].(string)); err != nil {
		return err
	}

	return nil
}
