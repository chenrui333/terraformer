// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

type VultrService struct { //nolint
	terraformutils.Service
}

func (s *VultrService) generateClient() (*govultr.Client, error) {
	apiKey, ok := s.Args["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, errors.New("vultr: api_key arg is missing or not a string")
	}

	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})
	return govultr.NewClient(oauth2.NewClient(ctx, tokenSource)), nil
}
