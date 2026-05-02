// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesProviderInitHandlesMissingVerboseArg(t *testing.T) {
	provider := KubernetesProvider{verbose: "true"}

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.verbose != "" {
		t.Fatalf("verbose = %q, want empty", provider.verbose)
	}
}

func TestKubernetesProviderInitStoresVerboseArg(t *testing.T) {
	var provider KubernetesProvider

	if err := provider.Init([]string{"true"}); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.verbose != "true" {
		t.Fatalf("verbose = %q, want true", provider.verbose)
	}
}

func TestAddDefaultServiceAccountService(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "ServiceAccount"}: {},
	}

	addDefaultServiceAccountService(resources, clientset, listableResources, func(name string) bool {
		return name == "kubernetes_default_service_account_v1"
	})

	service, ok := resources[defaultServiceAccountServiceName]
	if !ok {
		t.Fatalf("resources[%q] was not registered", defaultServiceAccountServiceName)
	}
	defaultServiceAccount, ok := service.(*DefaultServiceAccount)
	if !ok {
		t.Fatalf("service type = %T, want *DefaultServiceAccount", service)
	}
	if defaultServiceAccount.TerraformType != "kubernetes_default_service_account_v1" {
		t.Fatalf("TerraformType = %q, want %q", defaultServiceAccount.TerraformType, "kubernetes_default_service_account_v1")
	}
}

func TestAddDefaultServiceAccountServiceRequiresProviderType(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "ServiceAccount"}: {},
	}

	addDefaultServiceAccountService(resources, clientset, listableResources, func(string) bool {
		return false
	})

	if _, ok := resources[defaultServiceAccountServiceName]; ok {
		t.Fatalf("resources[%q] was registered without provider type support", defaultServiceAccountServiceName)
	}
}

func TestAddDefaultServiceAccountServiceRequiresServiceAccountAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()

	addDefaultServiceAccountService(resources, clientset, map[kubernetesResourceID]struct{}{}, func(name string) bool {
		return name == "kubernetes_default_service_account_v1"
	})

	if _, ok := resources[defaultServiceAccountServiceName]; ok {
		t.Fatalf("resources[%q] was registered without serviceaccounts API support", defaultServiceAccountServiceName)
	}
}

func TestAddNodeTaintService(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "Node"}: {},
	}

	addNodeTaintService(resources, clientset, listableResources, func(name string) bool {
		return name == nodeTaintTerraformType
	})

	service, ok := resources[nodeTaintServiceName]
	if !ok {
		t.Fatalf("resources[%q] was not registered", nodeTaintServiceName)
	}
	nodeTaint, ok := service.(*NodeTaint)
	if !ok {
		t.Fatalf("service type = %T, want *NodeTaint", service)
	}
	if nodeTaint.TerraformType != nodeTaintTerraformType {
		t.Fatalf("TerraformType = %q, want %q", nodeTaint.TerraformType, nodeTaintTerraformType)
	}
}

func TestAddNodeTaintServiceRequiresProviderType(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "Node"}: {},
	}

	addNodeTaintService(resources, clientset, listableResources, func(string) bool {
		return false
	})

	if _, ok := resources[nodeTaintServiceName]; ok {
		t.Fatalf("resources[%q] was registered without provider type support", nodeTaintServiceName)
	}
}

func TestAddNodeTaintServiceRequiresNodeAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()

	addNodeTaintService(resources, clientset, map[kubernetesResourceID]struct{}{}, func(name string) bool {
		return name == nodeTaintTerraformType
	})

	if _, ok := resources[nodeTaintServiceName]; ok {
		t.Fatalf("resources[%q] was registered without nodes API support", nodeTaintServiceName)
	}
}

func TestAddConfigMapDataService(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "ConfigMap"}: {},
	}

	addConfigMapDataService(resources, clientset, listableResources, func(name string) bool {
		return name == configMapDataTerraformType
	})

	service, ok := resources[configMapDataServiceName]
	if !ok {
		t.Fatalf("resources[%q] was not registered", configMapDataServiceName)
	}
	configMapData, ok := service.(*ConfigMapData)
	if !ok {
		t.Fatalf("service type = %T, want *ConfigMapData", service)
	}
	if configMapData.TerraformType != configMapDataTerraformType {
		t.Fatalf("TerraformType = %q, want %q", configMapData.TerraformType, configMapDataTerraformType)
	}
}

