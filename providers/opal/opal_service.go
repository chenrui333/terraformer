// SPDX-License-Identifier: Apache-2.0

package opal

import (
	"fmt"
	"net/url"
	"path"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/opalsecurity/opal-go"
)

type OpalService struct { //nolint
	terraformutils.Service
}

func (s *OpalService) newClient() (*opal.APIClient, error) {
	conf := opal.NewConfiguration()

	conf.DefaultHeader["Authorization"] = fmt.Sprintf("Bearer %s", s.GetArgs()["token"].(string))
	u, err := url.Parse(s.GetArgs()["base_url"].(string))
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "/v1")
	conf.Servers = opal.ServerConfigurations{{
		URL: u.String(),
	}}

	return opal.NewAPIClient(conf), nil
}
