// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type ListsGenerator struct {
	CloudflareService
}

func (g *ListsGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	lists, err := api.ListLists(ctx, account, cf.ListListsParams{})
	if err != nil {
		return err
	}
	for _, list := range lists {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			list.ID,
			cloudflareResourceName(account.Identifier, list.Name, list.ID),
			"cloudflare_list",
			"cloudflare",
			map[string]string{"account_id": account.Identifier},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
