// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
	"k8s.io/client-go/tools/clientcmd"
)

type Provider struct {
	terraformutils.Provider
}

var helmDefaultKubeConfigPath = func() string {
	return clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
}

var helmKubeEnvPairs = []struct {
	providerEnv  string
	discoveryEnv string
}{
	{providerEnv: "KUBE_CTX", discoveryEnv: "HELM_KUBECONTEXT"},
	{providerEnv: "KUBE_HOST", discoveryEnv: "HELM_KUBEAPISERVER"},
	{providerEnv: "KUBE_INSECURE", discoveryEnv: "HELM_KUBEINSECURE_SKIP_TLS_VERIFY"},
	{providerEnv: "KUBE_TLS_SERVER_NAME", discoveryEnv: "HELM_KUBETLS_SERVER_NAME"},
	{providerEnv: "KUBE_TOKEN", discoveryEnv: "HELM_KUBETOKEN"},
}

func (p *Provider) Init(_ []string) error {
	return nil
}

func (p *Provider) GetName() string {
	return "helm"
}

func (p *Provider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (Provider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *Provider) GetConfig() cty.Value {
	configureHelmProviderKubeEnv()
	return cty.EmptyObjectVal
}

func (p *Provider) GetBasicConfig() cty.Value {
	return p.GetConfig()
}

func (p *Provider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return fmt.Errorf("helm: %s not supported service", serviceName)
	}
	return nil
}

func (p *Provider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"release": &ReleaseGenerator{},
	}
}

func (p *Provider) ValidateImport(resources []string) error {
	for _, resource := range resources {
		if _, ok := p.GetSupportedService()[resource]; !ok {
			return fmt.Errorf("helm: %s not supported service", resource)
		}
	}
	return nil
}

func configureHelmProviderKubeEnv() {
	configureHelmProviderKubeConfigPathEnv()
	for _, pair := range helmKubeEnvPairs {
		mirrorHelmKubeEnvPair(pair.providerEnv, pair.discoveryEnv)
	}
}

func configureHelmProviderKubeConfigPathEnv() {
	providerPaths := helmProviderKubeConfigPaths()
	if len(providerPaths) > 0 {
		_ = os.Setenv("KUBECONFIG", strings.Join(providerPaths, string(os.PathListSeparator)))
		return
	}

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		paths := splitNonEmptyPathList(kubeconfig)
		switch len(paths) {
		case 0:
			return
		case 1:
			_ = os.Setenv("KUBE_CONFIG_PATH", paths[0])
		default:
			_ = os.Setenv("KUBE_CONFIG_PATHS", strings.Join(paths, string(os.PathListSeparator)))
		}
		return
	}

	defaultPath := helmDefaultKubeConfigPath()
	if defaultPath == "" {
		return
	}
	if _, err := os.Stat(defaultPath); err == nil {
		_ = os.Setenv("KUBE_CONFIG_PATH", defaultPath)
		if os.Getenv("KUBECONFIG") == "" {
			_ = os.Setenv("KUBECONFIG", defaultPath)
		}
	}
}

func helmProviderKubeConfigPaths() []string {
	if value := os.Getenv("KUBE_CONFIG_PATHS"); value != "" {
		return splitNonEmptyPathList(value)
	}
	if value := os.Getenv("KUBE_CONFIG_PATH"); value != "" {
		return []string{value}
	}
	return nil
}

func mirrorHelmKubeEnvPair(providerEnv, discoveryEnv string) {
	providerValue := os.Getenv(providerEnv)
	discoveryValue := os.Getenv(discoveryEnv)
	switch {
	case providerValue != "":
		_ = os.Setenv(discoveryEnv, providerValue)
	case providerValue == "" && discoveryValue != "":
		_ = os.Setenv(providerEnv, discoveryValue)
	}
}

func splitNonEmptyPathList(value string) []string {
	parts := filepath.SplitList(value)
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			paths = append(paths, part)
		}
	}
	return paths
}
