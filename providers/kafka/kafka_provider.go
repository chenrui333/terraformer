// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

type Provider struct {
	terraformutils.Provider
	config Config
}

func (p *Provider) Init(args []string) error {
	config, err := configFromArgs(args)
	if err != nil {
		return err
	}
	if err := config.validate(); err != nil {
		return err
	}
	p.config = config
	return nil
}

func (p *Provider) GetName() string {
	return "kafka"
}

func (p *Provider) GetConfig() cty.Value {
	return cty.ObjectVal(p.safeConfigCTY())
}

func (p *Provider) GetBasicConfig() cty.Value {
	return p.GetConfig()
}

func (p *Provider) GetProviderData(_ ...string) map[string]interface{} {
	providerConfig := p.safeConfigMap()
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"kafka": providerConfig,
		},
	}
}

func (p *Provider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"config": p.config,
	})
	return nil
}

func (p *Provider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"topics": &TopicGenerator{},
	}
}

func (Provider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *Provider) safeConfigMap() map[string]interface{} {
	config := map[string]interface{}{
		"bootstrap_servers": p.config.BootstrapServers,
		"kafka_version":     p.config.KafkaVersion,
		"tls_enabled":       p.config.TLSEnabled,
		"skip_tls_verify":   p.config.SkipTLSVerify,
	}
	if p.config.SASLMechanism != "" {
		config["sasl_mechanism"] = p.config.SASLMechanism
	}
	if p.config.SASLUsername != "" {
		config["sasl_username"] = p.config.SASLUsername
	}
	if p.config.SASLAWSRegion != "" {
		config["sasl_aws_region"] = p.config.SASLAWSRegion
	}
	if p.config.SASLAWSContainerAuthorizationTokenFile != "" {
		config["sasl_aws_container_authorization_token_file"] = p.config.SASLAWSContainerAuthorizationTokenFile
	}
	if p.config.SASLAWSContainerCredentialsFullURI != "" {
		config["sasl_aws_container_credentials_full_uri"] = p.config.SASLAWSContainerCredentialsFullURI
	}
	if p.config.SASLAWSRoleARN != "" {
		config["sasl_aws_role_arn"] = p.config.SASLAWSRoleARN
	}
	if p.config.SASLAWSExternalID != "" {
		config["sasl_aws_external_id"] = p.config.SASLAWSExternalID
	}
	if p.config.SASLAWSProfile != "" {
		config["sasl_aws_profile"] = p.config.SASLAWSProfile
	}
	if len(p.config.SASLAWSSharedConfigFiles) > 0 {
		config["sasl_aws_shared_config_files"] = p.config.SASLAWSSharedConfigFiles
	}
	if p.config.SASLTokenURL != "" {
		config["sasl_token_url"] = p.config.SASLTokenURL
	}
	if len(p.config.SASLOAuthScopes) > 0 {
		config["sasl_oauth_scopes"] = p.config.SASLOAuthScopes
	}
	if p.config.CACert != "" {
		config["ca_cert"] = p.config.CACert
	}
	if p.config.ClientCert != "" {
		config["client_cert"] = p.config.ClientCert
	}
	if p.config.Timeout != 0 {
		config["timeout"] = p.config.Timeout
	}
	return config
}

func (p *Provider) safeConfigCTY() map[string]cty.Value {
	config := map[string]cty.Value{
		"bootstrap_servers": ctyStringList(p.config.BootstrapServers),
		"kafka_version":     cty.StringVal(p.config.KafkaVersion),
		"tls_enabled":       cty.BoolVal(p.config.TLSEnabled),
		"skip_tls_verify":   cty.BoolVal(p.config.SkipTLSVerify),
	}
	if p.config.SASLMechanism != "" {
		config["sasl_mechanism"] = cty.StringVal(p.config.SASLMechanism)
	}
	if p.config.SASLUsername != "" {
		config["sasl_username"] = cty.StringVal(p.config.SASLUsername)
	}
	if p.config.SASLAWSRegion != "" {
		config["sasl_aws_region"] = cty.StringVal(p.config.SASLAWSRegion)
	}
	if p.config.SASLAWSContainerAuthorizationTokenFile != "" {
		config["sasl_aws_container_authorization_token_file"] = cty.StringVal(p.config.SASLAWSContainerAuthorizationTokenFile)
	}
	if p.config.SASLAWSContainerCredentialsFullURI != "" {
		config["sasl_aws_container_credentials_full_uri"] = cty.StringVal(p.config.SASLAWSContainerCredentialsFullURI)
	}
	if p.config.SASLAWSRoleARN != "" {
		config["sasl_aws_role_arn"] = cty.StringVal(p.config.SASLAWSRoleARN)
	}
	if p.config.SASLAWSExternalID != "" {
		config["sasl_aws_external_id"] = cty.StringVal(p.config.SASLAWSExternalID)
	}
	if p.config.SASLAWSProfile != "" {
		config["sasl_aws_profile"] = cty.StringVal(p.config.SASLAWSProfile)
	}
	if len(p.config.SASLAWSSharedConfigFiles) > 0 {
		config["sasl_aws_shared_config_files"] = ctyStringList(p.config.SASLAWSSharedConfigFiles)
	}
	if p.config.SASLTokenURL != "" {
		config["sasl_token_url"] = cty.StringVal(p.config.SASLTokenURL)
	}
	if len(p.config.SASLOAuthScopes) > 0 {
		config["sasl_oauth_scopes"] = ctyStringList(p.config.SASLOAuthScopes)
	}
	if p.config.CACert != "" {
		config["ca_cert"] = cty.StringVal(p.config.CACert)
	}
	if p.config.ClientCert != "" {
		config["client_cert"] = cty.StringVal(p.config.ClientCert)
	}
	if p.config.Timeout != 0 {
		config["timeout"] = cty.NumberIntVal(int64(p.config.Timeout))
	}
	return config
}

func ctyStringList(values []string) cty.Value {
	ctyValues := make([]cty.Value, 0, len(values))
	for _, value := range values {
		ctyValues = append(ctyValues, cty.StringVal(value))
	}
	if len(ctyValues) == 0 {
		return cty.ListValEmpty(cty.String)
	}
	return cty.ListVal(ctyValues)
}
