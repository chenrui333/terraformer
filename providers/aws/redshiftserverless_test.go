// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"
	"time"

	redshiftserverlesstypes "github.com/aws/aws-sdk-go-v2/service/redshiftserverless/types"
	"github.com/aws/smithy-go"
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

	snapshot := redshiftserverlesstypes.Snapshot{SnapshotName: redshiftServerlessString("daily")}
	if got := redshiftServerlessSnapshotImportID(snapshot); got != "daily" {
		t.Fatalf("snapshot import ID = %q, want daily", got)
	}

	usageLimit := redshiftserverlesstypes.UsageLimit{UsageLimitId: redshiftServerlessString("usage-limit-id")}
	if got := redshiftServerlessUsageLimitImportID(usageLimit); got != "usage-limit-id" {
		t.Fatalf("usage limit import ID = %q, want usage-limit-id", got)
	}

	endpoint := redshiftserverlesstypes.EndpointAccess{EndpointName: redshiftServerlessString("analytics-endpoint")}
	if got := redshiftServerlessEndpointAccessImportID(endpoint); got != "analytics-endpoint" {
		t.Fatalf("endpoint access import ID = %q, want analytics-endpoint", got)
	}

	association := redshiftserverlesstypes.Association{
		CustomDomainName: redshiftServerlessString("analytics.example.com"),
		WorkgroupName:    redshiftServerlessString("reporting"),
	}
	if got := redshiftServerlessCustomDomainAssociationImportID(association); got != "reporting,analytics.example.com" {
		t.Fatalf("custom domain association import ID = %q, want reporting,analytics.example.com", got)
	}
	if got := redshiftServerlessCustomDomainAssociationImportIDFromParts("", "analytics.example.com"); got != "" {
		t.Fatalf("custom domain association empty import ID = %q, want empty", got)
	}

	resourcePolicy := redshiftserverlesstypes.ResourcePolicy{ResourceArn: redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily")}
	if got := redshiftServerlessResourcePolicyImportID(resourcePolicy); got != "arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily" {
		t.Fatalf("resource policy import ID = %q, want snapshot ARN", got)
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

func TestNewRedshiftServerlessSnapshotResource(t *testing.T) {
	resource, ok := newRedshiftServerlessSnapshotResource(redshiftserverlesstypes.Snapshot{
		NamespaceName:           redshiftServerlessString("analytics"),
		SnapshotName:            redshiftServerlessString("daily"),
		SnapshotRetentionPeriod: redshiftServerlessInt32(7),
		Status:                  redshiftserverlesstypes.SnapshotStatusAvailable,
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"daily",
		redshiftServerlessResourceName("snapshot", "analytics", "daily"),
		redshiftServerlessSnapshotResourceType,
	)
	assertRedshiftServerlessAttribute(t, resource, "namespace_name", "analytics")
	assertRedshiftServerlessAttribute(t, resource, "snapshot_name", "daily")
	assertRedshiftServerlessAttribute(t, resource, "retention_period", "7")

	if _, ok := newRedshiftServerlessSnapshotResource(redshiftserverlesstypes.Snapshot{
		NamespaceName: redshiftServerlessString("analytics"),
		Status:        redshiftserverlesstypes.SnapshotStatusAvailable,
	}); ok {
		t.Fatal("snapshot with empty name should be skipped")
	}
	if _, ok := newRedshiftServerlessSnapshotResource(redshiftserverlesstypes.Snapshot{
		SnapshotName: redshiftServerlessString("daily"),
		Status:       redshiftserverlesstypes.SnapshotStatusAvailable,
	}); ok {
		t.Fatal("snapshot with empty namespace should be skipped")
	}
	if _, ok := newRedshiftServerlessSnapshotResource(redshiftserverlesstypes.Snapshot{
		NamespaceName: redshiftServerlessString("analytics"),
		SnapshotName:  redshiftServerlessString("daily"),
		Status:        redshiftserverlesstypes.SnapshotStatusCreating,
	}); ok {
		t.Fatal("creating snapshot should be skipped")
	}
}

func TestNewRedshiftServerlessUsageLimitResource(t *testing.T) {
	resource, ok := newRedshiftServerlessUsageLimitResource(redshiftserverlesstypes.UsageLimit{
		Amount:       redshiftServerlessInt64(60),
		BreachAction: redshiftserverlesstypes.UsageLimitBreachActionEmitMetric,
		Period:       redshiftserverlesstypes.UsageLimitPeriodDaily,
		ResourceArn:  redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/reporting"),
		UsageLimitId: redshiftServerlessString("limit-123"),
		UsageType:    redshiftserverlesstypes.UsageLimitUsageTypeServerlessCompute,
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"limit-123",
		redshiftServerlessResourceName("usage-limit", "arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/reporting", "limit-123"),
		redshiftServerlessUsageLimitResourceType,
	)
	expected := map[string]string{
		"amount":        "60",
		"breach_action": "emit-metric",
		"period":        "daily",
		"resource_arn":  "arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/reporting",
		"usage_type":    "serverless-compute",
	}
	for key, want := range expected {
		assertRedshiftServerlessAttribute(t, resource, key, want)
	}

	validUsageLimit := func() redshiftserverlesstypes.UsageLimit {
		return redshiftserverlesstypes.UsageLimit{
			Amount:       redshiftServerlessInt64(60),
			ResourceArn:  redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/reporting"),
			UsageLimitId: redshiftServerlessString("limit-123"),
			UsageType:    redshiftserverlesstypes.UsageLimitUsageTypeServerlessCompute,
		}
	}
	tests := []struct {
		name   string
		mutate func(*redshiftserverlesstypes.UsageLimit)
	}{
		{name: "empty usage limit ID", mutate: func(usageLimit *redshiftserverlesstypes.UsageLimit) { usageLimit.UsageLimitId = nil }},
		{name: "empty resource ARN", mutate: func(usageLimit *redshiftserverlesstypes.UsageLimit) { usageLimit.ResourceArn = nil }},
		{name: "empty amount", mutate: func(usageLimit *redshiftserverlesstypes.UsageLimit) { usageLimit.Amount = nil }},
		{name: "zero amount", mutate: func(usageLimit *redshiftserverlesstypes.UsageLimit) { usageLimit.Amount = redshiftServerlessInt64(0) }},
		{name: "empty usage type", mutate: func(usageLimit *redshiftserverlesstypes.UsageLimit) { usageLimit.UsageType = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usageLimit := validUsageLimit()
			tt.mutate(&usageLimit)
			if _, ok := newRedshiftServerlessUsageLimitResource(usageLimit); ok {
				t.Fatal("usage limit should be skipped")
			}
		})
	}
}

func TestNewRedshiftServerlessEndpointAccessResource(t *testing.T) {
	resource, ok := newRedshiftServerlessEndpointAccessResource(redshiftserverlesstypes.EndpointAccess{
		EndpointName:   redshiftServerlessString("analytics-endpoint"),
		EndpointStatus: redshiftServerlessString("ACTIVE"),
		SubnetIds:      []string{"subnet-1", "subnet-2"},
		VpcSecurityGroups: []redshiftserverlesstypes.VpcSecurityGroupMembership{
			{VpcSecurityGroupId: redshiftServerlessString("sg-1")},
			{VpcSecurityGroupId: redshiftServerlessString("sg-2")},
			{},
		},
		WorkgroupName: redshiftServerlessString("reporting"),
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"analytics-endpoint",
		redshiftServerlessResourceName("endpoint-access", "reporting", "analytics-endpoint"),
		redshiftServerlessEndpointAccessResourceType,
	)
	expected := map[string]string{
		"endpoint_name":            "analytics-endpoint",
		"subnet_ids.#":             "2",
		"subnet_ids.1":             "subnet-2",
		"vpc_security_group_ids.#": "2",
		"vpc_security_group_ids.1": "sg-2",
		"workgroup_name":           "reporting",
	}
	for key, want := range expected {
		assertRedshiftServerlessAttribute(t, resource, key, want)
	}

	validEndpoint := func() redshiftserverlesstypes.EndpointAccess {
		return redshiftserverlesstypes.EndpointAccess{
			EndpointName:   redshiftServerlessString("analytics-endpoint"),
			EndpointStatus: redshiftServerlessString("ACTIVE"),
			SubnetIds:      []string{"subnet-1"},
			WorkgroupName:  redshiftServerlessString("reporting"),
		}
	}
	tests := []struct {
		name   string
		mutate func(*redshiftserverlesstypes.EndpointAccess)
	}{
		{name: "empty endpoint name", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) { endpoint.EndpointName = nil }},
		{name: "empty workgroup name", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) { endpoint.WorkgroupName = nil }},
		{name: "empty subnet IDs", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) { endpoint.SubnetIds = nil }},
		{name: "creating status", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) {
			endpoint.EndpointStatus = redshiftServerlessString("CREATING")
		}},
		{name: "deleting status", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) {
			endpoint.EndpointStatus = redshiftServerlessString("DELETING")
		}},
		{name: "failed status", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) {
			endpoint.EndpointStatus = redshiftServerlessString("FAILED")
		}},
		{name: "empty status", mutate: func(endpoint *redshiftserverlesstypes.EndpointAccess) { endpoint.EndpointStatus = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := validEndpoint()
			tt.mutate(&endpoint)
			if _, ok := newRedshiftServerlessEndpointAccessResource(endpoint); ok {
				t.Fatal("endpoint access should be skipped")
			}
		})
	}
}

