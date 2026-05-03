// SPDX-License-Identifier: Apache-2.0

package alicloud

import (
	"errors"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

// AliCloudProvider Provider for alicloud
type AliCloudProvider struct { //nolint
	terraformutils.Provider
	region  string
	profile string
}

const GlobalRegion = "alicloud-global"

// GetConfig Converts json config to go-cty
func (p *AliCloudProvider) GetConfig() cty.Value {
	profile := p.profile
	config, err := LoadConfigFromProfile(profile)
	if err != nil {
		fmt.Println("ERROR:", err)
	}

	region := p.region
	if region == "" {
		region = config.RegionID
	}

	var val cty.Value
	if config.RAMRoleArn != "" {
		val = cty.ObjectVal(map[string]cty.Value{
			"region":  cty.StringVal(region),
			"profile": cty.StringVal(profile),
			"assume_role": cty.SetVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"role_arn": cty.StringVal(config.RAMRoleArn),
				}),
			}),
		})
	} else {
		val = cty.ObjectVal(map[string]cty.Value{
			"region":  cty.StringVal(region),
			"profile": cty.StringVal(profile),
		})
	}

	return val
}

// GetResourceConnections Gets resource connections for alicloud
func (p AliCloudProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		// TODO: Not implemented
	}
}

// GetProviderData Used for generated HCL2 for the provider
func (p AliCloudProvider) GetProviderData(_ ...string) map[string]interface{} {
	alicloudConfig := map[string]interface{}{}
	if p.region == GlobalRegion {
		alicloudConfig["region"] = "cn-hangzhou"
	} else {
		alicloudConfig["region"] = p.region
	}
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"alicloud": alicloudConfig,
		},
	}
}

// Init Loads up command line arguments in the provider
func (p *AliCloudProvider) Init(args []string) error {
	if len(args) < 2 {
		return errors.New("alicloud: expected 2 init args (region, profile)")
	}

	p.region = args[0]
	p.profile = args[1]
	return nil
}

// GetName Gets name of provider
func (p *AliCloudProvider) GetName() string {
	return "alicloud"
}

// InitService Initializes the AliCloud service
func (p *AliCloudProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("alicloud: " + serviceName + " not supported service")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"region":  p.region,
		"profile": p.profile,
	})
	return nil
}

// GetSupportedService Gets a list of all supported services
func (p *AliCloudProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"dns":     &DNSGenerator{},
		"ecs":     &EcsGenerator{},
		"keypair": &KeyPairGenerator{},
		"nat":     &NatGatewayGenerator{},
		"pvtz":    &PvtzGenerator{},
		"ram":     &RAMGenerator{},
		"rds":     &RdsGenerator{},
		"sg":      &SgGenerator{},
		"slb":     &SlbGenerator{},
		"vpc":     &VpcGenerator{},
		"vswitch": &VSwitchGenerator{},
	}
}
