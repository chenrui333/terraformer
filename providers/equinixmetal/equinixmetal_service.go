// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/packethost/packngo"
)

type EquinixMetalService struct { //nolint
	terraformutils.Service
}

func (s *EquinixMetalService) generateClient() *packngo.Client {
	client, _ := packngo.NewClient()
	return client
}