func TestNewRedshiftServerlessCustomDomainAssociationResource(t *testing.T) {
	expiry := time.Unix(1700000000, 0)
	resource, ok := newRedshiftServerlessCustomDomainAssociationResource(redshiftserverlesstypes.Association{
		CustomDomainCertificateArn:        redshiftServerlessString("arn:aws:acm:us-east-1:123456789012:certificate/cert-1"),
		CustomDomainCertificateExpiryTime: &expiry,
		CustomDomainName:                  redshiftServerlessString("analytics.example.com"),
		WorkgroupName:                     redshiftServerlessString("reporting"),
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"reporting,analytics.example.com",
		redshiftServerlessResourceName("custom-domain-association", "reporting", "analytics.example.com"),
		redshiftServerlessCustomDomainAssociationResourceType,
	)
	assertRedshiftServerlessAttribute(t, resource, "custom_domain_certificate_arn", "arn:aws:acm:us-east-1:123456789012:certificate/cert-1")
	assertRedshiftServerlessAttribute(t, resource, "custom_domain_name", "analytics.example.com")
	assertRedshiftServerlessAttribute(t, resource, "workgroup_name", "reporting")

	validAssociation := func() redshiftserverlesstypes.Association {
		return redshiftserverlesstypes.Association{
			CustomDomainCertificateArn:        redshiftServerlessString("arn:aws:acm:us-east-1:123456789012:certificate/cert-1"),
			CustomDomainCertificateExpiryTime: &expiry,
			CustomDomainName:                  redshiftServerlessString("analytics.example.com"),
			WorkgroupName:                     redshiftServerlessString("reporting"),
		}
	}
	tests := []struct {
		name   string
		mutate func(*redshiftserverlesstypes.Association)
	}{
		{name: "empty workgroup", mutate: func(association *redshiftserverlesstypes.Association) { association.WorkgroupName = nil }},
		{name: "empty custom domain", mutate: func(association *redshiftserverlesstypes.Association) { association.CustomDomainName = nil }},
		{name: "empty certificate ARN", mutate: func(association *redshiftserverlesstypes.Association) { association.CustomDomainCertificateArn = nil }},
		{name: "empty certificate expiry", mutate: func(association *redshiftserverlesstypes.Association) {
			association.CustomDomainCertificateExpiryTime = nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			association := validAssociation()
			tt.mutate(&association)
			if _, ok := newRedshiftServerlessCustomDomainAssociationResource(association); ok {
				t.Fatal("custom domain association should be skipped")
			}
		})
	}
}

func TestNewRedshiftServerlessResourcePolicyResource(t *testing.T) {
	policy := `{"Version":"2012-10-17","Statement":[]}`
	resource, ok := newRedshiftServerlessResourcePolicyResource(redshiftserverlesstypes.ResourcePolicy{
		Policy:      redshiftServerlessString(policy),
		ResourceArn: redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily"),
	})
	assertRedshiftServerlessResource(
		t,
		resource,
		ok,
		"arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily",
		redshiftServerlessResourceName("resource-policy", "arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily"),
		redshiftServerlessResourcePolicyResourceType,
	)
	assertRedshiftServerlessAttribute(t, resource, "policy", policy)
	assertRedshiftServerlessAttribute(t, resource, "resource_arn", "arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily")

	if _, ok := newRedshiftServerlessResourcePolicyResource(redshiftserverlesstypes.ResourcePolicy{
		Policy: redshiftServerlessString(policy),
	}); ok {
		t.Fatal("resource policy with empty ARN should be skipped")
	}
	if _, ok := newRedshiftServerlessResourcePolicyResource(redshiftserverlesstypes.ResourcePolicy{
		ResourceArn: redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily"),
	}); ok {
		t.Fatal("resource policy with empty policy should be skipped")
	}
}

func TestRedshiftServerlessPostConvertHookWrapsResourcePolicy(t *testing.T) {
	policy := "{\"Resource\":\"" + "$" + "{aws:username}\"}"
	resource, ok := newRedshiftServerlessResourcePolicyResource(redshiftserverlesstypes.ResourcePolicy{
		Policy:      redshiftServerlessString(policy),
		ResourceArn: redshiftServerlessString("arn:aws:redshift-serverless:us-east-1:123456789012:snapshot/analytics/daily"),
	})
	if !ok {
		t.Fatal("resource policy was skipped")
	}
	resource.Item = map[string]interface{}{"policy": policy}
	g := &RedshiftServerlessGenerator{}
	g.Resources = []terraformutils.Resource{resource}
	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:username}\"}\nPOLICY"
	if got := g.Resources[0].Item["policy"]; got != want {
		t.Fatalf("wrapped policy = %q, want %q", got, want)
	}
}

