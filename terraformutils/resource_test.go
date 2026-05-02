// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/providerproto"
	"github.com/zclconf/go-cty/cty"
)

func TestNewResource(t *testing.T) {
	r := NewResource("vpc-123", "my-vpc", "aws_vpc", "aws",
		map[string]string{"cidr": "10.0.0.0/16"},
		[]string{"tags."},
		map[string]interface{}{"extra": "field"},
	)

	if r.InstanceState.ID != "vpc-123" {
		t.Errorf("ID = %q, want %q", r.InstanceState.ID, "vpc-123")
	}
	if r.InstanceInfo.Type != "aws_vpc" {
		t.Errorf("Type = %q, want %q", r.InstanceInfo.Type, "aws_vpc")
	}
	wantName := TfSanitize("my-vpc")
	if r.ResourceName != wantName {
		t.Errorf("ResourceName = %q, want %q", r.ResourceName, wantName)
	}
	if r.Provider != "aws" {
		t.Errorf("Provider = %q, want %q", r.Provider, "aws")
	}
	wantID := "aws_vpc." + wantName
	if r.InstanceInfo.Id != wantID {
		t.Errorf("InstanceInfo.Id = %q, want %q", r.InstanceInfo.Id, wantID)
	}
}

func TestNewResourceSanitizesName(t *testing.T) {
	r := NewResource("id", "my resource/name", "aws_vpc", "aws", nil, nil, nil)
	if r.ResourceName != TfSanitize("my resource/name") {
		t.Errorf("ResourceName = %q, want sanitized form", r.ResourceName)
	}
}

func TestNewSimpleResource(t *testing.T) {
	r := NewSimpleResource("id-1", "name-1", "aws_s3_bucket", "aws", []string{"tags."})
	if r.InstanceState.ID != "id-1" {
		t.Errorf("ID = %q, want %q", r.InstanceState.ID, "id-1")
	}
	if len(r.AllowEmptyValues) != 1 || r.AllowEmptyValues[0] != "tags." {
		t.Errorf("AllowEmptyValues = %v, want [tags.]", r.AllowEmptyValues)
	}
}

func TestGetIDKeyDefault(t *testing.T) {
	r := Resource{
		InstanceState: &tfcompat.InstanceState{
			Attributes: map[string]string{"name": "test"},
		},
	}
	if got := r.GetIDKey(); got != "id" {
		t.Errorf("GetIDKey() = %q, want %q", got, "id")
	}
}

func TestGetIDKeySelfLink(t *testing.T) {
	r := Resource{
		InstanceState: &tfcompat.InstanceState{
			Attributes: map[string]string{"self_link": "https://..."},
		},
	}
	if got := r.GetIDKey(); got != "self_link" {
		t.Errorf("GetIDKey() = %q, want %q", got, "self_link")
	}
}

func TestServiceName(t *testing.T) {
	tests := []struct {
		resourceType string
		provider     string
		want         string
	}{
		{"aws_vpc", "aws", "vpc"},
		{"google_compute_instance", "google", "compute_instance"},
		{"azurerm_resource_group", "azurerm", "resource_group"},
	}

	for _, tc := range tests {
		r := Resource{
			Provider:     tc.provider,
			InstanceInfo: &tfcompat.InstanceInfo{Type: tc.resourceType},
		}
		if got := r.ServiceName(); got != tc.want {
			t.Errorf("ServiceName() for %q = %q, want %q", tc.resourceType, got, tc.want)
		}
	}
}

func TestResourceFilterIsApplicable(t *testing.T) {
	tests := []struct {
		name        string
		filterSvc   string
		serviceName string
		want        bool
	}{
		{"empty filter matches all", "", "any_service", true},
		{"matching service", "vpc", "vpc", true},
		{"non-matching service", "vpc", "subnet", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rf := &ResourceFilter{ServiceName: tc.filterSvc}
			if got := rf.IsApplicable(tc.serviceName); got != tc.want {
				t.Errorf("IsApplicable(%q) = %v, want %v", tc.serviceName, got, tc.want)
			}
		})
	}
}

func TestResourceFilterIsInitial(t *testing.T) {
	rf := &ResourceFilter{FieldPath: "id"}
	if !rf.isInitial() {
		t.Error("isInitial() = false for FieldPath=id")
	}

	rf2 := &ResourceFilter{FieldPath: "name"}
	if rf2.isInitial() {
		t.Error("isInitial() = true for FieldPath=name")
	}
}

