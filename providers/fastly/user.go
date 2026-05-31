// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/fastly/go-fastly/v15/fastly"
)

type UserGenerator struct {
	FastlyService
}

func (g *UserGenerator) loadUsers(client *fastly.Client, customerID string) error {
	users, err := client.ListCustomerUsers(context.Background(), &fastly.ListCustomerUsersInput{CustomerID: customerID})
	if err != nil {
		return err
	}
	for _, user := range users {
		userID := fastlyStringValue(user.UserID)
		if userID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			userID,
			userID,
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
