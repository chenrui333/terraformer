package opsgenie

import (
	"errors"
	"os"

	"github.com/zclconf/go-cty/cty"

	"github.com/chenrui333/terraformer/terraformutils"
)

type OpsgenieProvider struct { //nolint
	terraformutils.Provider

	APIKey string
}

func (p *OpsgenieProvider) Init(args []string) error {
	apiKey := os.Getenv("OPSGENIE_API_KEY")
	if len(args) > 0 && args[0] != "" {
		apiKey = args[0]
	}
	p.APIKey = apiKey
	if apiKey == "" {
		return errors.New("required API Key missing")
	}

	return nil
}

func (p *OpsgenieProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service = service
	terraformutils.ConfigureService(p.Service, serviceName, verbose, p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"api-key": p.APIKey,
	})
	return nil
}

func (p *OpsgenieProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"api_key": cty.StringVal(p.APIKey),
	})
}

func (p *OpsgenieProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *OpsgenieProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *OpsgenieProvider) GetName() string {
	return "opsgenie"
}

func (p *OpsgenieProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"user":    &UserGenerator{},
		"team":    &TeamGenerator{},
		"service": &ServiceGenerator{},
	}
}
