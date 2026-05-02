// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestInitDynamicResources(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "apiregistration.k8s.io",
		Version:  "v1",
		Resource: "apiservices",
	}
	apiService := newUnstructured("apiregistration.k8s.io/v1", "APIService", "v1.apps.example.com", "")
	ownedAPIService := newUnstructured("apiregistration.k8s.io/v1", "APIService", "owned.example.com", "")
	ownedAPIService.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       "owner",
		UID:        "owner-uid",
	}})

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "APIServiceList"},
		apiService,
		ownedAPIService,
	)

	kind := &Kind{
		Group:         "apiregistration.k8s.io",
		Version:       "v1",
		Name:          "APIService",
		ResourceName:  "apiservices",
		TerraformType: "kubernetes_api_service_v1",
	}
	if err := kind.initDynamicResources(client); err != nil {
		t.Fatalf("initDynamicResources() error = %v", err)
	}

	if len(kind.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(kind.Resources))
	}
	resource := kind.Resources[0]
	if resource.InstanceState.ID != "v1.apps.example.com" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "v1.apps.example.com")
	}
	if resource.InstanceInfo.Type != "kubernetes_api_service_v1" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "kubernetes_api_service_v1")
	}
}

func TestInitDynamicResourcesNamespaced(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "widgets",
	}
	widget := newUnstructured("example.com/v1", "Widget", "sample", "default")
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "WidgetList"},
		widget,
	)

	kind := &Kind{
		Group:        "example.com",
		Version:      "v1",
		Name:         "Widget",
		ResourceName: "widgets",
		Namespaced:   true,
	}
	if err := kind.initDynamicResources(client); err != nil {
		t.Fatalf("initDynamicResources() error = %v", err)
	}

	if len(kind.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(kind.Resources))
	}
	resource := kind.Resources[0]
	if resource.InstanceState.ID != "default/sample" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "default/sample")
	}
	if resource.InstanceInfo.Type != "kubernetes_widget" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "kubernetes_widget")
	}
}

func TestInitDynamicResourcesManifestImportID(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1",
		Resource: "widgets",
	}
	widget := newUnstructured("example.com/v1", "Widget", "sample", "default")
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "WidgetList"},
		widget,
	)

	kind := &Kind{
		Group:         "example.com",
		Version:       "v1",
		Name:          "Widget",
		ResourceName:  "widgets",
		Namespaced:    true,
		TerraformType: manifestTerraformResourceName,
	}
	if err := kind.initDynamicResources(client); err != nil {
		t.Fatalf("initDynamicResources() error = %v", err)
	}

	if len(kind.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(kind.Resources))
	}
	resource := kind.Resources[0]
	if resource.InstanceState.ID != "apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample")
	}
	wantResourceName := terraformutils.TfSanitize("example.com/v1/Widget/default/sample")
	if resource.ResourceName != wantResourceName {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, wantResourceName)
	}
	if resource.InstanceInfo.Type != manifestTerraformResourceName {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, manifestTerraformResourceName)
	}
}

func TestInitDynamicResourcesManifestImportIDClusterScoped(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}
	crd := newUnstructured("apiextensions.k8s.io/v1", "CustomResourceDefinition", "widgets.example.com", "")
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "CustomResourceDefinitionList"},
		crd,
	)

	kind := &Kind{
		Group:         "apiextensions.k8s.io",
		Version:       "v1",
		Name:          "CustomResourceDefinition",
		ResourceName:  "customresourcedefinitions",
		TerraformType: manifestTerraformResourceName,
	}
	if err := kind.initDynamicResources(client); err != nil {
		t.Fatalf("initDynamicResources() error = %v", err)
	}

	if len(kind.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(kind.Resources))
	}
	if kind.Resources[0].InstanceState.ID != "apiVersion=apiextensions.k8s.io/v1,kind=CustomResourceDefinition,name=widgets.example.com" {
		t.Fatalf("resource ID = %q, want %q", kind.Resources[0].InstanceState.ID, "apiVersion=apiextensions.k8s.io/v1,kind=CustomResourceDefinition,name=widgets.example.com")
	}
	wantResourceName := terraformutils.TfSanitize("apiextensions.k8s.io/v1/CustomResourceDefinition/widgets.example.com")
	if kind.Resources[0].ResourceName != wantResourceName {
		t.Fatalf("resource name = %q, want %q", kind.Resources[0].ResourceName, wantResourceName)
	}
}

func TestManifestResourceNameIncludesKind(t *testing.T) {
	widget := newUnstructured("example.com/v1", "Widget", "sample", "default")
	gadget := newUnstructured("example.com/v1", "Gadget", "sample", "default")
	kind := &Kind{
		Group:         "example.com",
		Version:       "v1",
		Name:          "Widget",
		Namespaced:    true,
		TerraformType: manifestTerraformResourceName,
	}
	widgetName := kind.resourceName(*widget)
	kind.Name = "Gadget"
	gadgetName := kind.resourceName(*gadget)

	if widgetName == gadgetName {
		t.Fatalf("manifest resource names collided: %q", widgetName)
	}
	if widgetName != "example.com/v1/Widget/default/sample" {
		t.Fatalf("widget resource name = %q, want %q", widgetName, "example.com/v1/Widget/default/sample")
	}
	if gadgetName != "example.com/v1/Gadget/default/sample" {
		t.Fatalf("gadget resource name = %q, want %q", gadgetName, "example.com/v1/Gadget/default/sample")
	}
}

func TestInitDynamicResourcesRequiresResourceName(t *testing.T) {
	client := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	kind := &Kind{Name: "APIService"}

	if err := kind.initDynamicResources(client); err == nil {
		t.Fatal("initDynamicResources() error = nil, want error")
	}
}

func newUnstructured(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	resource := &unstructured.Unstructured{}
	resource.SetAPIVersion(apiVersion)
	resource.SetKind(kind)
	resource.SetName(name)
	resource.SetNamespace(namespace)
	return resource
}
