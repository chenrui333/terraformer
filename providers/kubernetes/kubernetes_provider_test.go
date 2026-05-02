// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

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
