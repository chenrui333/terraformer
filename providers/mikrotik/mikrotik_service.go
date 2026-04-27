// SPDX-License-Identifier: Apache-2.0
package mikrotik

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-mikrotik/client"
)

type MikrotikService struct { //nolint
	terraformutils.Service
}

func (m *MikrotikService) generateClient() *client.Mikrotik {
	return client.NewClient(
		client.GetConfigFromEnv(),
	)
}
