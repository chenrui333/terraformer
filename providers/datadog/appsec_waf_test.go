// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func TestAppSecWafCustomRuleCreateResource(t *testing.T) {
	data := datadogV2.NewApplicationSecurityWafCustomRuleDataWithDefaults()
	data.SetId("custom-rule-abc")

	generator := &AppSecWafCustomRuleGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "custom-rule-abc" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "custom-rule-abc")
	}
	if resource.InstanceInfo.Type != "datadog_appsec_waf_custom_rule" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_appsec_waf_custom_rule")
	}
}

func TestAppSecWafCustomRuleCreateResourceMissingID(t *testing.T) {
	generator := &AppSecWafCustomRuleGenerator{}
	_, err := generator.createResource(datadogV2.ApplicationSecurityWafCustomRuleData{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestAppSecWafCustomRuleCreateResources(t *testing.T) {
	first := datadogV2.NewApplicationSecurityWafCustomRuleDataWithDefaults()
	first.SetId("rule-1")
	second := datadogV2.NewApplicationSecurityWafCustomRuleDataWithDefaults()
	second.SetId("rule-2")

	generator := &AppSecWafCustomRuleGenerator{}
	resources, err := generator.createResources([]datadogV2.ApplicationSecurityWafCustomRuleData{*first, *second})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
}

func TestAppSecWafCustomRuleCreateResourcesEmpty(t *testing.T) {
	generator := &AppSecWafCustomRuleGenerator{}
	resources, err := generator.createResources(nil)
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 0 {
		t.Fatalf("resource count = %d, want %d", len(resources), 0)
	}
}

func TestAppSecWafExclusionFilterCreateResource(t *testing.T) {
	data := datadogV2.NewApplicationSecurityWafExclusionFilterResourceWithDefaults()
	data.SetId("exclusion-filter-xyz")

	generator := &AppSecWafExclusionFilterGenerator{}
	resource, err := generator.createResource(*data)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "exclusion-filter-xyz" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "exclusion-filter-xyz")
	}
	if resource.InstanceInfo.Type != "datadog_appsec_waf_exclusion_filter" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_appsec_waf_exclusion_filter")
	}
}

func TestAppSecWafExclusionFilterCreateResourceMissingID(t *testing.T) {
	generator := &AppSecWafExclusionFilterGenerator{}
	_, err := generator.createResource(datadogV2.ApplicationSecurityWafExclusionFilterResource{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestAppSecWafExclusionFilterCreateResources(t *testing.T) {
	first := datadogV2.NewApplicationSecurityWafExclusionFilterResourceWithDefaults()
	first.SetId("filter-1")
	second := datadogV2.NewApplicationSecurityWafExclusionFilterResourceWithDefaults()
	second.SetId("filter-2")

	generator := &AppSecWafExclusionFilterGenerator{}
	resources, err := generator.createResources([]datadogV2.ApplicationSecurityWafExclusionFilterResource{*first, *second})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
}
