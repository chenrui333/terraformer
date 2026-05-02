// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"errors"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/zclconf/go-cty/cty"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestMetadataPatchInitResourcesLabels(t *testing.T) {
	configMap := newUnstructured("v1", "ConfigMap", "app-config", "default")
	configMap.SetLabels(map[string]string{
		"app":   "demo",
		"empty": "",
	})
	noLabels := newUnstructured("v1", "ConfigMap", "no-labels", "default")
	ownedConfigMap := newUnstructured("v1", "ConfigMap", "owned-config", "default")
	ownedConfigMap.SetLabels(map[string]string{"app": "owned"})
	ownedConfigMap.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       "owner",
		UID:        "owner-uid",
	}})
	namespace := newUnstructured("v1", "Namespace", "team-a", "")
	namespace.SetLabels(map[string]string{"team": "a"})

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			{Version: "v1", Resource: "namespaces"}: "NamespaceList",
		},
		configMap,
		noLabels,
		ownedConfigMap,
		namespace,
	)
	service := &MetadataPatch{
		TerraformType:     labelsTerraformType,
		AttributeName:     "labels",
		AllowEmptyPattern: labelsAllowEmptyPattern,
	}

	if err := service.initResources(client, metadataPatchTestAPIResources()); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 2 {
		t.Fatalf("Resources len = %d, want 2", len(service.Resources))
	}
	resources := resourcesByID(service.Resources)
	resource := resources["apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config"]
	if resource.InstanceInfo.Type != labelsTerraformType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, labelsTerraformType)
	}
	if resource.ResourceName != "tfer--labels-002F-v1-002F-ConfigMap-002F-default-002F-app-config" {
		t.Fatalf("resource name = %q, want ConfigMap labels resource name", resource.ResourceName)
	}
	for key, want := range map[string]string{
		"id":                   "apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config",
		"api_version":          "v1",
		"kind":                 "ConfigMap",
		"metadata.#":           "1",
		"metadata.0.name":      "app-config",
		"metadata.0.namespace": "default",
		"labels.%":             "2",
		"labels.app":           "demo",
		"labels.empty":         "",
	} {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != labelsAllowEmptyPattern {
		t.Fatalf("AllowEmptyValues = %#v, want %#v", resource.AllowEmptyValues, []string{labelsAllowEmptyPattern})
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(resource.InstanceState.Attributes, metadataPatchTestBlock("labels", true).ImpliedType()); err != nil {
		t.Fatalf("labels flatmap does not decode into provider schema: %v", err)
	}

	namespaceResource := resources["apiVersion=v1,kind=Namespace,name=team-a"]
	if _, ok := namespaceResource.InstanceState.Attributes["metadata.0.namespace"]; ok {
		t.Fatal("cluster-scoped metadata resource included namespace attribute")
	}
}

func TestMetadataPatchInitResourcesAnnotations(t *testing.T) {
	configMap := newUnstructured("v1", "ConfigMap", "app-config", "default")
	configMap.SetAnnotations(map[string]string{
		"example.com/owner": "platform",
	})
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			{Version: "v1", Resource: "namespaces"}: "NamespaceList",
		},
		configMap,
	)
	service := &MetadataPatch{
		TerraformType:     annotationsTerraformType,
		AttributeName:     "annotations",
		AllowEmptyPattern: annotationsAllowEmptyPattern,
	}

	if err := service.initResources(client, metadataPatchTestAPIResources()); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	resource := service.Resources[0]
	for key, want := range map[string]string{
		"id":                            "apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config",
		"api_version":                   "v1",
		"kind":                          "ConfigMap",
		"metadata.#":                    "1",
		"metadata.0.name":               "app-config",
		"metadata.0.namespace":          "default",
		"annotations.%":                 "1",
		"annotations.example.com/owner": "platform",
	} {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(resource.InstanceState.Attributes, metadataPatchTestBlock("annotations", false).ImpliedType()); err != nil {
		t.Fatalf("annotations flatmap does not decode into provider schema: %v", err)
	}
}

func TestKubernetesPreferredResourcesKeepsPartialDiscoveryResults(t *testing.T) {
	lists := metadataPatchTestAPIResources()
	got, err := kubernetesPreferredResources(metadataPatchDiscoveryClient{
		lists: lists,
		err: &discovery.ErrGroupDiscoveryFailed{Groups: map[schema.GroupVersion]error{
			{Group: "broken.example.com", Version: "v1"}: errors.New("unavailable"),
		}},
	})
	if err != nil {
		t.Fatalf("kubernetesPreferredResources() error = %v", err)
	}
	if len(got) != len(lists) {
		t.Fatalf("lists len = %d, want %d", len(got), len(lists))
	}
}

