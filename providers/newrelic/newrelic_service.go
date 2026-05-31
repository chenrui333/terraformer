// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"github.com/chenrui333/terraformer/terraformutils"
	newrelic "github.com/newrelic/newrelic-client-go/v2/newrelic"
)

type NewRelicService struct { //nolint
	terraformutils.Service
}

func (s *NewRelicService) Client() (*newrelic.NewRelic, error) {
	return newrelic.New(newrelic.ConfigPersonalAPIKey(s.GetArgs()["apiKey"].(string)))
}
