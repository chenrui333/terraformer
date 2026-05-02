// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type AccessTokenGenerator struct {
	LaunchDarklyService
}

func getAccessTokens(ctx context.Context, client *ldapi.APIClient) ([]ldapi.Token, error) {
	tokens, err := getAccessTokensWithShowAll(ctx, client, true)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func getAccessTokensWithShowAll(ctx context.Context, client *ldapi.APIClient, showAll bool) ([]ldapi.Token, error) {
	var allTokens []ldapi.Token
	for offset := int64(0); ; offset += pageSize {
		request := client.AccessTokensApi.GetTokens(ctx).
			Limit(pageSize).
			Offset(offset)
		if showAll {
			request = request.ShowAll(true)
		}
		tokens, resp, err := request.Execute()
		closeResponseBody(resp)
		if err != nil {
			if showAll && offset == 0 && resp != nil && resp.StatusCode == http.StatusForbidden {
				return getAccessTokensWithShowAll(ctx, client, false)
			}
			return nil, err
		}
		if tokens == nil {
			break
		}
		items := tokens.GetItems()
		allTokens = append(allTokens, items...)
		if len(items) < pageSize {
			break
		}
	}
	return allTokens, nil
}

func accessTokenResourceName(name, id string) string {
	return resourceNameWithID(name, id)
}

func (g *AccessTokenGenerator) loadAccessTokens(ctx context.Context, client *ldapi.APIClient) error {
	tokens, err := getAccessTokens(ctx, client)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		tokenID := token.GetId()
		resource := terraformutils.NewResource(
			tokenID,
			accessTokenResourceName(token.GetName(), tokenID),
			"launchdarkly_access_token",
			"launchdarkly",
			map[string]string{},
			[]string{},
			map[string]interface{}{})
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *AccessTokenGenerator) InitResources() error {
	return g.loadAccessTokens(g.GetArgs()["ctx"].(context.Context), g.GetArgs()["client"].(*ldapi.APIClient))
}
