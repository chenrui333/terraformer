// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"github.com/chenrui333/terraformer/providers/commercetools/connectivity"
	"github.com/chenrui333/terraformer/terraformutils"
)

type CommercetoolsService struct { //nolint
	terraformutils.Service
}

func (s *CommercetoolsService) newClient() (*connectivity.Client, error) {
	cfg := connectivity.Config{
		ClientID:     s.GetArgs()["client_id"].(string),
		ClientSecret: s.GetArgs()["client_secret"].(string),
		ClientScope:  s.GetArgs()["client_scope"].(string),
		ProjectKey:   s.GetArgs()["project_key"].(string),
		TokenURL:     s.GetArgs()["token_url"].(string) + "/oauth/token",
		BaseURL:      s.GetArgs()["base_url"].(string),
	}

	return cfg.NewClient()
}
