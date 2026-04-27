// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
)

type ChannelGenerator struct {
	CommercetoolsService
}

// InitResources generates Terraform Resources from Commercetools API
func (g *ChannelGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return err
	}

	channels, err := client.Project().Channels().Get().Execute(context.Background())
	if err != nil {
		return err
	}
	for _, channel := range channels.Results {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			channel.ID,
			channel.Key,
			"commercetools_channel",
			"commercetools",
			map[string]string{},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