func TestAddConfigMapDataServiceRequiresProviderType(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "ConfigMap"}: {},
	}

	addConfigMapDataService(resources, clientset, listableResources, func(string) bool {
		return false
	})

	if _, ok := resources[configMapDataServiceName]; ok {
		t.Fatalf("resources[%q] was registered without provider type support", configMapDataServiceName)
	}
}

func TestAddConfigMapDataServiceRequiresConfigMapAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()

	addConfigMapDataService(resources, clientset, map[kubernetesResourceID]struct{}{}, func(name string) bool {
		return name == configMapDataTerraformType
	})

	if _, ok := resources[configMapDataServiceName]; ok {
		t.Fatalf("resources[%q] was registered without configmaps API support", configMapDataServiceName)
	}
}

func TestAddSecretDataService(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "Secret"}: {},
	}

	addSecretDataService(resources, clientset, listableResources, func(name string) bool {
		return name == secretDataTerraformType
	})

	service, ok := resources[secretDataServiceName]
	if !ok {
		t.Fatalf("resources[%q] was not registered", secretDataServiceName)
	}
	secretData, ok := service.(*SecretData)
	if !ok {
		t.Fatalf("service type = %T, want *SecretData", service)
	}
	if secretData.TerraformType != secretDataTerraformType {
		t.Fatalf("TerraformType = %q, want %q", secretData.TerraformType, secretDataTerraformType)
	}
}

func TestAddSecretDataServiceRequiresProviderType(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()
	listableResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "Secret"}: {},
	}

	addSecretDataService(resources, clientset, listableResources, func(string) bool {
		return false
	})

	if _, ok := resources[secretDataServiceName]; ok {
		t.Fatalf("resources[%q] was registered without provider type support", secretDataServiceName)
	}
}

func TestAddSecretDataServiceRequiresSecretAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	clientset := fake.NewSimpleClientset()

	addSecretDataService(resources, clientset, map[kubernetesResourceID]struct{}{}, func(name string) bool {
		return name == secretDataTerraformType
	})

	if _, ok := resources[secretDataServiceName]; ok {
		t.Fatalf("resources[%q] was registered without secrets API support", secretDataServiceName)
	}
}

func TestAddKubernetesResourceServiceDisambiguatesManifestPluralCollisions(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	resource := metav1.APIResource{
		Name:       "widgets",
		Kind:       "Widget",
		Namespaced: true,
	}

	addKubernetesResourceService(resources, "example.com", "v1", resource, manifestTerraformResourceName, true)
	addKubernetesResourceService(resources, "other.example.com", "v1", resource, manifestTerraformResourceName, true)

	if len(resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(resources))
	}
	for _, key := range []string{"example.com/v1/widgets", "other.example.com/v1/widgets"} {
		service, ok := resources[key]
		if !ok {
			t.Fatalf("resources[%q] was not registered", key)
		}
		kind := service.(*Kind)
		if kind.ResourceName != "widgets" {
			t.Fatalf("ResourceName = %q, want %q", kind.ResourceName, "widgets")
		}
		if kind.TerraformType != manifestTerraformResourceName {
			t.Fatalf("TerraformType = %q, want %q", kind.TerraformType, manifestTerraformResourceName)
		}
	}
}

func TestAddKubernetesResourceServiceKeepsNativePluralKey(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	resource := metav1.APIResource{
		Name: "services",
		Kind: "Service",
	}

	addKubernetesResourceService(resources, "", "v1", resource, "kubernetes_service_v1", false)

	if _, ok := resources["services"]; !ok {
		t.Fatal("native resource was not registered with its plural key")
	}
	if _, ok := resources["v1/services"]; ok {
		t.Fatal("native resource was registered with a manifest-style qualified key")
	}
}

