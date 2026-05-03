// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	sslmatepkcs12 "software.sslmate.com/src/go-pkcs12"

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

type azureCLIUnavailableFallbackCredential struct {
	credential azcore.TokenCredential
}

func (c azureCLIUnavailableFallbackCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	token, err := c.credential.GetToken(ctx, opts)
	if azureCLIErrorIsUnavailable(err) {
		return token, azidentity.NewCredentialUnavailableError(err.Error())
	}
	return token, err
}

func azureCLIErrorIsUnavailable(err error) bool {
	if err == nil {
		return false
	}
	lowerMsg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"azure cli not found on path",
		"executable not found on path",
	} {
		if strings.Contains(lowerMsg, marker) {
			return true
		}
	}

	for _, marker := range []string{
		"aadsts",
		"claims",
		"denied",
		"expired",
		"forbidden",
		"invalid",
		"scope",
		"tenant",
	} {
		if strings.Contains(lowerMsg, marker) {
			return false
		}
	}
	return strings.Contains(lowerMsg, "please run 'az login'") ||
		strings.Contains(lowerMsg, "please run \"az login\"")
}

type lazyDefaultAzureCredential struct {
	options    *azidentity.DefaultAzureCredentialOptions
	once       sync.Once
	credential azcore.TokenCredential
	err        error
}

func (c *lazyDefaultAzureCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.once.Do(func() {
		c.credential, c.err = azidentity.NewDefaultAzureCredential(c.options)
	})
	if c.err != nil {
		return azcore.AccessToken{}, c.err
	}
	return c.credential.GetToken(ctx, opts)
}

