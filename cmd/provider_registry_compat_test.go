// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
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

const (
	registryCompatibilityMaxAttempts = 3
	registryCompatibilityMaxParallel = 8
	registryCompatibilityUserAgent   = "terraformer-provider-compat-test/1.0"
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
	semaphore := make(chan struct{}, registryCompatibilityMaxParallel)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

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

// Keep this constructor list in sync with providerImporterSubcommands. The
// audit intentionally uses real provider constructors so it validates the same
// provider GetName values that Terraformer uses during imports.
func registryAuditProviderSources() map[string]string {
	providers := map[string]string{}
	for _, providerGen := range providerGeneratorConstructors() {
		provider := providerGen()
		providers[provider.GetName()] = terraformutils.ProviderSource(provider.GetName())
	}
	return providers
}

func fetchLatestStableRegistryProviderVersion(client *http.Client, source string) (registryProviderVersion, error) {
	var lastErr error
	for attempt := 1; attempt <= registryCompatibilityMaxAttempts; attempt++ {
		version, err := fetchLatestStableRegistryProviderVersionOnce(client, source)
		if err == nil {
			return version, nil
		}
		lastErr = err

		var httpErr registryHTTPError
		if !errors.As(err, &httpErr) || !httpErr.retryable() {
			return registryProviderVersion{}, err
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return registryProviderVersion{}, fmt.Errorf("failed to fetch registry.terraform.io/%s after %d attempts: %w", source, registryCompatibilityMaxAttempts, lastErr)
}

func fetchLatestStableRegistryProviderVersionOnce(client *http.Client, source string) (registryProviderVersion, error) {
	url := fmt.Sprintf("https://registry.terraform.io/v1/providers/%s/versions", source)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return registryProviderVersion{}, err
	}
	req.Header.Set("User-Agent", registryCompatibilityUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return registryProviderVersion{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return registryProviderVersion{}, registryHTTPError{url: url, statusCode: resp.StatusCode, status: resp.Status}
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

type registryHTTPError struct {
	url        string
	statusCode int
	status     string
}

func (e registryHTTPError) Error() string {
	return fmt.Sprintf("%s returned %s", e.url, e.status)
}

func (e registryHTTPError) retryable() bool {
	return e.statusCode == http.StatusTooManyRequests || e.statusCode >= http.StatusInternalServerError
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
