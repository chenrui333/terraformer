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
	"time"

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
	AuxiliaryTenantIDs            []string
	ClientCertificatePassword     string
	ClientCertificatePath         string
	ClientID                      string
	ClientSecret                  string
	CustomManagedIdentityEndpoint string
	Environment                   string
	GitHubOIDCTokenRequestToken   string
	GitHubOIDCTokenRequestURL     string
	MetadataHost                  string
	SubscriptionID                string
	TenantID                      string
	UseClientCertificate          bool
	UseClientSecret               bool
	UseGitHubOIDC                 bool
	UseManagedIdentity            bool
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
	metadataHost := os.Getenv("ARM_METADATA_HOSTNAME")
	environment = normalizeEnvironment(environment)
	if metadataHost == "" && environment == "" {
		return fmt.Errorf("unsupported ARM_ENVIRONMENT %q (supported: public, china, usgovernment and their Azure SDK aliases; or set ARM_METADATA_HOSTNAME for custom clouds)", os.Getenv("ARM_ENVIRONMENT"))
	}
	var auxTenants []string
	if v := os.Getenv("ARM_AUXILIARY_TENANT_IDS"); v != "" {
		auxTenants = strings.Split(v, ";")
		if len(auxTenants) > 3 {
			return fmt.Errorf("the provider only supports 3 auxiliary tenant IDs for ARM_AUXILIARY_TENANT_IDS")
		}
	}
	p.config = providerConfig{
		AuxiliaryTenantIDs:            auxTenants,
		ClientCertificatePassword:     os.Getenv("ARM_CLIENT_CERTIFICATE_PASSWORD"),
		ClientCertificatePath:         os.Getenv("ARM_CLIENT_CERTIFICATE_PATH"),
		ClientID:                      os.Getenv("ARM_CLIENT_ID"),
		ClientSecret:                  os.Getenv("ARM_CLIENT_SECRET"),
		CustomManagedIdentityEndpoint: os.Getenv("ARM_MSI_ENDPOINT"),
		Environment:                   environment,
		GitHubOIDCTokenRequestToken:   os.Getenv("ARM_OIDC_REQUEST_TOKEN"),
		GitHubOIDCTokenRequestURL:     os.Getenv("ARM_OIDC_REQUEST_URL"),
		MetadataHost:                  metadataHost,
		SubscriptionID:                subscriptionID,
		TenantID:                      os.Getenv("ARM_TENANT_ID"),
		UseClientCertificate:          os.Getenv("ARM_CLIENT_CERTIFICATE_PATH") != "",
		UseClientSecret:               os.Getenv("ARM_CLIENT_SECRET") != "",
		UseGitHubOIDC:                 os.Getenv("ARM_USE_OIDC") != "",
		UseManagedIdentity:            os.Getenv("ARM_USE_MSI") != "",
	}
	return nil
}

func (p *AzureProvider) getTokenCredential() (azcore.TokenCredential, error) {
	cloudCfg := p.clientOptions.Cloud
	isCustomCloud := p.config.MetadataHost != ""
	if p.config.UseClientSecret {
		opts := &azidentity.ClientSecretCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			DisableInstanceDiscovery:   isCustomCloud,
		}
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
		opts := &azidentity.ClientCertificateCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			DisableInstanceDiscovery:   isCustomCloud,
		}
		opts.Cloud = cloudCfg
		return azidentity.NewClientCertificateCredential(
			p.config.TenantID, p.config.ClientID, certs, key, opts)
	}
	if p.config.UseManagedIdentity {
		if p.config.CustomManagedIdentityEndpoint != "" {
			os.Setenv("MSI_ENDPOINT", p.config.CustomManagedIdentityEndpoint)
		}
		opts := &azidentity.ManagedIdentityCredentialOptions{}
		opts.Cloud = cloudCfg
		if p.config.ClientID != "" {
			opts.ID = azidentity.ClientID(p.config.ClientID)
		}
		return azidentity.NewManagedIdentityCredential(opts)
	}
	if p.config.UseGitHubOIDC {
		const oidcAudience = "api://AzureADTokenExchange"
		getAssertion := func(ctx context.Context) (string, error) {
			reqURL := p.config.GitHubOIDCTokenRequestURL
			if !strings.Contains(reqURL, "audience=") {
				if strings.Contains(reqURL, "?") {
					reqURL += "&audience=" + oidcAudience
				} else {
					reqURL += "?audience=" + oidcAudience
				}
			}
			req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
		opts := &azidentity.ClientAssertionCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			DisableInstanceDiscovery:   isCustomCloud,
		}
		opts.Cloud = cloudCfg
		return azidentity.NewClientAssertionCredential(
			p.config.TenantID, p.config.ClientID, getAssertion, opts)
	}
	cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
		AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
		Subscription:               p.config.SubscriptionID,
		TenantID:                   p.config.TenantID,
	})
	if err != nil {
		return azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			TenantID:                   p.config.TenantID,
		})
	}
	return cliCred, nil
}

