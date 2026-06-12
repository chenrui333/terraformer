// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"fmt"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

type LinodeService struct { //nolint
	terraformutils.Service
}

func (s *LinodeService) generateClient() (linodego.Client, error) {
	token, ok := s.Args["token"].(string)
	if !ok || token == "" {
		return linodego.Client{}, fmt.Errorf("linode: token arg is missing or not a string")
	}

	linodeClient, err := linodego.NewClient(newLinodeHTTPClient())
	if err != nil {
		return linodeClient, err
	}
	linodeClient.SetToken(token)
	linodeClient.SetDebug(s.Verbose)
	return linodeClient, nil
}

func newLinodeHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	}
	return &http.Client{Transport: transport.Clone()}
}