func TestResourceFilterByID(t *testing.T) {
	r := Resource{
		Provider: "aws",
		InstanceInfo: &tfcompat.InstanceInfo{
			Type: "aws_vpc",
		},
		InstanceState: &tfcompat.InstanceState{
			ID: "vpc-123",
		},
	}

	rf := &ResourceFilter{
		FieldPath:        "id",
		AcceptableValues: []string{"vpc-123"},
	}
	if !rf.Filter(r) {
		t.Error("Filter should accept resource with matching ID")
	}

	rf2 := &ResourceFilter{
		FieldPath:        "id",
		AcceptableValues: []string{"vpc-999"},
	}
	if rf2.Filter(r) {
		t.Error("Filter should reject resource with non-matching ID")
	}
}

func TestTypedAttributesAsMapFiltersIgnoredTopLevelAttributes(t *testing.T) {
	raw := json.RawMessage(`{
		"id": "apiVersion=example.com/v1,kind=Widget,name=sample",
		"manifest": {
			"apiVersion": "example.com/v1",
			"kind": "Widget"
		},
		"object": {
			"computed": true
		}
	}`)
	attributes, err := typedAttributesAsMap(raw, []*regexp.Regexp{
		regexp.MustCompile("^id$"),
		regexp.MustCompile("^object$"),
	})
	if err != nil {
		t.Fatalf("typedAttributesAsMap() error = %v", err)
	}
	if _, ok := attributes["id"]; ok {
		t.Fatal("id attribute was not filtered")
	}
	if _, ok := attributes["object"]; ok {
		t.Fatal("object attribute was not filtered")
	}
	manifest, ok := attributes["manifest"].(map[string]interface{})
	if !ok {
		t.Fatalf("manifest attribute type = %T, want map[string]interface{}", attributes["manifest"])
	}
	if manifest["apiVersion"] != "example.com/v1" {
		t.Fatalf("manifest.apiVersion = %v, want %q", manifest["apiVersion"], "example.com/v1")
	}
}

func TestConvertTFstateUsesTypedManifestWhenFlatmapHasNoManifest(t *testing.T) {
	resource := Resource{
		InstanceInfo: &tfcompat.InstanceInfo{Type: "kubernetes_manifest"},
		InstanceState: &tfcompat.InstanceState{
			Attributes: map[string]string{
				"id": "apiVersion=example.com/v1,kind=Widget,name=sample",
			},
			TypedAttributes: json.RawMessage("{\"id\":\"apiVersion=example.com/v1,kind=Widget,name=sample\",\"manifest\":{\"apiVersion\":\"example.com/v1\",\"kind\":\"Widget\",\"metadata\":{\"name\":\"sample\"}}}"),
		},
		IgnoreKeys: []string{"^id$"},
	}

	if err := resource.convertTFstate(kubernetesManifestTestSchema()); err != nil {
		t.Fatalf("ConvertTFstate() error = %v", err)
	}

	manifest, ok := resource.Item["manifest"].(map[string]interface{})
	if !ok {
		t.Fatalf("manifest attribute type = %T, want map[string]interface{}", resource.Item["manifest"])
	}
	if manifest["apiVersion"] != "example.com/v1" {
		t.Fatalf("manifest.apiVersion = %v, want %q", manifest["apiVersion"], "example.com/v1")
	}
	if _, ok := resource.Item["id"]; ok {
		t.Fatal("id attribute was not filtered from typed manifest fallback")
	}
}

func TestConvertTFstateKeepsParsedManifestWhenFlatmapHasManifest(t *testing.T) {
	resource := Resource{
		InstanceInfo: &tfcompat.InstanceInfo{Type: "kubernetes_manifest"},
		InstanceState: &tfcompat.InstanceState{
			Attributes: map[string]string{
				"manifest.%":          "2",
				"manifest.apiVersion": "example.com/v1",
				"manifest.kind":       "Widget",
			},
			TypedAttributes: json.RawMessage("{\"manifest\":{\"apiVersion\":\"typed.example.com/v1\",\"kind\":\"Widget\"}}"),
		},
	}

	if err := resource.convertTFstate(kubernetesManifestTestSchema()); err != nil {
		t.Fatalf("ConvertTFstate() error = %v", err)
	}

	manifest := resource.Item["manifest"].(map[string]interface{})
	if manifest["apiVersion"] != "example.com/v1" {
		t.Fatalf("manifest.apiVersion = %v, want flatmap value %q", manifest["apiVersion"], "example.com/v1")
	}
}

func kubernetesManifestTestSchema() *providerproto.GetProviderSchemaResponse {
	return &providerproto.GetProviderSchemaResponse{
		ResourceTypes: map[string]configschema.Schema{
			"kubernetes_manifest": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"manifest": {
							Type:     cty.Map(cty.String),
							Optional: true,
						},
					},
				},
			},
		},
	}
}