func TestPostProcessImportResourcesRemovesOnlyImportedDefaultServiceAccounts(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"serviceaccounts": {
			terraformutils.NewSimpleResource("ns-a/default", "ns-a/default", "kubernetes_service_account_v1", "kubernetes", nil),
			terraformutils.NewSimpleResource("ns-b/default", "ns-b/default", "kubernetes_service_account_v1", "kubernetes", nil),
			terraformutils.NewSimpleResource("ns-a/builder", "ns-a/builder", "kubernetes_service_account_v1", "kubernetes", nil),
		},
		defaultServiceAccountServiceName: {
			terraformutils.NewSimpleResource("ns-a/default", "ns-a/default", "kubernetes_default_service_account_v1", "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got["serviceaccounts"], []string{"ns-b/default", "ns-a/builder"})
	assertResourceIDs(t, got[defaultServiceAccountServiceName], []string{"ns-a/default"})
}

func TestPostProcessImportResourcesKeepsServiceAccountsWithoutDefaultServiceAccountImport(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"serviceaccounts": {
			terraformutils.NewSimpleResource("ns-a/default", "ns-a/default", "kubernetes_service_account_v1", "kubernetes", nil),
			terraformutils.NewSimpleResource("ns-a/builder", "ns-a/builder", "kubernetes_service_account_v1", "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got["serviceaccounts"], []string{"ns-a/default", "ns-a/builder"})
}

func TestPostProcessImportResourcesDoesNotAddServiceAccountsService(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		defaultServiceAccountServiceName: {
			terraformutils.NewSimpleResource("ns-a/default", "ns-a/default", "kubernetes_default_service_account_v1", "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got["serviceaccounts"]; ok {
		t.Fatal("serviceaccounts service was added")
	}
}

func TestPostProcessImportResourcesRemovesOverlappingConfigMapData(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"configmaps": {
			terraformutils.NewSimpleResource("default/app-config", "default/app-config", "kubernetes_config_map_v1", "kubernetes", nil),
		},
		configMapDataServiceName: {
			terraformutils.NewSimpleResource("default/app-config", "default/app-config", configMapDataTerraformType, "kubernetes", nil),
			terraformutils.NewSimpleResource("default/other-config", "default/other-config", configMapDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got["configmaps"], []string{"default/app-config"})
	assertResourceIDs(t, got[configMapDataServiceName], []string{"default/other-config"})
}

func TestPostProcessImportResourcesKeepsConfigMapDataWithoutConfigMapsImport(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		configMapDataServiceName: {
			terraformutils.NewSimpleResource("default/app-config", "default/app-config", configMapDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got[configMapDataServiceName], []string{"default/app-config"})
}

func TestPostProcessImportResourcesRemovesConfigMapDataServiceWhenAllOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"configmaps": {
			terraformutils.NewSimpleResource("default/app-config", "default/app-config", "kubernetes_config_map_v1", "kubernetes", nil),
		},
		configMapDataServiceName: {
			terraformutils.NewSimpleResource("default/app-config", "default/app-config", configMapDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[configMapDataServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after all entries overlapped configmaps", configMapDataServiceName)
	}
}

func TestPostProcessImportResourcesRemovesOverlappingSecretData(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"secrets": {
			terraformutils.NewSimpleResource("default/app-secret", "default/app-secret", "kubernetes_secret_v1", "kubernetes", nil),
		},
		secretDataServiceName: {
			terraformutils.NewSimpleResource("default/app-secret", "default/app-secret", secretDataTerraformType, "kubernetes", nil),
			terraformutils.NewSimpleResource("default/other-secret", "default/other-secret", secretDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got["secrets"], []string{"default/app-secret"})
	assertResourceIDs(t, got[secretDataServiceName], []string{"default/other-secret"})
}

func TestPostProcessImportResourcesKeepsSecretDataWithoutSecretsImport(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		secretDataServiceName: {
			terraformutils.NewSimpleResource("default/app-secret", "default/app-secret", secretDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got[secretDataServiceName], []string{"default/app-secret"})
}

func TestPostProcessImportResourcesRemovesSecretDataServiceWhenAllOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	resourcesByService := map[string][]terraformutils.Resource{
		"secrets": {
			terraformutils.NewSimpleResource("default/app-secret", "default/app-secret", "kubernetes_secret_v1", "kubernetes", nil),
		},
		secretDataServiceName: {
			terraformutils.NewSimpleResource("default/app-secret", "default/app-secret", secretDataTerraformType, "kubernetes", nil),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[secretDataServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after all entries overlapped secrets", secretDataServiceName)
	}
}

func assertResourceIDs(t *testing.T, resources []terraformutils.Resource, want []string) {
	t.Helper()

	if len(resources) != len(want) {
		t.Fatalf("Resources len = %d, want %d", len(resources), len(want))
	}

	seen := map[string]bool{}
	for _, resource := range resources {
		seen[resource.InstanceState.ID] = true
	}
	for _, id := range want {
		if !seen[id] {
			t.Fatalf("resource ID %q was not imported", id)
		}
	}
}
