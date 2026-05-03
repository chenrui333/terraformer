// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
	"os"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type NewRelicProvider struct { //nolint
	terraformutils.Provider
	accountID int
	APIKey    string
	Region    string
}

func (p *NewRelicProvider) Init(args []string) error {
	apiKey := os.Getenv("NEW_RELIC_API_KEY")
	accountID := 0
	accountIDSet := false
	region := "US"

	if len(args) > 0 && args[0] != "" {
		apiKey = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		parsedAccountID, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		accountID = parsedAccountID
		accountIDSet = true
	} else if accountIDs := os.Getenv("NEW_RELIC_ACCOUNT_ID"); accountIDs != "" {
		parsedAccountID, err := strconv.Atoi(accountIDs)
		if err != nil {
			return err
		}
		accountID = parsedAccountID
		accountIDSet = true
	}
	if len(args) > 2 && args[2] != "" {
		region = args[2]
	}
	p.APIKey = apiKey
	p.accountID = accountID
	p.Region = region
	if !accountIDSet {
		return errors.New("newrelic: account id is required")
	}
	if accountID <= 0 {
		return errors.New("newrelic: account id must be greater than 0")
	}
	return nil
}

func (p *NewRelicProvider) GetName() string {
	return "newrelic"
}

func (p *NewRelicProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"account_id": cty.NumberIntVal(int64(p.accountID)),
		"api_key":    cty.StringVal(p.APIKey),
		"region":     cty.StringVal(p.Region),
	})
}

func (p *NewRelicProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (NewRelicProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *NewRelicProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"alert":           &AlertGenerator{},
		"alert_channel":   &AlertChannelGenerator{},
		"alert_condition": &AlertConditionGenerator{},
		"alert_policy":    &AlertPolicyGenerator{},
		"infra":           &InfraGenerator{},
		"synthetics":      &SyntheticsGenerator{},
		"tags":            &TagsGenerator{},
	}
}

func (p *NewRelicProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("newrelic: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{"apiKey": p.APIKey})

	return nil
}