func TestKubernetesPreferredResourcesReturnsNonPartialDiscoveryErrors(t *testing.T) {
	if _, err := kubernetesPreferredResources(metadataPatchDiscoveryClient{err: errors.New("boom")}); err == nil {
		t.Fatal("kubernetesPreferredResources() error = nil, want error")
	}
}

func TestMetadataPatchInitResourcesSkipsListFailures(t *testing.T) {
	configMap := newUnstructured("v1", "ConfigMap", "app-config", "default")
	configMap.SetLabels(map[string]string{"app": "demo"})
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Version: "v1", Resource: "configmaps"}: "ConfigMapList",
			{Version: "v1", Resource: "secrets"}:    "SecretList",
		},
		configMap,
	)
	client.PrependReactor("list", "secrets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	service := &MetadataPatch{
		TerraformType:     labelsTerraformType,
		AttributeName:     "labels",
		AllowEmptyPattern: labelsAllowEmptyPattern,
	}

	if err := service.initResources(client, []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			{Name: "secrets", Kind: "Secret", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
		},
	}}); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	assertResourceIDs(t, service.Resources, []string{"apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config"})
}

func TestMetadataPatchSupportsResourceRequiresReadAndPatchVerbs(t *testing.T) {
	tests := []struct {
		name     string
		resource metav1.APIResource
		want     bool
	}{
		{
			name:     "manageable resource",
			resource: metav1.APIResource{Name: "configmaps", Kind: "ConfigMap", Verbs: []string{"get", "list", "patch"}},
			want:     true,
		},
		{
			name:     "list only",
			resource: metav1.APIResource{Name: "widgets", Kind: "Widget", Verbs: []string{"list"}},
		},
		{
			name:     "missing patch",
			resource: metav1.APIResource{Name: "widgets", Kind: "Widget", Verbs: []string{"get", "list"}},
		},
		{
			name:     "missing get",
			resource: metav1.APIResource{Name: "widgets", Kind: "Widget", Verbs: []string{"list", "patch"}},
		},
		{
			name:     "subresource",
			resource: metav1.APIResource{Name: "configmaps/status", Kind: "ConfigMap", Verbs: []string{"get", "list", "patch"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metadataPatchSupportsResource(tt.resource); got != tt.want {
				t.Fatalf("metadataPatchSupportsResource() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestAddMetadataPatchServices(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}

	addMetadataPatchServices(resources, func(name string) bool {
		return name == labelsTerraformType || name == annotationsTerraformType
	})

	if service, ok := resources[labelsServiceName].(*MetadataPatch); !ok || service.TerraformType != labelsTerraformType {
		t.Fatalf("labels service = %#v, want MetadataPatch with %q", resources[labelsServiceName], labelsTerraformType)
	}
	if service, ok := resources[annotationsServiceName].(*MetadataPatch); !ok || service.TerraformType != annotationsTerraformType {
		t.Fatalf("annotations service = %#v, want MetadataPatch with %q", resources[annotationsServiceName], annotationsTerraformType)
	}
}

func TestAddMetadataPatchServicesRequiresProviderTypes(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}

	addMetadataPatchServices(resources, func(string) bool {
		return false
	})

	if len(resources) != 0 {
		t.Fatalf("resources len = %d, want 0", len(resources))
	}
}

func TestPostProcessImportResourcesRemovesOverlappingMetadataPatches(t *testing.T) {
	provider := KubernetesProvider{}
	fullConfigMap := terraformutils.NewResource(
		"default/app-config",
		"default/app-config",
		"kubernetes_config_map_v1",
		"kubernetes",
		map[string]string{
			"metadata.#":           "1",
			"metadata.0.name":      "app-config",
			"metadata.0.namespace": "default",
		},
		nil,
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"configmaps": {fullConfigMap},
		labelsServiceName: {
			metadataPatchTestResource(labelsTerraformType, "apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config"),
			metadataPatchTestResource(labelsTerraformType, "apiVersion=v1,kind=ConfigMap,namespace=default,name=other-config"),
		},
		annotationsServiceName: {
			metadataPatchTestResource(annotationsTerraformType, "apiVersion=v1,kind=ConfigMap,namespace=default,name=app-config"),
			metadataPatchTestResource(annotationsTerraformType, "apiVersion=v1,kind=ConfigMap,namespace=default,name=other-config"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got[labelsServiceName], []string{"apiVersion=v1,kind=ConfigMap,namespace=default,name=other-config"})
	assertResourceIDs(t, got[annotationsServiceName], []string{"apiVersion=v1,kind=ConfigMap,namespace=default,name=other-config"})
}

func TestPostProcessImportResourcesRemovesFallbackNativeMetadataPatchOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	fullWidget := terraformutils.NewResource(
		"default/sample",
		"default/sample",
		"kubernetes_widget",
		"kubernetes",
		map[string]string{
			"metadata.#":           "1",
			"metadata.0.name":      "sample",
			"metadata.0.namespace": "default",
		},
		nil,
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"widgets": {fullWidget},
		labelsServiceName: {
			metadataPatchTestResource(labelsTerraformType, "apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample"),
			metadataPatchTestResource(labelsTerraformType, "apiVersion=example.com/v1,kind=Widget,namespace=default,name=other"),
			metadataPatchTestResource(labelsTerraformType, "apiVersion=example.com/v1,kind=Gadget,namespace=default,name=sample"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got[labelsServiceName], []string{
		"apiVersion=example.com/v1,kind=Widget,namespace=default,name=other",
		"apiVersion=example.com/v1,kind=Gadget,namespace=default,name=sample",
	})
}

func TestPostProcessImportResourcesRemovesManifestMetadataPatchOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	manifest := terraformutils.NewSimpleResource(
		"apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample",
		"example.com/v1/Widget/default/sample",
		manifestTerraformResourceName,
		"kubernetes",
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"example.com/v1/widgets": {manifest},
		labelsServiceName: {
			metadataPatchTestResource(labelsTerraformType, "apiVersion=example.com/v1,kind=Widget,namespace=default,name=sample"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[labelsServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after manifest overlap", labelsServiceName)
	}
}

func TestPostProcessImportResourcesRemovesClusterScopedNativeMetadataPatchOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	namespace := terraformutils.NewResource(
		"team-a",
		"team-a",
		"kubernetes_namespace_v1",
		"kubernetes",
		map[string]string{
			"metadata.#":      "1",
			"metadata.0.name": "team-a",
		},
		nil,
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"namespaces": {namespace},
		labelsServiceName: {
			metadataPatchTestResource(labelsTerraformType, "apiVersion=v1,kind=Namespace,name=team-a"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[labelsServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after cluster-scoped native overlap", labelsServiceName)
	}
}

func TestPostProcessImportResourcesRemovesClusterScopedManifestMetadataPatchOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	manifest := terraformutils.NewSimpleResource(
		"apiVersion=example.com/v1,kind=Widget,name=sample",
		"example.com/v1/Widget/sample",
		manifestTerraformResourceName,
		"kubernetes",
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"example.com/v1/widgets": {manifest},
		labelsServiceName: {
			metadataPatchTestResource(labelsTerraformType, "apiVersion=example.com/v1,kind=Widget,name=sample"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[labelsServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after cluster-scoped manifest overlap", labelsServiceName)
	}
}

func metadataPatchTestAPIResources() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			{Name: "configmaps/status", Kind: "ConfigMap", Namespaced: true, Verbs: []string{"get", "list"}},
			{Name: "secrets", Kind: "Secret", Namespaced: true, Verbs: []string{"get"}},
			{Name: "namespaces", Kind: "Namespace", Verbs: []string{"get", "list", "patch"}},
		},
	}}
}

type metadataPatchDiscoveryClient struct {
	discovery.DiscoveryInterface
	lists []*metav1.APIResourceList
	err   error
}

func (m metadataPatchDiscoveryClient) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return m.lists, m.err
}

func metadataPatchTestBlock(attributeName string, required bool) *configschema.Block {
	metadataAttribute := &configschema.Attribute{
		Type: cty.Map(cty.String),
	}
	if required {
		metadataAttribute.Required = true
	} else {
		metadataAttribute.Optional = true
	}

	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Computed: true,
			},
			"api_version": {
				Type:     cty.String,
				Required: true,
			},
			"kind": {
				Type:     cty.String,
				Required: true,
			},
			attributeName: metadataAttribute,
			"field_manager": {
				Type:     cty.String,
				Optional: true,
			},
			"force": {
				Type:     cty.Bool,
				Optional: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"metadata": {
				Nesting:  configschema.NestingList,
				MinItems: 1,
				MaxItems: 1,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"name": {
							Type:     cty.String,
							Required: true,
						},
						"namespace": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourcesByID(resources []terraformutils.Resource) map[string]terraformutils.Resource {
	byID := map[string]terraformutils.Resource{}
	for _, resource := range resources {
		byID[resource.InstanceState.ID] = resource
	}
	return byID
}

func metadataPatchTestResource(resourceType, id string) terraformutils.Resource {
	return terraformutils.NewResource(
		id,
		id,
		resourceType,
		"kubernetes",
		map[string]string{"id": id},
		nil,
		nil,
	)
}
