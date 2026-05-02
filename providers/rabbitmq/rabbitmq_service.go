// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
)

type RBTService struct {
	terraformutils.Service
}

func (s *RBTService) generateRequest(uri string) ([]byte, error) {
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Args["endpoint"].(string)+uri, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.Args["username"].(string), s.Args["password"].(string))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("rabbitmq GET %s failed: %s; reading response body: %w", uri, resp.Status, err)
		}
		return nil, fmt.Errorf("rabbitmq GET %s failed: %s: %s", uri, resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
