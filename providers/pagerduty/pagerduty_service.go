// SPDX-License-Identifier: Apache-2.0

package pagerduty

import (
	"github.com/chenrui333/terraformer/terraformutils"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
)

type PagerDutyService struct { //nolint
	terraformutils.Service
}

func (s *PagerDutyService) Client() (*pagerduty.Client, error) {
	client, err := pagerduty.NewClient(&pagerduty.Config{Token: s.GetArgs()["token"].(string)})
	if err != nil {
		return nil, err
	}
	return client, nil
}
