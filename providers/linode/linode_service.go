// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
	"golang.org/x/oauth2"
)

type LinodeService struct { //nolint
	terraformutils.Service
}

func (s *LinodeService) generateClient() (linodego.Client, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.Args["token"].(string)})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}
	linodeClient, err := linodego.NewClient(oauth2Client)
	if err != nil {
		return linodeClient, err
	}
	linodeClient.SetDebug(s.Verbose)
	return linodeClient, nil
}
