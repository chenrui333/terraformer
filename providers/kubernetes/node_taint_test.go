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

func TestNodeTaintInitResources(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "worker-a"},
			Spec: corev1.NodeSpec{
				Taints: []corev1.Taint{
					{
						Key:    "dedicated",
						Value:  "gpu",
						Effect: corev1.TaintEffectNoSchedule,
					},
					{
						Key:    "node-role.kubernetes.io/control-plane",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "worker-b"},
		},
	)

	service := &NodeTaint{}
	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	resource := service.Resources[0]
	if resource.InstanceInfo.Type != nodeTaintTerraformType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, nodeTaintTerraformType)
	}
	wantID := "worker-a,dedicated=gpu:NoSchedule,node-role.kubernetes.io/control-plane=:NoSchedule"
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.ResourceName != "tfer--worker-a" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--worker-a")
	}

	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"id":             wantID,
		"metadata.#":     "1",
		"taint.#":        "2",
		"taint.0.key":    "dedicated",
		"taint.0.value":  "gpu",
		"taint.0.effect": "NoSchedule",
		"taint.1.key":    "node-role.kubernetes.io/control-plane",
		"taint.1.value":  "",
		"taint.1.effect": "NoSchedule",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != nodeTaintAllowEmptyPattern {
		t.Fatalf("AllowEmptyValues = %#v, want %#v", resource.AllowEmptyValues, []string{nodeTaintAllowEmptyPattern})
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(attributes, nodeTaintTestBlock().ImpliedType()); err != nil {
		t.Fatalf("node taint flatmap does not decode into provider schema: %v", err)
	}
}

func TestNodeTaintInitResourcesUsesConfiguredTerraformType(t *testing.T) {
	clientset := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-a"},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{{
				Key:    "dedicated",
				Value:  "gpu",
				Effect: corev1.TaintEffectNoSchedule,
			}},
		},
	})
	service := &NodeTaint{TerraformType: "custom_node_taint"}

	if err := service.initResources(clientset); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(service.Resources))
	}
	if service.Resources[0].InstanceInfo.Type != "custom_node_taint" {
		t.Fatalf("resource type = %q, want %q", service.Resources[0].InstanceInfo.Type, "custom_node_taint")
	}
}

func nodeTaintTestBlock() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Computed: true,
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
					},
				},
			},
			"taint": {
				Nesting:  configschema.NestingList,
				MinItems: 1,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"effect": {
							Type:     cty.String,
							Required: true,
						},
						"key": {
							Type:     cty.String,
							Required: true,
						},
						"value": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
	}
}
