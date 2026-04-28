// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
)

type registryVersionsResponse struct {
	Versions []registryProviderVersion `json:"versions"`
}

type registryProviderVersion struct {
	Version   string                     `json:"version"`
	Protocols []string                   `json:"protocols"`
	Platforms []registryProviderPlatform `json:"platforms"`
}

type registryProviderPlatform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

func TestProviderRegistryCompatibility(t *testing.T) {
	if os.Getenv("TERRAFORMER_PROVIDER_COMPAT_TEST") == "" {
		t.Skip("set TERRAFORMER_PROVIDER_COMPAT_TEST to audit Terraform Registry provider compatibility")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	requiredPlatforms := requiredRegistryPlatforms()
	providers := registryAuditProviderSources()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			source := providers[name]
			version, err := fetchLatestStableRegistryProviderVersion(client, source)
			if err != nil {
				t.Fatal(err)
			}
			if !supportsTerraform1ProviderProtocol(version.Protocols) {
				t.Fatalf("registry.terraform.io/%s latest stable version %s protocols = %v, want protocol 5.x or 6.x", source, version.Version, version.Protocols)
			}
			missingPlatforms := missingRegistryPlatforms(version.Platforms, requiredPlatforms)
			if len(missingPlatforms) > 0 {
				t.Fatalf("registry.terraform.io/%s latest stable version %s missing platforms %v", source, version.Version, missingPlatforms)
			}
			t.Logf("registry.terraform.io/%s version %s protocols=%v", source, version.Version, version.Protocols)
		})
	}
}

func registryAuditProviderSources() map[string]string {
	providers := map[string]string{}
	for _, providerGen := range []func() terraformutils.ProviderGenerator{
		newGoogleProvider,
		newAWSProvider,
		newAzureProvider,
		newAliCloudProvider,
		newIbmProvider,
		newDigitalOceanProvider,
		newEquinixMetalProvider,
		newFastlyProvider,
		newHerokuProvider,
		newLaunchDarklyProvider,
		newLinodeProvider,
		newNs1Provider,
		newOpenStackProvider,
		newTencentCloudProvider,
		newVultrProvider,
		newYandexProvider,
		newIonosCloudProvider,
		newKubernetesProvider,
		newOctopusDeployProvider,
		newRabbitMQProvider,
		newMyrasecProvider,
		newCloudflareProvider,
		newPanosProvider,
		newAzureDevOpsProvider,
		newAzureADProvider,
		newGitHubProvider,
		newGitLabProvider,
		newDataDogProvider,
		newNewRelicProvider,
		newMackerelProvider,
		newGrafanaProvider,
		newPagerDutyProvider,
		newOpsgenieProvider,
		newHoneycombioProvider,
		newOpalProvider,
		newKeycloakProvider,
		newLogzioProvider,
		newCommercetoolsProvider,
		newMikrotikProvider,
		newXenorchestraProvider,
		newGmailfilterProvider,
		newVaultProvider,
		newOktaProvider,
		newAuth0Provider,
	} {
		provider := providerGen()
		providers[provider.GetName()] = terraformutils.ProviderSource(provider.GetName())
	}
	return providers
}

func fetchLatestStableRegistryProviderVersion(client *http.Client, source string) (registryProviderVersion, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/versions", source)
	resp, err := client.Get(url)
	if err != nil {
		return registryProviderVersion{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return registryProviderVersion{}, fmt.Errorf("%s returned %s", url, resp.Status)
	}

	var payload registryVersionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return registryProviderVersion{}, err
	}
	version, ok := latestStableRegistryProviderVersion(payload.Versions)
	if !ok {
		return registryProviderVersion{}, fmt.Errorf("registry.terraform.io/%s has no stable semver versions", source)
	}
	return version, nil
}

func latestStableRegistryProviderVersion(versions []registryProviderVersion) (registryProviderVersion, bool) {
	var latest registryProviderVersion
	latestParts := parsedRegistryProviderVersion{}
	found := false
	for _, version := range versions {
		parts, ok := parseRegistryProviderVersion(version.Version)
		if !ok || parts.prerelease {
			continue
		}
		if !found || parts.compare(latestParts) > 0 {
			latest = version
			latestParts = parts
			found = true
		}
	}
	return latest, found
}

type parsedRegistryProviderVersion struct {
	major      int
	minor      int
	patch      int
	prerelease bool
}

func parseRegistryProviderVersion(version string) (parsedRegistryProviderVersion, bool) {
	version, prerelease, _ := strings.Cut(version, "-")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return parsedRegistryProviderVersion{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return parsedRegistryProviderVersion{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return parsedRegistryProviderVersion{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return parsedRegistryProviderVersion{}, false
	}
	return parsedRegistryProviderVersion{major: major, minor: minor, patch: patch, prerelease: prerelease != ""}, true
}

func (v parsedRegistryProviderVersion) compare(other parsedRegistryProviderVersion) int {
	for _, pair := range [][2]int{{v.major, other.major}, {v.minor, other.minor}, {v.patch, other.patch}} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}
	return 0
}

func supportsTerraform1ProviderProtocol(protocols []string) bool {
	for _, protocol := range protocols {
		if strings.HasPrefix(protocol, "5.") || strings.HasPrefix(protocol, "6.") {
			return true
		}
	}
	return false
}

func requiredRegistryPlatforms() []string {
	platforms := strings.FieldsFunc(os.Getenv("TERRAFORMER_PROVIDER_COMPAT_PLATFORMS"), func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	if len(platforms) == 0 {
		return []string{"linux_amd64", "darwin_arm64"}
	}
	return platforms
}

func missingRegistryPlatforms(platforms []registryProviderPlatform, requiredPlatforms []string) []string {
	available := map[string]bool{}
	for _, platform := range platforms {
		available[platform.OS+"_"+platform.Arch] = true
	}
	var missing []string
	for _, platform := range requiredPlatforms {
		if !available[platform] {
			missing = append(missing, platform)
		}
	}
	return missing
}