func (p *AzureProvider) getClientOptions() (*arm.ClientOptions, error) {
	opts := &arm.ClientOptions{
		AuxiliaryTenants: p.config.AuxiliaryTenantIDs,
	}
	if p.config.MetadataHost != "" {
		cloudCfg, err := discoverCloudConfig(p.config.MetadataHost, p.config.Environment)
		if err != nil {
			return nil, fmt.Errorf("discovering cloud config from %s: %w", p.config.MetadataHost, err)
		}
		opts.Cloud = cloudCfg
	} else {
		switch p.config.Environment {
		case "china":
			opts.Cloud = cloud.AzureChina
		case "usgovernment":
			opts.Cloud = cloud.AzureGovernment
		}
	}
	return opts, nil
}

func normalizeEnvironment(env string) string {
	switch strings.ToLower(env) {
	case "public", "azurecloud", "azurepubliccloud":
		return "public"
	case "china", "azurechinacloud":
		return "china"
	case "usgovernment", "azureusgovernment", "azureusgovernmentcloud":
		return "usgovernment"
	default:
		return ""
	}
}

func discoverCloudConfig(metadataHost, environment string) (cloud.Configuration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := fmt.Sprintf("https://%s/metadata/endpoints?api-version=2019-05-01", metadataHost)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return cloud.Configuration{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return cloud.Configuration{}, err
	}
	defer resp.Body.Close()
	type cloudEndpoint struct {
		Authentication struct {
			LoginEndpoint string   `json:"loginEndpoint"`
			Audiences     []string `json:"audiences"`
		} `json:"authentication"`
		ResourceManager string `json:"resourceManager"`
		Name            string `json:"name"`
	}
	var endpoints []cloudEndpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return cloud.Configuration{}, fmt.Errorf("decoding metadata endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		return cloud.Configuration{}, fmt.Errorf("metadata endpoint returned no environments")
	}
	var meta *cloudEndpoint
	if environment != "" {
		for i := range endpoints {
			if strings.EqualFold(endpoints[i].Name, environment) {
				meta = &endpoints[i]
				break
			}
		}
	}
	if meta == nil {
		meta = &endpoints[0]
	}
	audience := meta.ResourceManager
	if len(meta.Authentication.Audiences) > 0 {
		audience = meta.Authentication.Audiences[0]
	}
	return cloud.Configuration{
		ActiveDirectoryAuthorityHost: meta.Authentication.LoginEndpoint,
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Audience: audience,
				Endpoint: meta.ResourceManager,
			},
		},
	}, nil
}

func (p *AzureProvider) Init(args []string) error {
	err := p.setEnvConfig()
	if err != nil {
		return err
	}

	clientOptions, err := p.getClientOptions()
	if err != nil {
		return err
	}
	p.clientOptions = clientOptions

	credential, err := p.getTokenCredential()
	if err != nil {
		return err
	}
	p.credential = credential
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
