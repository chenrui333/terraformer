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
	if os.Getenv("KUBE_CONFIG_PATH") == "" && os.Getenv("KUBE_CONFIG_PATHS") == "" {
		configureHelmProviderKubeConfigPathEnv()
	}
	if os.Getenv("KUBE_CTX") == "" {
		if context := os.Getenv("HELM_KUBECONTEXT"); context != "" {
			_ = os.Setenv("KUBE_CTX", context)
		}
	}
}

func configureHelmProviderKubeConfigPathEnv() {
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
