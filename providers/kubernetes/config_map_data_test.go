// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/zclconf/go-cty/cty"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConfigMapDataInitResources(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "app-config",
			},
			Data: map[string]string{
				"empty": "",
				"key":   "value",
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "no-data",
			},
		},
	)

	service := &ConfigMapData{}
	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	resource := service.Resources[0]
	if resource.InstanceInfo.Type != configMapDataTerraformType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, configMapDataTerraformType)
	}
	if resource.InstanceState.ID != "default/app-config" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "default/app-config")
	}
	if resource.ResourceName != "tfer--default-002F-app-config" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--default-002F-app-config")
	}

	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"id":                   "default/app-config",
		"metadata.#":           "1",
		"metadata.0.name":      "app-config",
		"metadata.0.namespace": "default",
		"data.%":               "2",
		"data.empty":           "",
		"data.key":             "value",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != configMapDataAllowEmptyPattern {
		t.Fatalf("AllowEmptyValues = %#v, want %#v", resource.AllowEmptyValues, []string{configMapDataAllowEmptyPattern})
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(attributes, configMapDataTestBlock().ImpliedType()); err != nil {
		t.Fatalf("config map data flatmap does not decode into provider schema: %v", err)
	}
}

func TestConfigMapDataInitResourcesUsesConfiguredTerraformType(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "app-config",
		},
		Data: map[string]string{"key": "value"},
	})
	service := &ConfigMapData{TerraformType: "custom_config_map_data"}

	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	if service.Resources[0].InstanceInfo.Type != "custom_config_map_data" {
		t.Fatalf("resource type = %q, want %q", service.Resources[0].InstanceInfo.Type, "custom_config_map_data")
	}
}

func configMapDataTestBlock() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Computed: true,
			},
			"data": {
				Type:     cty.Map(cty.String),
				Required: true,
			},
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
