// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
)

type AzureProvider struct { //nolint
	terraformutils.Provider
	config        providerConfig
	credential    azcore.TokenCredential
	clientOptions *arm.ClientOptions
	resourceGroup string
}

type providerConfig struct {
	ClientCertificatePassword   string
	ClientCertificatePath       string
	ClientID                    string
	ClientSecret                string
	Environment                 string
	GitHubOIDCTokenRequestToken string
	GitHubOIDCTokenRequestURL   string
	SubscriptionID              string
	TenantID                    string
	UseClientCertificate        bool
	UseClientSecret             bool
	UseGitHubOIDC               bool
	UseManagedIdentity          bool
}

func (p *AzureProvider) setEnvConfig() error {
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		return errors.New("set ARM_SUBSCRIPTION_ID env var")
	}
	environment := os.Getenv("ARM_ENVIRONMENT")
	if environment == "" {
		environment = "public"
	}
	p.config = providerConfig{
		ClientCertificatePassword:   os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD"),
		ClientCertificatePath:       os.Getenv("ARM_CLIENT_CERTIFICATE_PATH"),
		ClientID:                    os.Getenv("ARM_CLIENT_ID"),
		ClientSecret:                os.Getenv("ARM_CLIENT_SECRET"),
		Environment:                 environment,
		GitHubOIDCTokenRequestToken: os.Getenv("ARM_OIDC_REQUEST_TOKEN"),
		GitHubOIDCTokenRequestURL:   os.Getenv("ARM_OIDC_REQUEST_URL"),
		SubscriptionID:              subscriptionID,
		TenantID:                    os.Getenv("ARM_TENANT_ID"),
		UseClientCertificate:        os.Getenv("ARM_CLIENT_CERTIFICATE_PATH") != "",
		UseClientSecret:             os.Getenv("ARM_CLIENT_SECRET") != "",
		UseGitHubOIDC:               os.Getenv("ARM_USE_OIDC") != "",
		UseManagedIdentity:          os.Getenv("ARM_USE_MSI") != "",
	}
	return nil
}

func (p *AzureProvider) getTokenCredential() (azcore.TokenCredential, error) {
	cloudCfg := p.getClientOptions().Cloud
	if p.config.UseClientSecret {
		opts := &azidentity.ClientSecretCredentialOptions{}
		opts.Cloud = cloudCfg
		return azidentity.NewClientSecretCredential(
			p.config.TenantID, p.config.ClientID, p.config.ClientSecret, opts)
	}
	if p.config.UseClientCertificate {
		certData, err := os.ReadFile(p.config.ClientCertificatePath)
		if err != nil {
			return nil, fmt.Errorf("reading client certificate: %w", err)
		}
		certs, key, err := azidentity.ParseCertificates(certData, []byte(p.config.ClientCertificatePassword))
		if err != nil {
			return nil, fmt.Errorf("parsing client certificate: %w", err)
		}
		opts := &azidentity.ClientCertificateCredentialOptions{}
		opts.Cloud = cloudCfg
		return azidentity.NewClientCertificateCredential(
			p.config.TenantID, p.config.ClientID, certs, key, opts)
	}
	if p.config.UseManagedIdentity {
		opts := &azidentity.ManagedIdentityCredentialOptions{}
		opts.Cloud = cloudCfg
		if p.config.ClientID != "" {
			opts.ID = azidentity.ClientID(p.config.ClientID)
		}
		return azidentity.NewManagedIdentityCredential(opts)
	}
	if p.config.UseGitHubOIDC {
		getAssertion := func(ctx context.Context) (string, error) {
			req, err := http.NewRequestWithContext(ctx, "GET", p.config.GitHubOIDCTokenRequestURL, nil)
			if err != nil {
				return "", fmt.Errorf("creating OIDC token request: %w", err)
			}
			req.Header.Set("Authorization", "Bearer "+p.config.GitHubOIDCTokenRequestToken)
			req.Header.Set("Accept", "application/json; api-version=2.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("requesting OIDC token: %w", err)
			}
			defer resp.Body.Close()
			var result struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return "", fmt.Errorf("decoding OIDC token response: %w", err)
			}
			return result.Value, nil
		}
		opts := &azidentity.ClientAssertionCredentialOptions{}
		opts.Cloud = cloudCfg
		return azidentity.NewClientAssertionCredential(
			p.config.TenantID, p.config.ClientID, getAssertion, opts)
	}
	opts := &azidentity.DefaultAzureCredentialOptions{}
	opts.Cloud = cloudCfg
	return azidentity.NewDefaultAzureCredential(opts)
}

func (p *AzureProvider) getClientOptions() *arm.ClientOptions {
	opts := &arm.ClientOptions{}
	switch strings.ToLower(p.config.Environment) {
	case "china":
		opts.Cloud = cloud.AzureChina
	case "usgovernment":
		opts.Cloud = cloud.AzureGovernment
	}
	return opts
}

func (p *AzureProvider) Init(args []string) error {
	err := p.setEnvConfig()
	if err != nil {
		return err
	}

	credential, err := p.getTokenCredential()
	if err != nil {
		return err
	}
	p.credential = credential
	p.clientOptions = p.getClientOptions()
	p.resourceGroup = args[0]

	return nil
}

func (p *AzureProvider) GetName() string {
	return "azurerm"
}

func (p *AzureProvider) GetProviderData(_ ...string) map[string]interface{} {
	version := providerwrapper.GetProviderVersion(p.GetName())
	if strings.Contains(version, "v2.") {
		return map[string]interface{}{
			"provider": map[string]interface{}{
				"azurerm": map[string]interface{}{
					"features": map[string]interface{}{},
				},
			},
		}
	}
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"azurerm": map[string]interface{}{
				"version": version,
			},
		},
	}
}

