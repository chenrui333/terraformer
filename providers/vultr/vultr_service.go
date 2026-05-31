// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

type VultrService struct { //nolint
	terraformutils.Service
}

func (s *VultrService) generateClient() *govultr.Client {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.Args["api_key"].(string)})
	return govultr.NewClient(oauth2.NewClient(ctx, tokenSource))
}