func TestRedshiftServerlessOptionalResourceLoaderErrors(t *testing.T) {
	g := &RedshiftServerlessGenerator{}
	called := false
	if err := g.loadOptionalResources([]redshiftServerlessOptionalResourceLoader{
		{name: "missing", load: func() error { return &redshiftserverlesstypes.ResourceNotFoundException{} }},
		{name: "next", load: func() error {
			called = true
			return nil
		}},
	}); err != nil {
		t.Fatalf("loadOptionalResources() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("loadOptionalResources() should continue after not found")
	}

	called = false
	if err := g.loadOptionalResources([]redshiftServerlessOptionalResourceLoader{
		{name: "denied", load: func() error { return &redshiftserverlesstypes.AccessDeniedException{} }},
		{name: "next", load: func() error {
			called = true
			return nil
		}},
	}); err != nil {
		t.Fatalf("loadOptionalResources() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("loadOptionalResources() should continue after access denied")
	}

	boom := errors.New("boom")
	called = false
	err := g.loadOptionalResources([]redshiftServerlessOptionalResourceLoader{
		{name: "boom", load: func() error { return boom }},
		{name: "next", load: func() error {
			called = true
			return nil
		}},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
	}
	if called {
		t.Fatal("loadOptionalResources() should stop after unexpected error")
	}
}

func TestRedshiftServerlessOptionalResourceErrorSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed resource not found", err: &redshiftserverlesstypes.ResourceNotFoundException{}, want: true},
		{name: "typed access denied", err: &redshiftserverlesstypes.AccessDeniedException{}, want: true},
		{name: "generic access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "unexpected generic error", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "nil error", err: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redshiftServerlessOptionalResourceErrorSkippable(tt.err); got != tt.want {
				t.Fatalf("redshiftServerlessOptionalResourceErrorSkippable() = %v, want %v", got, tt.want)
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

	if !redshiftServerlessSnapshotImportable(redshiftserverlesstypes.Snapshot{
		NamespaceName: redshiftServerlessString("analytics"),
		SnapshotName:  redshiftServerlessString("daily"),
		Status:        redshiftserverlesstypes.SnapshotStatusAvailable,
	}) {
		t.Fatal("available snapshot should be importable")
	}
	for _, status := range []redshiftserverlesstypes.SnapshotStatus{"", redshiftserverlesstypes.SnapshotStatusCreating, redshiftserverlesstypes.SnapshotStatusCopying, redshiftserverlesstypes.SnapshotStatusDeleted, redshiftserverlesstypes.SnapshotStatusCancelled, redshiftserverlesstypes.SnapshotStatusFailed} {
		if redshiftServerlessSnapshotImportable(redshiftserverlesstypes.Snapshot{
			NamespaceName: redshiftServerlessString("analytics"),
			SnapshotName:  redshiftServerlessString("daily"),
			Status:        status,
		}) {
			t.Fatalf("snapshot status %q should not be importable", status)
		}
	}

	for _, status := range []string{"ACTIVE", "MODIFYING"} {
		if !redshiftServerlessEndpointAccessImportable(redshiftserverlesstypes.EndpointAccess{
			EndpointName:   redshiftServerlessString("analytics-endpoint"),
			EndpointStatus: redshiftServerlessString(status),
			SubnetIds:      []string{"subnet-1"},
			WorkgroupName:  redshiftServerlessString("reporting"),
		}) {
			t.Fatalf("endpoint access status %q should be importable", status)
		}
	}
	for _, status := range []string{"", "CREATING", "DELETING", "FAILED"} {
		if redshiftServerlessEndpointAccessImportable(redshiftserverlesstypes.EndpointAccess{
			EndpointName:   redshiftServerlessString("analytics-endpoint"),
			EndpointStatus: redshiftServerlessString(status),
			SubnetIds:      []string{"subnet-1"},
			WorkgroupName:  redshiftServerlessString("reporting"),
		}) {
			t.Fatalf("endpoint access status %q should not be importable", status)
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

func redshiftServerlessInt64(value int64) *int64 {
	return &value
}
