// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderInit(t *testing.T) {
	provider := &Provider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
}

func TestProviderSupportedServices(t *testing.T) {
	provider := &Provider{}
	services := provider.GetSupportedService()
	if _, ok := services["release"]; !ok {
		t.Fatalf("release service not registered: %#v", services)
	}
	if err := provider.InitService("release", false); err != nil {
		t.Fatalf("InitService(release) error = %v", err)
	}
	if provider.GetService() == nil {
		t.Fatal("InitService(release) did not set service")
	}
	if provider.GetService().GetName() != "release" {
		t.Fatalf("service name = %q, want release", provider.GetService().GetName())
	}
	if provider.GetService().GetProviderName() != "helm" {
		t.Fatalf("service provider name = %q, want helm", provider.GetService().GetProviderName())
	}
}

func TestProviderRejectsUnsupportedService(t *testing.T) {
	provider := &Provider{}
	if err := provider.InitService("chart", false); err == nil {
		t.Fatal("expected unsupported service error")
	} else if !strings.Contains(err.Error(), "chart not supported service") {
		t.Fatalf("InitService error = %q, want unsupported service", err)
	}
	if provider.GetService() != nil {
		t.Fatalf("unsupported service left stale service %T", provider.GetService())
	}
}

func TestProviderValidateImportAllowsRelease(t *testing.T) {
	provider := &Provider{}
	if err := provider.ValidateImport([]string{"release"}); err != nil {
		t.Fatalf("ValidateImport(release) error = %v", err)
	}
}

func TestProviderGetConfigBridgesKubeconfigForRefresh(t *testing.T) {
	t.Setenv("KUBE_CONFIG_PATH", "")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "/tmp/terraformer-helm-kubeconfig")
	t.Setenv("KUBE_CTX", "")
	t.Setenv("HELM_KUBECONTEXT", "staging")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBE_CONFIG_PATH"); got != "/tmp/terraformer-helm-kubeconfig" {
		t.Fatalf("KUBE_CONFIG_PATH = %q, want bridged KUBECONFIG", got)
	}
	if got := os.Getenv("KUBE_CONFIG_PATHS"); got != "" {
		t.Fatalf("KUBE_CONFIG_PATHS = %q, want empty for single KUBECONFIG path", got)
	}
	if got := os.Getenv("KUBE_CTX"); got != "staging" {
		t.Fatalf("KUBE_CTX = %q, want HELM_KUBECONTEXT", got)
	}
}

func TestProviderGetConfigBridgesProviderKubeEnvForDiscovery(t *testing.T) {
	t.Setenv("KUBE_CONFIG_PATH", "/tmp/provider-kubeconfig")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBE_CTX", "provider-context")
	t.Setenv("HELM_KUBECONTEXT", "")
	t.Setenv("KUBE_HOST", "https://example.test")
	t.Setenv("HELM_KUBEAPISERVER", "")
	t.Setenv("KUBE_TOKEN", "provider-token")
	t.Setenv("HELM_KUBETOKEN", "")
	t.Setenv("KUBE_INSECURE", "true")
	t.Setenv("HELM_KUBEINSECURE_SKIP_TLS_VERIFY", "")
	t.Setenv("KUBE_TLS_SERVER_NAME", "api.example.test")
	t.Setenv("HELM_KUBETLS_SERVER_NAME", "")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBECONFIG"); got != "/tmp/provider-kubeconfig" {
		t.Fatalf("KUBECONFIG = %q, want provider kubeconfig path", got)
	}
	if got := os.Getenv("HELM_KUBECONTEXT"); got != "provider-context" {
		t.Fatalf("HELM_KUBECONTEXT = %q, want provider context", got)
	}
	if got := os.Getenv("HELM_KUBEAPISERVER"); got != "https://example.test" {
		t.Fatalf("HELM_KUBEAPISERVER = %q, want provider host", got)
	}
	if got := os.Getenv("HELM_KUBETOKEN"); got != "provider-token" {
		t.Fatalf("HELM_KUBETOKEN = %q, want provider token", got)
	}
	if got := os.Getenv("HELM_KUBEINSECURE_SKIP_TLS_VERIFY"); got != "true" {
		t.Fatalf("HELM_KUBEINSECURE_SKIP_TLS_VERIFY = %q, want provider insecure flag", got)
	}
	if got := os.Getenv("HELM_KUBETLS_SERVER_NAME"); got != "api.example.test" {
		t.Fatalf("HELM_KUBETLS_SERVER_NAME = %q, want provider TLS server name", got)
	}
}

