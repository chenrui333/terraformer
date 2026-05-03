// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	"github.com/zclconf/go-cty/cty"
)

type TencentCloudProvider struct { //nolint
	terraformutils.Provider
	region     string
	credential common.Credential
}

func (p *TencentCloudProvider) getCredential() error {
	secretID := os.Getenv("TENCENTCLOUD_SECRET_ID")
	if secretID == "" {
		return errors.New("TENCENTCLOUD_SECRET_ID must be set")
	}
	secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY")
	if secretKey == "" {
		return errors.New("TENCENTCLOUD_SECRET_KEY must be set")
	}
	token := os.Getenv("TENCENTCLOUD_SECURITY_TOKEN")

	p.credential = common.Credential{
		SecretId:  secretID,
		SecretKey: secretKey,
		Token:     token,
	}
	return nil
}

func (p *TencentCloudProvider) GetConfig() cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"region": cty.StringVal(p.region),
	})
}

func (p *TencentCloudProvider) GetName() string {
	return "tencentcloud"
}

func (p *TencentCloudProvider) GetSource() string {
	return "tencentcloudstack/" + p.GetName()
}

func (p *TencentCloudProvider) Init(args []string) error {
	if len(args) < 1 {
		return errors.New("tencentcloud: expected 1 init arg (region)")
	}

	err := p.getCredential()
	if err != nil {
		return err
	}
	p.region = args[0]
	return nil
}

func (p *TencentCloudProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("tencentcloud: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"region":     p.region,
		"credential": p.credential,
	})
	return nil
}

func (p *TencentCloudProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"cvm":            &CvmGenerator{},
		"vpc":            &VpcGenerator{},
		"cdn":            &CdnGenerator{},
		"as":             &AsGenerator{},
		"clb":            &ClbGenerator{},
		"cos":            &CosGenerator{},
		"key_pair":       &KeyPairGenerator{},
		"security_group": &SecurityGroupGenerator{},
		"cbs":            &CbsGenerator{},
		"cfs":            &CfsGenerator{},
		"elasticsearch":  &EsGenerator{},
		"gaap":           &GaapGenerator{},
		"mongodb":        &MongodbGenerator{},
		"mysql":          &MysqlGenerator{},
		"redis":          &RedisGenerator{},
		"ssl":            &SslGenerator{},
		"scf":            &ScfGenerator{},
		"tcaplus":        &TcaplusGenerator{},
		"vpn":            &VpnGenerator{},
		"eip":            &EipGenerator{},
		"subnet":         &SubnetGenerator{},
		"route_table":    &RouteTableGenerator{},
		"nat_gateway":    &NatGatewayGenerator{},
		"acl":            &ACLGenerator{},
		"pts":            &PtsGenerator{},
		"tat":            &TatGenerator{},
		"dnspod":         &DnspodGenerator{},
		"ses":            &SesGenerator{},
	}
}

func (p *TencentCloudProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"cvm": {
			"vpc":            []string{"vpc_id", "id"},
			"subnet":         []string{"subnet_id", "id"},
			"security_group": []string{"security_groups", "id"},
			"key_pair":       []string{"key_name", "id"},
		},
		"as": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_ids", "id"},
			"clb":    []string{"forward_balancer_ids", "id"},
		},
		"clb": {
			"vpc":            []string{"vpc_id", "id", "target_region_info_vpc_id", "id"},
			"subnet":         []string{"subnet_id", "id"},
			"security_group": []string{"security_groups", "id"},
			"cvm":            []string{"targets.instance_id", "id"},
		},
		"cfs": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"elasticsearch": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"mongodb": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"mysql": {
			"vpc":            []string{"vpc_id", "id"},
			"subnet":         []string{"subnet_id", "id"},
			"security_group": []string{"security_groups", "id"},
		},
		"redis": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"scf": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
			"cos":    []string{"cos_bucket_name", "id"},
		},
		"tcaplus": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"vpn": {
			"vpc": []string{"vpc_id", "id"},
		},
		"subnet": {
			"vpc":         []string{"vpc_id", "id"},
			"route_table": []string{"route_table_id", "id"},
		},
		"route_table": {
			"vpc":         []string{"vpc_id", "id"},
			"route_table": []string{"route_table_id", "id"},
			"nat_gateway": []string{"next_hub", "id"},
			"vpn":         []string{"next_hub", "id"},
		},
		"nat_gateway": {
			"vpc": []string{"vpc_id", "id"},
		},
		"acl": {
			"vpc":    []string{"vpc_id", "id"},
			"subnet": []string{"subnet_id", "id"},
		},
		"eip": {
			"cvm": []string{"instance_id", "id"},
		},
		"cbs": {
			"cvm": []string{"instance_id", "id"},
		},
	}
}

func (p *TencentCloudProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			p.GetName(): map[string]interface{}{
				"region": p.region,
			},
		},
	}
}

func NewTencentCloudClientProfile() *profile.ClientProfile {
	cpf := profile.NewClientProfile()

	// all request use method POST
	cpf.HttpProfile.ReqMethod = "POST"
	// request timeout
	cpf.HttpProfile.ReqTimeout = 300
	// default language
	cpf.Language = "en-US"

	return cpf
}
