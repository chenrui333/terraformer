// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	oktaV6 "github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/okta/terraform-provider-okta/sdk"
)

type OktaService struct { //nolint
	terraformutils.Service
}

func (s *OktaService) Client() (context.Context, *oktaV6.APIClient, error) {
	orgName := s.Args["org_name"].(string)
	baseURL := s.Args["base_url"].(string)
	apiToken := s.Args["api_token"].(string)

	orgURL := fmt.Sprintf("https://%v.%v", orgName, baseURL)

	config, err := oktaV6.NewConfiguration(
		oktaV6.WithOrgUrl(orgURL),
		oktaV6.WithToken(apiToken),
	)
	if err != nil {
		return nil, nil, err
	}
	client := oktaV6.NewAPIClient(config)

	return context.Background(), client, nil
}

func (s *OktaService) ClientV6() (context.Context, *oktaV6.APIClient, error) {
	return s.Client()
}

func (s *OktaService) APISupplementClient() (context.Context, *sdk.APISupplement, error) {
	baseURL := s.Args["base_url"].(string)
	orgName := s.Args["org_name"].(string)
	apiToken := s.Args["api_token"].(string)

	orgURL := fmt.Sprintf("https://%v.%v", orgName, baseURL)

	ctx, client, err := sdk.NewClient(
		context.Background(),
		sdk.WithOrgUrl(orgURL),
		sdk.WithToken(apiToken),
	)
	if err != nil {
		return ctx, nil, err
	}

	apiSupplementClient := &sdk.APISupplement{
		RequestExecutor: client.CloneRequestExecutor(),
	}

	return ctx, apiSupplementClient, nil
}
