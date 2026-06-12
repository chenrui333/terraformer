// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type LinodeService struct { //nolint
	terraformutils.Service
}

func (s *LinodeService) generateClient() (linodego.Client, error) {
	linodeClient, err := linodego.NewClient(nil)
	if err != nil {
		return linodeClient, err
	}
	linodeClient.SetToken(s.Args["token"].(string))
	linodeClient.SetDebug(s.Verbose)
	return linodeClient, nil
}
