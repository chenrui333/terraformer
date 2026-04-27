// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type UserGenerator struct {
	PagerDutyService
}

func (g *UserGenerator) createUserResources(client *pagerduty.Client) error {
	var offset = 0
	options := pagerduty.ListUsersOptions{}
	for {
		options.Offset = offset
		resp, _, err := client.Users.List(&options)
		if err != nil {
			return err
		}

		for _, user := range resp.Users {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				user.ID,
				fmt.Sprintf("user_%s", user.ID),
				"pagerduty_user",
				g.ProviderName,
				[]string{},
			))
		}

		if !resp.More {
			break
		}

		offset += resp.Limit
	}

	return nil
}

func (g *UserGenerator) InitResources() error {
	client, err := g.Client()
	if err != nil {
		return err
	}

	funcs := []func(*pagerduty.Client) error{
		g.createUserResources,
	}

	for _, f := range funcs {
		err := f(client)
		if err != nil {
			return err
		}
	}

	return nil
}