func (AzureProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{
		"analysis": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"app_service": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"application_gateway": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"cosmosdb": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"container": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"database": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"databricks": {
			"resource_group": []string{
				"resource_group_name", "name",
				"managed_resource_group_name", "name",
				"location", "location",
			},
			"storage_account": []string{"storage_account_name", "name"},
			"subnet": []string{
				"public_subnet_name", "name",
				"private_subnet_name", "name",
			},
			"virtual_network": []string{"virtual_network_id", "id"},
		},
		"data_factory": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"data_factory": []string{
				"data_factory_name", "name",
				"data_factory_id", "id",
				"linked_service_name", "name",
				"integration_runtime_name", "name",
			},
			"databricks":      []string{"existing_cluster_id", "id"},
			"keyvault":        []string{"keyvault_id", "id"},
			"storage_account": []string{"storage_account_id", "id"},
		},
		"disk": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"dns": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"eventhub": {
			"resource_group": []string{"resource_group_name", "name"},
			"eventhub": []string{
				"eventhub_name", "name",
				"namespace_name", "name",
			},
		},
		"keyvault": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"load_balancer": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"network_interface": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"subnet": []string{"subnet_id", "id"},
		},
		"network_security_group": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"network_security_group": []string{"network_security_group_name", "name"},
		},
		"network_watcher": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"network_watcher": []string{"network_watcher_name", "name"},
			"storage_account": []string{"storage_account_id", "id"},
		},
		"private_dns": {
			"resource_group":  []string{"resource_group_name", "name"},
			"virtual_network": []string{"virtual_network_id", "id"},
			"private_dns": []string{
				"zone_name", "name",
				"private_dns_zone_name", "name",
			},
		},
		"private_endpoint": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"subnet": []string{"subnet_id", "id"},
		},
		"public_ip": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"purview": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"redis": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"route_table": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"route_table": []string{"route_table_name", "name"},
		},
		"scaleset": {
			"resource_group": []string{"resource_group_name", "name"},
		},
		"ssh_public_key": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
		},
		"storage_account": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"virtual_network": []string{"virtual_network_subnet_ids", "id"},
		},
		"storage_blob": {
			"storage_account":   []string{"storage_account_name", "name"},
			"storage_container": []string{"storage_container_name", "name"},
		},
		"storage_container": {
			"storage_account": []string{"storage_account_name", "name"},
		},
		"synapse": {
			"resource_group": []string{
				"resource_group_name", "name",
				"managed_resource_group_name", "name",
			},
			"synapse": []string{"synapse_workspace_id", "id"},
		},
		"subnet": {
			"resource_group":         []string{"resource_group_name", "name"},
			"virtual_network":        []string{"virtual_network_name", "name"},
			"network_security_group": []string{"network_security_group_id", "id"},
			"route_table":            []string{"route_table_id", "id"},
			"subnet":                 []string{"subnet_id", "id"},
		},
		"virtual_machine": {
			"resource_group": []string{
				"resource_group_name", "name",
				"location", "location",
			},
			"network_interface": []string{"network_interface_ids", "id"},
		},
		"virtual_network": {
			"resource_group": []string{"resource_group_name", "name"},
		},
	}
}

func (p *AzureProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"analysis":                             &AnalysisGenerator{},
		"app_service":                          &AppServiceGenerator{},
		"application_gateway":                  &ApplicationGatewayGenerator{},
		"cosmosdb":                             &CosmosDBGenerator{},
		"container":                            &ContainerGenerator{},
		"database":                             &DatabasesGenerator{},
		"databricks":                           &DatabricksGenerator{},
		"data_factory":                         &DataFactoryGenerator{},
		"disk":                                 &DiskGenerator{},
		"dns":                                  &DNSGenerator{},
		"eventhub":                             &EventHubGenerator{},
		"keyvault":                             &KeyVaultGenerator{},
		"load_balancer":                        &LoadBalancerGenerator{},
		"management_lock":                      &ManagementLockGenerator{},
		"network_interface":                    &NetworkInterfaceGenerator{},
		"network_security_group":               &NetworkSecurityGroupGenerator{},
		"network_watcher":                      &NetworkWatcherGenerator{},
		"private_dns":                          &PrivateDNSGenerator{},
		"private_endpoint":                     &PrivateEndpointGenerator{},
		"public_ip":                            &PublicIPGenerator{},
		"purview":                              &PurviewGenerator{},
		"redis":                                &RedisGenerator{},
		"resource_group":                       &ResourceGroupGenerator{},
		"route_table":                          &RouteTableGenerator{},
		"scaleset":                             &ScaleSetGenerator{},
		"security_center_contact":              &SecurityCenterContactGenerator{},
		"security_center_subscription_pricing": &SecurityCenterSubscriptionPricingGenerator{},
		"ssh_public_key":                       &SSHPublicKeyGenerator{},
		"storage_account":                      &StorageAccountGenerator{},
		"storage_blob":                         &StorageBlobGenerator{},
		"storage_container":                    &StorageContainerGenerator{},
		"synapse":                              &SynapseGenerator{},
		"subnet":                               &SubnetGenerator{},
		"virtual_machine":                      &VirtualMachineGenerator{},
		"virtual_network":                      &VirtualNetworkGenerator{},
	}
}

func (p *AzureProvider) InitService(serviceName string, verbose bool) error {
	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("azurerm: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"config":         p.config,
		"credential":     p.credential,
		"clientOptions":  p.clientOptions,
		"resource_group": p.resourceGroup,
	})
	return nil
}
