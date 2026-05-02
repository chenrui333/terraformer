// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v2/okta"
)

type AuthorizationServerGenerator struct {
	OktaService
}

func (g AuthorizationServerGenerator) createResources(authorizationServerList []*okta.AuthorizationServer) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, authorizationServer := range authorizationServerList {
		resourceType := "okta_auth_server"
		if authorizationServer.Name == "default" {
			resourceType = "okta_auth_server_default"
		}

		resources = append(resources, terraformutils.NewSimpleResource(
			authorizationServer.Id,
			"auth_server_"+authorizationServer.Name,
			resourceType,
			"okta",
			[]string{}))
	}
	return resources
}

func (g *AuthorizationServerGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, err := getAuthorizationServers(ctx, client)
	if err != nil {
		return err
	}

	g.Resources = g.createResources(output)
	return nil
}

func getAuthorizationServers(ctx context.Context, client *okta.Client) ([]*okta.AuthorizationServer, error) {
	output, resp, err := client.AuthorizationServer.ListAuthorizationServers(ctx, nil)
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextAuthorizationServerSet []*okta.AuthorizationServer
		resp, err = resp.Next(ctx, &nextAuthorizationServerSet)
		if err != nil {
			return nil, err
		}
		output = append(output, nextAuthorizationServerSet...)
	}

	return output, nil
}
