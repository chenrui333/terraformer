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

func TestSecretDataInitResources(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "app-secret",
			},
			Data: map[string][]byte{
				"empty":   {},
				"setting": []byte("example-value"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "no-data",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "owned-secret",
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       "owner",
					UID:        "owner-uid",
				}},
			},
			Data: map[string][]byte{"setting": []byte("example-value")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "binary-secret",
			},
			Data: map[string][]byte{"payload": {0xff, 0xfe}},
		},
	)

	service := &SecretData{}
	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	resource := service.Resources[0]
	if resource.InstanceInfo.Type != secretDataTerraformType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, secretDataTerraformType)
	}
	if resource.InstanceState.ID != "default/app-secret" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "default/app-secret")
	}
	if resource.ResourceName != "tfer--default-002F-app-secret" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--default-002F-app-secret")
	}

	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"id":                   "default/app-secret",
		"metadata.#":           "1",
		"metadata.0.name":      "app-secret",
		"metadata.0.namespace": "default",
		"data.%":               "2",
		"data.empty":           "",
		"data.setting":         "example-value",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != secretDataAllowEmptyPattern {
		t.Fatalf("AllowEmptyValues = %#v, want %#v", resource.AllowEmptyValues, []string{secretDataAllowEmptyPattern})
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(attributes, secretDataTestBlock().ImpliedType()); err != nil {
		t.Fatalf("secret data flatmap does not decode into provider schema: %v", err)
	}
}

func TestSecretDataInitResourcesUsesConfiguredTerraformType(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "app-secret",
		},
		Data: map[string][]byte{"setting": []byte("example-value")},
	})
	service := &SecretData{TerraformType: "custom_secret_data"}

	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	if service.Resources[0].InstanceInfo.Type != "custom_secret_data" {
		t.Fatalf("resource type = %q, want %q", service.Resources[0].InstanceInfo.Type, "custom_secret_data")
	}
}

func secretDataTestBlock() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Computed: true,
			},
			"data": {
				Type:      cty.Map(cty.String),
				Required:  true,
				Sensitive: true,
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
