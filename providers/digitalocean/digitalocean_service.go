// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type DigitalOceanService struct { //nolint
	terraformutils.Service
}

func (s *DigitalOceanService) generateClient() *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: s.Args["token"].(string),
	}
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)
	return client
}
