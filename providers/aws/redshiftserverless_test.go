// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestRedshiftServerlessResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "workgroup", "", "analytics", "reporting"}, want: "workgroup/analytics/reporting"},
		{name: "preserves segment boundaries", parts: []string{"namespace", "analytics"}, want: "namespace/analytics"},
		{name: "fallback", parts: nil, want: redshiftServerlessResourceNameFallback},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redshiftServerlessResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("redshiftServerlessResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRedshiftServerlessResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(redshiftServerlessResourceName("workgroup", "ab", "c"))
	second := terraformutils.TfSanitize(redshiftServerlessResourceName("workgroup", "a", "bc"))
	if first == second {
		t.Fatalf("redshiftServerlessResourceName() collision after sanitize: %q", first)
	}
}

func TestRedshiftServerlessImportIDs(t *testing.T) {
	namespace := redshiftserverlesstypes.Namespace{NamespaceName: redshiftServerlessString("analytics")}
	if got := redshiftServerlessNamespaceImportID(namespace); got != "analytics" {
		t.Fatalf("namespace import ID = %q, want analytics", got)
	}

	workgroup := redshiftserverlesstypes.Workgroup{WorkgroupName: redshiftServerlessString("reporting")}
	if got := redshiftServerlessWorkgroupImportID(workgroup); got != "reporting" {
		t.Fatalf("workgroup import ID = %q, want reporting", got)
	}
}

