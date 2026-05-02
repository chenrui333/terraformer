// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultServiceAccountInitResources(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "default",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "default",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "builder",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "owned",
				Name:      "default",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "v1",
					Kind:       "ConfigMap",
					Name:       "owner",
					UID:        "owner-uid",
				}},
			},
		},
	)

	service := &DefaultServiceAccount{
		TerraformType: "kubernetes_default_service_account_v1",
	}
	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 2 {
		t.Fatalf("Resources len = %d, want 2", len(service.Resources))
	}

	wantIDs := map[string]bool{
		"default/default":     false,
		"kube-system/default": false,
	}
	for _, resource := range service.Resources {
		if _, ok := wantIDs[resource.InstanceState.ID]; !ok {
			t.Fatalf("unexpected resource ID %q", resource.InstanceState.ID)
		}
		wantIDs[resource.InstanceState.ID] = true
		if resource.InstanceInfo.Type != "kubernetes_default_service_account_v1" {
			t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "kubernetes_default_service_account_v1")
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Fatalf("resource ID %q was not imported", id)
		}
	}
}

func TestDefaultServiceAccountInitResourcesDefaultTerraformType(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "default",
		},
	})
	service := &DefaultServiceAccount{}

	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	if service.Resources[0].InstanceInfo.Type != "kubernetes_default_service_account" {
		t.Fatalf("resource type = %q, want %q", service.Resources[0].InstanceInfo.Type, "kubernetes_default_service_account")
	}
}

func TestKindSetSelectedResourcesControlsDefaultServiceAccountSkip(t *testing.T) {
	kind := &Kind{Name: "ServiceAccount", Version: "v1"}
	kind.SetSelectedResources([]string{"serviceaccounts"})
	if kind.SkipDefaultServiceAccount {
		t.Fatal("SkipDefaultServiceAccount = true for serviceaccounts-only import")
	}

	kind.SetSelectedResources([]string{"serviceaccounts", defaultServiceAccountServiceName})
	if !kind.SkipDefaultServiceAccount {
		t.Fatal("SkipDefaultServiceAccount = false when defaultserviceaccounts is selected")
	}

	kind.SetSelectedResources([]string{"serviceaccounts"})
	if kind.SkipDefaultServiceAccount {
		t.Fatal("SkipDefaultServiceAccount was not reset after defaultserviceaccounts was removed")
	}
}

func TestServiceAccountKindSkipsDefaultServiceAccounts(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "default",
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "builder",
			},
		},
	)
	kind := &Kind{
		Name:                      "ServiceAccount",
		Version:                   "v1",
		Namespaced:                true,
		TerraformType:             "kubernetes_service_account_v1",
		SkipDefaultServiceAccount: true,
	}

	if err := kind.initTypedResources(clientset); err != nil {
		t.Fatalf("initTypedResources() error = %v", err)
	}

	if len(kind.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(kind.Resources))
	}
	if kind.Resources[0].InstanceState.ID != "default/builder" {
		t.Fatalf("resource ID = %q, want %q", kind.Resources[0].InstanceState.ID, "default/builder")
	}
}