func TestProviderGetConfigBridgesMultipleKubeconfigPaths(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	kubeconfig := strings.Join([]string{first, second}, string(os.PathListSeparator))
	t.Setenv("KUBE_CONFIG_PATH", "")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", kubeconfig)
	t.Setenv("KUBE_CTX", "")
	t.Setenv("HELM_KUBECONTEXT", "")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBE_CONFIG_PATH"); got != "" {
		t.Fatalf("KUBE_CONFIG_PATH = %q, want empty for multiple KUBECONFIG paths", got)
	}
	if got := os.Getenv("KUBE_CONFIG_PATHS"); got != kubeconfig {
		t.Fatalf("KUBE_CONFIG_PATHS = %q, want %q", got, kubeconfig)
	}
}

func TestProviderGetConfigProviderKubeConfigPathsOverrideKubeconfig(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	providerKubeconfig := strings.Join([]string{first, second}, string(os.PathListSeparator))
	t.Setenv("KUBE_CONFIG_PATH", "")
	t.Setenv("KUBE_CONFIG_PATHS", providerKubeconfig)
	t.Setenv("KUBECONFIG", "/tmp/client-go-kubeconfig")
	t.Setenv("KUBE_CTX", "")
	t.Setenv("HELM_KUBECONTEXT", "")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBECONFIG"); got != providerKubeconfig {
		t.Fatalf("KUBECONFIG = %q, want provider config paths %q", got, providerKubeconfig)
	}
}

func TestProviderGetConfigUsesDefaultKubeconfigForRefresh(t *testing.T) {
	defaultKubeconfig := filepath.Join(t.TempDir(), ".kube", "config")
	if err := os.MkdirAll(filepath.Dir(defaultKubeconfig), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(defaultKubeconfig, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	originalDefaultKubeConfigPath := helmDefaultKubeConfigPath
	helmDefaultKubeConfigPath = func() string {
		return defaultKubeconfig
	}
	t.Cleanup(func() {
		helmDefaultKubeConfigPath = originalDefaultKubeConfigPath
	})
	t.Setenv("KUBE_CONFIG_PATH", "")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBE_CTX", "")
	t.Setenv("HELM_KUBECONTEXT", "")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBE_CONFIG_PATH"); got != defaultKubeconfig {
		t.Fatalf("KUBE_CONFIG_PATH = %q, want default kubeconfig %q", got, defaultKubeconfig)
	}
	if got := os.Getenv("KUBECONFIG"); got != defaultKubeconfig {
		t.Fatalf("KUBECONFIG = %q, want default kubeconfig %q", got, defaultKubeconfig)
	}
}

func TestProviderGetConfigPreservesProviderKubeconfigEnv(t *testing.T) {
	t.Setenv("KUBE_CONFIG_PATH", "/tmp/provider-kubeconfig")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "/tmp/client-go-kubeconfig")
	t.Setenv("KUBE_CTX", "provider-context")
	t.Setenv("HELM_KUBECONTEXT", "helm-context")

	provider := &Provider{}
	provider.GetConfig()

	if got := os.Getenv("KUBE_CONFIG_PATH"); got != "/tmp/provider-kubeconfig" {
		t.Fatalf("KUBE_CONFIG_PATH = %q, want existing provider env", got)
	}
	if got := os.Getenv("KUBECONFIG"); got != "/tmp/provider-kubeconfig" {
		t.Fatalf("KUBECONFIG = %q, want provider kubeconfig env to take discovery precedence", got)
	}
	if got := os.Getenv("KUBE_CTX"); got != "provider-context" {
		t.Fatalf("KUBE_CTX = %q, want existing provider context", got)
	}
	if got := os.Getenv("HELM_KUBECONTEXT"); got != "provider-context" {
		t.Fatalf("HELM_KUBECONTEXT = %q, want provider context to take discovery precedence", got)
	}
}