func TestNewRedshiftServerlessNamespaceResource(t *testing.T) {
	resource, ok := newRedshiftServerlessNamespaceResource(redshiftserverlesstypes.Namespace{
		AdminPasswordSecretArn: redshiftServerlessString("arn:aws:secretsmanager:us-east-1:123456789012:secret:redshift-serverless-admin"),
		NamespaceName:          redshiftServerlessString("analytics"),
		Status:                 redshiftserverlesstypes.NamespaceStatusAvailable,
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"analytics",
		redshiftServerlessResourceName("namespace", "analytics"),
		redshiftServerlessNamespaceResourceType,
	)
	assertRedshiftServerlessAttribute(t, resource, "namespace_name", "analytics")
	assertRedshiftServerlessAttribute(t, resource, "manage_admin_password", "true")

	if _, ok := newRedshiftServerlessNamespaceResource(redshiftserverlesstypes.Namespace{
		Status: redshiftserverlesstypes.NamespaceStatusAvailable,
	}); ok {
		t.Fatal("namespace with empty name should be skipped")
	}
	resource, ok = newRedshiftServerlessNamespaceResource(redshiftserverlesstypes.Namespace{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.NamespaceStatusModifying,
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"analytics",
		redshiftServerlessResourceName("namespace", "analytics"),
		redshiftServerlessNamespaceResourceType,
	)
	if _, ok := newRedshiftServerlessNamespaceResource(redshiftserverlesstypes.Namespace{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.NamespaceStatusDeleting,
	}); ok {
		t.Fatal("deleting namespace should be skipped")
	}
}

func TestNewRedshiftServerlessWorkgroupResource(t *testing.T) {
	enhancedVPCRouting := false
	publiclyAccessible := false
	resource, ok := newRedshiftServerlessWorkgroupResource(redshiftserverlesstypes.Workgroup{
		BaseCapacity:           redshiftServerlessInt32(32),
		EnhancedVpcRouting:     &enhancedVPCRouting,
		MaxCapacity:            redshiftServerlessInt32(128),
		NamespaceName:          redshiftServerlessString("analytics"),
		Port:                   redshiftServerlessInt32(5439),
		PricePerformanceTarget: &redshiftserverlesstypes.PerformanceTarget{Status: redshiftserverlesstypes.PerformanceTargetStatusDisabled, Level: redshiftServerlessInt32(50)},
		PubliclyAccessible:     &publiclyAccessible,
		SecurityGroupIds:       []string{"sg-1", "sg-2"},
		Status:                 redshiftserverlesstypes.WorkgroupStatusAvailable,
		SubnetIds:              []string{"subnet-1", "subnet-2"},
		TrackName:              redshiftServerlessString("current"),
		WorkgroupName:          redshiftServerlessString("reporting"),
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"reporting",
		redshiftServerlessResourceName("workgroup", "analytics", "reporting"),
		redshiftServerlessWorkgroupResourceType,
	)

	expected := map[string]string{
		"base_capacity":                      "32",
		"enhanced_vpc_routing":               "false",
		"max_capacity":                       "128",
		"namespace_name":                     "analytics",
		"port":                               "5439",
		"price_performance_target.#":         "1",
		"price_performance_target.0.enabled": "false",
		"price_performance_target.0.level":   "50",
		"publicly_accessible":                "false",
		"security_group_ids.#":               "2",
		"security_group_ids.1":               "sg-2",
		"subnet_ids.#":                       "2",
		"subnet_ids.1":                       "subnet-2",
		"track_name":                         "current",
		"workgroup_name":                     "reporting",
	}
	for key, want := range expected {
		assertRedshiftServerlessAttribute(t, resource, key, want)
	}
	assertRedshiftServerlessAllowEmptyValue(t, resource, `^price_performance_target\.\d+\.enabled$`)
	assertRedshiftServerlessAllowEmptyValue(t, resource, "^enhanced_vpc_routing$")
	assertRedshiftServerlessAllowEmptyValue(t, resource, "^publicly_accessible$")
}

func TestNewRedshiftServerlessWorkgroupResourceSkipsUnsafeWorkgroups(t *testing.T) {
	validWorkgroup := func() redshiftserverlesstypes.Workgroup {
		return redshiftserverlesstypes.Workgroup{
			NamespaceName: redshiftServerlessString("analytics"),
			Status:        redshiftserverlesstypes.WorkgroupStatusAvailable,
			WorkgroupName: redshiftServerlessString("reporting"),
		}
	}

	tests := []struct {
		name   string
		mutate func(*redshiftserverlesstypes.Workgroup)
	}{
		{name: "empty workgroup name", mutate: func(workgroup *redshiftserverlesstypes.Workgroup) { workgroup.WorkgroupName = nil }},
		{name: "empty namespace name", mutate: func(workgroup *redshiftserverlesstypes.Workgroup) { workgroup.NamespaceName = nil }},
		{name: "deleting status", mutate: func(workgroup *redshiftserverlesstypes.Workgroup) {
			workgroup.Status = redshiftserverlesstypes.WorkgroupStatusDeleting
		}},
		{name: "empty status", mutate: func(workgroup *redshiftserverlesstypes.Workgroup) { workgroup.Status = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workgroup := validWorkgroup()
			tt.mutate(&workgroup)
			if _, ok := newRedshiftServerlessWorkgroupResource(workgroup); ok {
				t.Fatal("workgroup should be skipped")
			}
		})
	}
}

func TestRedshiftServerlessStatusImportability(t *testing.T) {
	if !redshiftServerlessNamespaceImportable(redshiftserverlesstypes.Namespace{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.NamespaceStatusAvailable,
	}) {
		t.Fatal("available namespace should be importable")
	}
	if !redshiftServerlessNamespaceImportable(redshiftserverlesstypes.Namespace{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.NamespaceStatusModifying,
	}) {
		t.Fatal("modifying namespace should be importable")
	}
	for _, status := range []redshiftserverlesstypes.NamespaceStatus{"", redshiftserverlesstypes.NamespaceStatusDeleting} {
		if redshiftServerlessNamespaceImportable(redshiftserverlesstypes.Namespace{
			NamespaceName: redshiftServerlessString("analytics"),
			Status:        status,
		}) {
			t.Fatalf("namespace status %q should not be importable", status)
		}
	}

	if !redshiftServerlessWorkgroupImportable(redshiftserverlesstypes.Workgroup{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.WorkgroupStatusAvailable,
		WorkgroupName: redshiftServerlessString("reporting"),
	}) {
		t.Fatal("available workgroup should be importable")
	}
	for _, status := range []redshiftserverlesstypes.WorkgroupStatus{redshiftserverlesstypes.WorkgroupStatusCreating, redshiftserverlesstypes.WorkgroupStatusModifying} {
		if !redshiftServerlessWorkgroupImportable(redshiftserverlesstypes.Workgroup{
			NamespaceName: redshiftServerlessString("analytics"),
			Status:        status,
			WorkgroupName: redshiftServerlessString("reporting"),
		}) {
			t.Fatalf("workgroup status %q should be importable", status)
		}
	}
	for _, status := range []redshiftserverlesstypes.WorkgroupStatus{"", redshiftserverlesstypes.WorkgroupStatusDeleting} {
		if redshiftServerlessWorkgroupImportable(redshiftserverlesstypes.Workgroup{
			NamespaceName: redshiftServerlessString("analytics"),
			Status:        status,
			WorkgroupName: redshiftServerlessString("reporting"),
		}) {
			t.Fatalf("workgroup status %q should not be importable", status)
		}
	}
}

func assertRedshiftServerlessResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func assertRedshiftServerlessAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}

func assertRedshiftServerlessAllowEmptyValue(t *testing.T, resource terraformutils.Resource, want string) {
	t.Helper()
	for _, value := range resource.AllowEmptyValues {
		if value == want {
			return
		}
	}
	t.Fatalf("AllowEmptyValues = %v, want %q", resource.AllowEmptyValues, want)
}

func redshiftServerlessString(value string) *string {
	return &value
}

func redshiftServerlessInt32(value int32) *int32 {
	return &value
}
