// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"io"
	"net/http"

	"github.com/chenrui333/terraformer/terraformutils"
)

type RBTService struct {
	terraformutils.Service
}

func (s *RBTService) generateRequest(uri string) ([]byte, error) {
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", s.Args["endpoint"].(string)+uri, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.Args["username"].(string), s.Args["password"].(string))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
