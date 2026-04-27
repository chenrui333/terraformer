// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type VultrService struct { //nolint
	terraformutils.Service
}

func (s *VultrService) generateClient() *govultr.Client {
	return govultr.NewClient(nil, s.Args["api_key"].(string))
}