func getGitHubOIDCAssertion(ctx context.Context, requestURL, requestToken string) (string, error) {
	const oidcAudience = "api://AzureADTokenExchange"
	reqURL := requestURL
	if !strings.Contains(reqURL, "audience=") {
		if strings.Contains(reqURL, "?") {
			reqURL += "&audience=" + oidcAudience
		} else {
			reqURL += "?audience=" + oidcAudience
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating OIDC token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)
	req.Header.Set("Accept", "application/json; api-version=2.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting OIDC token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading OIDC token response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return "", fmt.Errorf("requesting OIDC token returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decoding OIDC token response: %w", err)
	}
	if result.Value == "" {
		return "", errors.New("oidc token response missing value")
	}
	return result.Value, nil
}

func parseClientCertificate(certData []byte, password string) ([]*x509.Certificate, crypto.PrivateKey, error) {
	certs, key, err := azidentity.ParseCertificates(certData, []byte(password))
	if err == nil {
		return certs, key, nil
	}

	pfxKey, cert, caCerts, pfxErr := sslmatepkcs12.DecodeChain(certData, password)
	if pfxErr != nil {
		return nil, nil, fmt.Errorf("%w; parsing PKCS#12 fallback: %w", err, pfxErr)
	}
	if cert == nil {
		return nil, nil, errors.New("found no certificate")
	}
	if pfxKey == nil {
		return nil, nil, errors.New("found no private key")
	}
	certs = append([]*x509.Certificate{cert}, caCerts...)
	return certs, pfxKey, nil
}

type customManagedIdentityCredential struct {
	clientID   string
	endpoint   string
	httpClient *http.Client
}

func (c *customManagedIdentityCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if len(opts.Scopes) != 1 {
		return azcore.AccessToken{}, fmt.Errorf("custom managed identity credential requires exactly one scope")
	}

	tokenURL, err := url.Parse(c.endpoint)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("parsing managed identity endpoint: %w", err)
	}
	query := tokenURL.Query()
	query.Set("api-version", "2018-02-01")
	query.Set("resource", strings.TrimSuffix(opts.Scopes[0], "/.default"))
	if c.clientID != "" {
		query.Set("client_id", c.clientID)
	}
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), http.NoBody)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("creating managed identity token request: %w", err)
	}
	req.Header.Set("Metadata", "true")

	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("requesting managed identity token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return azcore.AccessToken{}, fmt.Errorf("reading managed identity token response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return azcore.AccessToken{}, fmt.Errorf("managed identity endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		AccessToken string      `json:"access_token"`
		ExpiresIn   interface{} `json:"expires_in"`
		ExpiresOn   interface{} `json:"expires_on"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return azcore.AccessToken{}, fmt.Errorf("decoding managed identity token response: %w", err)
	}
	if result.AccessToken == "" {
		return azcore.AccessToken{}, errors.New("managed identity token response missing access_token")
	}

	return azcore.AccessToken{
		Token:     result.AccessToken,
		ExpiresOn: managedIdentityTokenExpiresOn(result.ExpiresIn, result.ExpiresOn),
	}, nil
}

func managedIdentityTokenExpiresOn(expiresIn interface{}, expiresOn interface{}) time.Time {
	if seconds, ok := tokenDurationSeconds(expiresIn); ok && seconds > 0 {
		return time.Now().Add(time.Duration(seconds) * time.Second)
	}
	if timestamp, ok := tokenDurationSeconds(expiresOn); ok && timestamp > 0 {
		return time.Unix(timestamp, 0)
	}
	return time.Time{}
}

func tokenDurationSeconds(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case string:
		if v == "" {
			return 0, false
		}
		seconds, err := strconv.ParseInt(v, 10, 64)
		return seconds, err == nil
	default:
		return 0, false
	}
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
	rawEnvironment := os.Getenv("ARM_ENVIRONMENT")
	if rawEnvironment == "" {
		rawEnvironment = "public"
	}
	metadataHost := os.Getenv("ARM_METADATA_HOSTNAME")
	environment := normalizeEnvironment(rawEnvironment)
	if environment == "" {
		if metadataHost == "" {
			return fmt.Errorf("unsupported ARM_ENVIRONMENT %q (supported: public, china, usgovernment and their Azure SDK aliases; or set ARM_METADATA_HOSTNAME for custom clouds)", os.Getenv("ARM_ENVIRONMENT"))
		}
		environment = rawEnvironment
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
	if p.config.UseClientCertificate && p.config.ClientID != "" && p.config.TenantID != "" {
		certData, err := os.ReadFile(p.config.ClientCertificatePath)
		if err != nil {
			return nil, fmt.Errorf("reading client certificate: %w", err)
		}
		certs, key, err := parseClientCertificate(certData, p.config.ClientCertificatePassword)
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
	if p.config.UseClientSecret && p.config.ClientID != "" && p.config.TenantID != "" {
		opts := &azidentity.ClientSecretCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			DisableInstanceDiscovery:   isCustomCloud,
		}
		opts.Cloud = cloudCfg
		return azidentity.NewClientSecretCredential(
			p.config.TenantID, p.config.ClientID, p.config.ClientSecret, opts)
	}
	if p.config.UseGitHubOIDC && p.config.ClientID != "" && p.config.TenantID != "" && p.config.GitHubOIDCTokenRequestURL != "" && p.config.GitHubOIDCTokenRequestToken != "" {
		getAssertion := func(ctx context.Context) (string, error) {
			return getGitHubOIDCAssertion(ctx, p.config.GitHubOIDCTokenRequestURL, p.config.GitHubOIDCTokenRequestToken)
		}
		opts := &azidentity.ClientAssertionCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			DisableInstanceDiscovery:   isCustomCloud,
		}
		opts.Cloud = cloudCfg
		return azidentity.NewClientAssertionCredential(
			p.config.TenantID, p.config.ClientID, getAssertion, opts)
	}
	if p.config.UseManagedIdentity {
		if p.config.CustomManagedIdentityEndpoint != "" {
			return &customManagedIdentityCredential{
				clientID: p.config.ClientID,
				endpoint: p.config.CustomManagedIdentityEndpoint,
			}, nil
		}
		opts := &azidentity.ManagedIdentityCredentialOptions{}
		opts.Cloud = cloudCfg
		if p.config.ClientID != "" {
			opts.ID = azidentity.ClientID(p.config.ClientID)
		}
		return azidentity.NewManagedIdentityCredential(opts)
	}
	defaultCred := &lazyDefaultAzureCredential{
		options: &azidentity.DefaultAzureCredentialOptions{
			AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
			ClientOptions:              azcore.ClientOptions{Cloud: cloudCfg},
			DisableInstanceDiscovery:   isCustomCloud,
			TenantID:                   p.config.TenantID,
		},
	}
	cliCred, cliErr := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
		AdditionallyAllowedTenants: p.config.AuxiliaryTenantIDs,
		Subscription:               p.config.SubscriptionID,
		TenantID:                   p.config.TenantID,
	})
	if cliErr == nil {
		return azidentity.NewChainedTokenCredential([]azcore.TokenCredential{
			azureCLIUnavailableFallbackCredential{credential: cliCred},
			defaultCred,
		}, nil)
	}
	return defaultCred, nil
}

func (p *AzureProvider) getClientOptions() (*arm.ClientOptions, error) {
	opts := &arm.ClientOptions{
		AuxiliaryTenants:      p.config.AuxiliaryTenantIDs,
		DisableRPRegistration: true,
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
		case "german":
			opts.Cloud = azureGermanCloud()
		}
	}
	return opts, nil
}

func azureGermanCloud() cloud.Configuration {
	return cloud.Configuration{
		ActiveDirectoryAuthorityHost: "https://login.microsoftonline.de/",
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Audience: "https://management.microsoftazure.de/",
				Endpoint: "https://management.microsoftazure.de",
			},
		},
	}
}

func normalizeEnvironment(env string) string {
	switch strings.ToLower(env) {
	case "public", "azurecloud", "azurepubliccloud":
		return "public"
	case "china", "azurechinacloud":
		return "china"
	case "usgovernment", "azureusgovernment", "azureusgovernmentcloud":
		return "usgovernment"
	case "german", "azuregermancloud":
		return "german"
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return cloud.Configuration{}, fmt.Errorf("metadata endpoint returned %s; reading response body: %w", resp.Status, err)
		}
		return cloud.Configuration{}, fmt.Errorf("metadata endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

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
		candidates := []string{environment}
		switch environment {
		case "public":
			candidates = append(candidates, "AzureCloud", "AzurePublicCloud")
		case "china":
			candidates = append(candidates, "AzureChinaCloud")
		case "usgovernment":
			candidates = append(candidates, "AzureUSGovernmentCloud", "AzureUSGovernment")
		case "german":
			candidates = append(candidates, "AzureGermanCloud")
		}
		for i := range endpoints {
			for _, name := range candidates {
				if strings.EqualFold(endpoints[i].Name, name) {
					meta = &endpoints[i]
					break
				}
			}
			if meta != nil {
				break
			}
		}
		if meta == nil {
			return cloud.Configuration{}, fmt.Errorf("metadata endpoint has no environment matching %q", environment)
		}
	} else {
		meta = &endpoints[0]
	}
	resourceManager := meta.ResourceManager
	if resourceManager == "" {
		resourceManager = fmt.Sprintf("https://%s/", metadataHost)
	}
	audience := resourceManager
	if len(meta.Authentication.Audiences) > 0 {
		audience = meta.Authentication.Audiences[0]
	}
	return cloud.Configuration{
		ActiveDirectoryAuthorityHost: meta.Authentication.LoginEndpoint,
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Audience: audience,
				Endpoint: resourceManager,
			},
		},
	}, nil
}

func (p *AzureProvider) Init(args []string) error {
	p.config = providerConfig{}
	p.credential = nil
	p.clientOptions = nil
	p.resourceGroup = ""

	if len(args) < 1 {
		return errors.New("azure: expected 1 init arg (resource group)")
	}

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
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("azurerm: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"config":         p.config,
		"credential":     p.credential,
		"clientOptions":  p.clientOptions,
		"resource_group": p.resourceGroup,
	})
	return nil
}
