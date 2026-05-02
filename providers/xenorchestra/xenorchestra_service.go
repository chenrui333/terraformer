// SPDX-License-Identifier: Apache-2.0
package xenorchestra

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-xenorchestra/client"
)

type XenorchestraService struct { //nolint
	terraformutils.Service
}

func (m *XenorchestraService) generateClient() (*client.Client, error) {
	config := client.Config{
		Url:      m.Args["url"].(string),
		Username: m.Args["username"].(string),
		Password: m.Args["password"].(string),
	}
	return client.NewClient(config)
}
