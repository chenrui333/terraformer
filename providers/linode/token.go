// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type TokenGenerator struct {
	LinodeService
}

func (g TokenGenerator) createResources(tokenList []linodego.Token) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, token := range tokenList {
		resources = append(resources, terraformutils.NewSimpleResource(
			strconv.Itoa(token.ID),
			strconv.Itoa(token.ID),
			"linode_token",
			"linode",
			[]string{}))
	}
	return resources
}

func (g *TokenGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := client.ListTokens(context.Background(), nil)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
