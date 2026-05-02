// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAddDefaultServiceAccountService(t *testing.T) {
	serviceAccounts := &Kind{Name: "ServiceAccount"}
	resources := map[string]terraformutils.ServiceGenerator{
		"serviceaccounts": serviceAccounts,
	}
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
	if !serviceAccounts.SkipDefaultServiceAccount {
		t.Fatal("serviceaccounts Kind did not skip default service accounts")
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
