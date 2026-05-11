// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	opensearchtypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearchserverless/document"
	opensearchserverlesstypes "github.com/aws/aws-sdk-go-v2/service/opensearchserverless/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestOpenSearchResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(openSearchResourceName("domain", "ab", "c"))
	second := terraformutils.TfSanitize(openSearchResourceName("domain", "a", "bc"))
	if first == second {
		t.Fatalf("openSearchResourceName() collision after sanitize: %q", first)
	}
	if got := openSearchResourceName(); got != openSearchResourceNameFallback {
		t.Fatalf("openSearchResourceName() = %q, want fallback", got)
	}
}

func TestOpenSearchImportIDs(t *testing.T) {
	domain := opensearchtypes.DomainStatus{DomainName: openSearchTestString("search")}
	if got := openSearchDomainImportID(domain); got != "search" {
		t.Fatalf("domain import ID = %q, want search", got)
	}
	if got := openSearchDomainPolicyImportID(domain); got != "esd-policy-search" {
		t.Fatalf("domain policy import ID = %q, want esd-policy-search", got)
	}
	if got := openSearchDomainSAMLImportID(domain); got != "search" {
		t.Fatalf("domain SAML import ID = %q, want search", got)
	}
	association := opensearchtypes.DomainPackageDetails{
		DomainName: openSearchTestString("search"),
		PackageID:  openSearchTestString("pkg-123"),
	}
	if got := openSearchPackageAssociationImportID(association); got != "search-pkg-123" {
		t.Fatalf("package association import ID = %q, want search-pkg-123", got)
	}
	if got := openSearchVPCEndpointImportID(opensearchtypes.VpcEndpoint{VpcEndpointId: openSearchTestString("vpce-123")}); got != "vpce-123" {
		t.Fatalf("VPC endpoint import ID = %q, want vpce-123", got)
	}
	if got := openSearchOutboundConnectionImportID(opensearchtypes.OutboundConnection{ConnectionId: openSearchTestString("conn-123")}); got != "conn-123" {
		t.Fatalf("outbound import ID = %q, want conn-123", got)
	}
	if got := openSearchInboundConnectionAccepterImportID(opensearchtypes.InboundConnection{ConnectionId: openSearchTestString("conn-123")}); got != "conn-123" {
		t.Fatalf("inbound accepter import ID = %q, want conn-123", got)
	}
}

func TestNewOpenSearchDomainResource(t *testing.T) {
	created := true
	domain := opensearchtypes.DomainStatus{
		Created:                &created,
		DomainName:             openSearchTestString("search"),
		DomainProcessingStatus: opensearchtypes.DomainProcessingStatusTypeModifying,
	}
	resource, ok := newOpenSearchDomainResource(domain)
	assertOpenSearchResource(t, resource, ok, "search", openSearchResourceName("domain", "search"), openSearchDomainResourceType)
	assertOpenSearchAttribute(t, resource, "domain_name", "search")

	creating := domain
	creating.DomainProcessingStatus = opensearchtypes.DomainProcessingStatusTypeCreating
	if _, ok := newOpenSearchDomainResource(creating); ok {
		t.Fatal("creating domain should be skipped")
	}
	created = false
	notCreated := domain
	notCreated.Created = &created
	if _, ok := newOpenSearchDomainResource(notCreated); ok {
		t.Fatal("not-created domain should be skipped")
	}
}

func TestNewOpenSearchDomainPolicyResource(t *testing.T) {
	resource, ok := newOpenSearchDomainPolicyResource(opensearchtypes.DomainStatus{
		AccessPolicies: openSearchTestString("{\"Version\":\"2012-10-17\"}"),
		DomainName:     openSearchTestString("search"),
	})
	assertOpenSearchResource(t, resource, ok, "esd-policy-search", openSearchResourceName("domain-policy", "search"), openSearchDomainPolicyResourceType)
	assertOpenSearchAttribute(t, resource, "domain_name", "search")
	assertOpenSearchAttribute(t, resource, "access_policies", "{\"Version\":\"2012-10-17\"}")

	if _, ok := newOpenSearchDomainPolicyResource(opensearchtypes.DomainStatus{DomainName: openSearchTestString("search")}); ok {
		t.Fatal("domain policy without access policies should be skipped")
	}
}

func TestCleanOpenSearchDomainItemRemovesPolicyOwnership(t *testing.T) {
	resource, ok := newOpenSearchDomainResource(opensearchtypes.DomainStatus{
		AccessPolicies: openSearchTestString("{\"Version\":\"2012-10-17\"}"),
		DomainName:     openSearchTestString("search"),
	})
	assertOpenSearchResource(t, resource, ok, "search", openSearchResourceName("domain", "search"), openSearchDomainResourceType)
	resource.Item = map[string]interface{}{
		"access_policies": "{\"Version\":\"2012-10-17\"}",
		"cognito_options": []interface{}{map[string]interface{}{"enabled": false}},
		"cluster_config":  []interface{}{map[string]interface{}{"warm_count": 0}},
		"engine_version":  "OpenSearch_2.11",
		"domain_name":     "search",
	}
	resource.InstanceState.Attributes["access_policies"] = "{\"Version\":\"2012-10-17\"}"
	resource.InstanceState.Attributes["cognito_options.0.enabled"] = "false"
	resource.InstanceState.Attributes["cluster_config.0.warm_count"] = "0"

	cleanOpenSearchDomainItem(&resource)

	if _, exists := resource.Item["access_policies"]; exists {
		t.Fatal("domain access_policies should be removed when aws_opensearch_domain_policy owns it")
	}
	if _, exists := resource.InstanceState.Attributes["access_policies"]; exists {
		t.Fatal("domain access_policies state should be removed when aws_opensearch_domain_policy owns it")
	}
	if _, exists := resource.Item["cognito_options"]; exists {
		t.Fatal("disabled cognito_options should still be removed")
	}
}

func TestNewOpenSearchDomainSAMLOptionsResource(t *testing.T) {
	enabled := true
	resource, ok := newOpenSearchDomainSAMLOptionsResource(opensearchtypes.DomainStatus{
		DomainName: openSearchTestString("search"),
		AdvancedSecurityOptions: &opensearchtypes.AdvancedSecurityOptions{
			SAMLOptions: &opensearchtypes.SAMLOptionsOutput{
				Enabled: &enabled,
				Idp: &opensearchtypes.SAMLIdp{
					EntityId:        openSearchTestString("entity"),
					MetadataContent: openSearchTestString("<xml/>"),
				},
				RolesKey:              openSearchTestString("roles"),
				SessionTimeoutMinutes: openSearchTestInt32(30),
				SubjectKey:            openSearchTestString("subject"),
			},
		},
	})
	assertOpenSearchResource(t, resource, ok, "search", openSearchResourceName("domain-saml-options", "search"), openSearchDomainSAMLOptionsResourceType)
	assertOpenSearchAttribute(t, resource, "saml_options.0.idp.0.entity_id", "entity")
	assertOpenSearchAttribute(t, resource, "saml_options.0.session_timeout_minutes", "30")

	enabled = false
	if _, ok := newOpenSearchDomainSAMLOptionsResource(opensearchtypes.DomainStatus{
		DomainName: openSearchTestString("search"),
		AdvancedSecurityOptions: &opensearchtypes.AdvancedSecurityOptions{
			SAMLOptions: &opensearchtypes.SAMLOptionsOutput{Enabled: &enabled},
		},
	}); ok {
		t.Fatal("disabled SAML options should be skipped")
	}
}

func TestNewOpenSearchVPCEndpointResource(t *testing.T) {
	resource, ok := newOpenSearchVPCEndpointResource(opensearchtypes.VpcEndpoint{
		DomainArn:     openSearchTestString("arn:aws:es:us-east-1:123456789012:domain/search"),
		Status:        opensearchtypes.VpcEndpointStatusUpdating,
		VpcEndpointId: openSearchTestString("vpce-123"),
		VpcOptions: &opensearchtypes.VPCDerivedInfo{
			SecurityGroupIds: []string{"sg-1", "sg-2"},
			SubnetIds:        []string{"subnet-1", "subnet-2"},
		},
	})
	assertOpenSearchResource(t, resource, ok, "vpce-123", openSearchResourceName("vpc-endpoint", "vpce-123"), openSearchVPCEndpointResourceType)
	assertOpenSearchAttribute(t, resource, "domain_arn", "arn:aws:es:us-east-1:123456789012:domain/search")
	assertOpenSearchAttribute(t, resource, "vpc_options.0.subnet_ids.#", "2")
	assertOpenSearchAttribute(t, resource, "vpc_options.0.security_group_ids.1", "sg-2")

	if _, ok := newOpenSearchVPCEndpointResource(opensearchtypes.VpcEndpoint{
		DomainArn:     openSearchTestString("arn:aws:es:us-east-1:123456789012:domain/search"),
		Status:        opensearchtypes.VpcEndpointStatusDeleting,
		VpcEndpointId: openSearchTestString("vpce-123"),
		VpcOptions:    &opensearchtypes.VPCDerivedInfo{SubnetIds: []string{"subnet-1"}},
	}); ok {
		t.Fatal("deleting VPC endpoint should be skipped")
	}
}

func TestNewOpenSearchPackageAssociationResource(t *testing.T) {
	resource, ok := newOpenSearchPackageAssociationResource(opensearchtypes.DomainPackageDetails{
		DomainName:          openSearchTestString("search"),
		DomainPackageStatus: opensearchtypes.DomainPackageStatusActive,
		PackageID:           openSearchTestString("pkg-123"),
	})
	assertOpenSearchResource(t, resource, ok, "search-pkg-123", openSearchResourceName("package-association", "search", "pkg-123"), openSearchPackageAssociationResourceType)
	assertOpenSearchAttribute(t, resource, "domain_name", "search")
	assertOpenSearchAttribute(t, resource, "package_id", "pkg-123")

	if _, ok := newOpenSearchPackageAssociationResource(opensearchtypes.DomainPackageDetails{
		DomainName:          openSearchTestString("search"),
		DomainPackageStatus: opensearchtypes.DomainPackageStatusAssociating,
		PackageID:           openSearchTestString("pkg-123"),
	}); ok {
		t.Fatal("associating package association should be skipped")
	}
}

func TestNewOpenSearchConnections(t *testing.T) {
	outbound, ok := newOpenSearchOutboundConnectionResource(opensearchtypes.OutboundConnection{
		ConnectionAlias: openSearchTestString("analytics"),
		ConnectionId:    openSearchTestString("conn-123"),
		ConnectionMode:  opensearchtypes.ConnectionModeDirect,
		ConnectionProperties: &opensearchtypes.ConnectionProperties{
			CrossClusterSearch: &opensearchtypes.CrossClusterSearchConnectionProperties{
				SkipUnavailable: opensearchtypes.SkipUnavailableStatusEnabled,
			},
		},
		ConnectionStatus: &opensearchtypes.OutboundConnectionStatus{StatusCode: opensearchtypes.OutboundConnectionStatusCodePendingAcceptance},
		LocalDomainInfo:  openSearchTestDomainInfo("local", "111111111111", "us-east-1"),
		RemoteDomainInfo: openSearchTestDomainInfo("remote", "222222222222", "us-west-2"),
	})
	assertOpenSearchResource(t, outbound, ok, "conn-123", openSearchResourceName("outbound-connection", "analytics", "conn-123"), openSearchOutboundConnectionResourceType)
	assertOpenSearchAttribute(t, outbound, "local_domain_info.0.domain_name", "local")
	assertOpenSearchAttribute(t, outbound, "connection_properties.0.cross_cluster_search.0.skip_unavailable", "ENABLED")

	if _, ok := newOpenSearchOutboundConnectionResource(opensearchtypes.OutboundConnection{
		ConnectionAlias:  openSearchTestString("analytics"),
		ConnectionId:     openSearchTestString("conn-123"),
		ConnectionStatus: &opensearchtypes.OutboundConnectionStatus{StatusCode: opensearchtypes.OutboundConnectionStatusCodeValidationFailed},
		LocalDomainInfo:  openSearchTestDomainInfo("local", "111111111111", "us-east-1"),
		RemoteDomainInfo: openSearchTestDomainInfo("remote", "222222222222", "us-west-2"),
	}); ok {
		t.Fatal("validation-failed outbound connection should be skipped")
	}

	inbound, ok := newOpenSearchInboundConnectionAccepterResource(opensearchtypes.InboundConnection{
		ConnectionId:     openSearchTestString("conn-123"),
		ConnectionStatus: &opensearchtypes.InboundConnectionStatus{StatusCode: opensearchtypes.InboundConnectionStatusCodeActive},
	})
	assertOpenSearchResource(t, inbound, ok, "conn-123", openSearchResourceName("inbound-connection-accepter", "conn-123"), openSearchInboundConnectionAccepterResourceType)
	assertOpenSearchAttribute(t, inbound, "connection_id", "conn-123")
}

func TestOpenSearchOptionalResourceErrorSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "resource not found", err: &opensearchtypes.ResourceNotFoundException{}, want: true},
		{name: "generic access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "unexpected generic error", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openSearchOptionalResourceErrorSkippable(tt.err); got != tt.want {
				t.Fatalf("openSearchOptionalResourceErrorSkippable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestOpenSearchOptionalResourceLoaderErrors(t *testing.T) {
	g := OpenSearchGenerator{}
	calledNext := false
	err := g.loadOptionalResources([]openSearchOptionalResourceLoader{
		{name: "VPC endpoints", load: func() error { return &opensearchtypes.ResourceNotFoundException{} }},
		{name: "outbound connections", load: func() error {
			calledNext = true
			return nil
		}},
	})
	if err != nil {
		t.Fatalf("loadOptionalResources() error = %v, want nil", err)
	}
	if !calledNext {
		t.Fatal("loadOptionalResources() should continue after skippable error")
	}
	boom := errors.New("boom")
	calledNext = false
	err = g.loadOptionalResources([]openSearchOptionalResourceLoader{
		{name: "VPC endpoints", load: func() error { return boom }},
		{name: "outbound connections", load: func() error {
			calledNext = true
			return nil
		}},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
	}
	if calledNext {
		t.Fatal("loadOptionalResources() should stop after unexpected error")
	}
}

func TestOpenSearchDomainPolicyHeredoc(t *testing.T) {
	g := &OpenSearchGenerator{}
	resource := terraformutils.Resource{Item: map[string]interface{}{"access_policies": "{\"Version\":\"2012-10-17\"}"}}
	wrapOpenSearchDomainPolicyHeredoc(g, &resource)
	if got := resource.Item["access_policies"]; got != "<<POLICY\n{\"Version\":\"2012-10-17\"}\nPOLICY" {
		t.Fatalf("access_policies heredoc = %q", got)
	}
}

func TestOpenSearchUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	resources, ok := unsupported["resources"]
	if !ok {
		t.Fatal("unsupported resources JSON missing resources field")
	}
	entries, ok := resources.([]interface{})
	if !ok {
		t.Fatalf("unsupported resources JSON resources field has type %T, want []interface{}", resources)
	}
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resources JSON entry has type %T, want map[string]interface{}", rawEntry)
		}
		if entry["resource"] != "aws_opensearch_package" {
			continue
		}
		if entry["service_family"] != "opensearch" || entry["status"] != "unsupported" || entry["reason"] == "" || entry["evidence"] == "" {
			t.Fatalf("invalid aws_opensearch_package unsupported entry: %+v", entry)
		}
		return
	}
	t.Fatal("aws_opensearch_package unsupported entry was not found")
}

func TestOpenSearchServerlessResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(openSearchServerlessResourceName("collection", "ab", "c"))
	second := terraformutils.TfSanitize(openSearchServerlessResourceName("collection", "a", "bc"))
	if first == second {
		t.Fatalf("openSearchServerlessResourceName() collision after sanitize: %q", first)
	}
	if got := openSearchServerlessResourceName(); got != openSearchServerlessResourceNameFallback {
		t.Fatalf("openSearchServerlessResourceName() = %q, want fallback", got)
	}
}

func TestOpenSearchServerlessImportIDs(t *testing.T) {
	collection := opensearchserverlesstypes.CollectionDetail{Id: openSearchTestString("col-123")}
	if got := openSearchServerlessCollectionImportID(collection); got != "col-123" {
		t.Fatalf("collection import ID = %q, want col-123", got)
	}
	accessPolicy := opensearchserverlesstypes.AccessPolicyDetail{Name: openSearchTestString("data"), Type: opensearchserverlesstypes.AccessPolicyTypeData}
	if got := openSearchServerlessAccessPolicyImportID(accessPolicy); got != "data/data" {
		t.Fatalf("access policy import ID = %q, want data/data", got)
	}
	securityPolicy := opensearchserverlesstypes.SecurityPolicyDetail{Name: openSearchTestString("network"), Type: opensearchserverlesstypes.SecurityPolicyTypeNetwork}
	if got := openSearchServerlessSecurityPolicyImportID(securityPolicy); got != "network/network" {
		t.Fatalf("security policy import ID = %q, want network/network", got)
	}
	securityConfig := opensearchserverlesstypes.SecurityConfigDetail{Id: openSearchTestString("saml/123456789012/sso")}
	if got := openSearchServerlessSecurityConfigImportID(securityConfig); got != "saml/123456789012/sso" {
		t.Fatalf("security config import ID = %q, want full ID", got)
	}
	lifecyclePolicy := opensearchserverlesstypes.LifecyclePolicyDetail{Name: openSearchTestString("retention"), Type: opensearchserverlesstypes.LifecyclePolicyTypeRetention}
	if got := openSearchServerlessLifecyclePolicyImportID(lifecyclePolicy); got != "retention/retention" {
		t.Fatalf("lifecycle policy import ID = %q, want retention/retention", got)
	}
}

func TestNewOpenSearchServerlessCollectionResource(t *testing.T) {
	resource, ok := newOpenSearchServerlessCollectionResource(opensearchserverlesstypes.CollectionDetail{
		Description:     openSearchTestString("analytics"),
		Id:              openSearchTestString("col-123"),
		Name:            openSearchTestString("search"),
		StandbyReplicas: opensearchserverlesstypes.StandbyReplicasEnabled,
		Status:          opensearchserverlesstypes.CollectionStatusUpdating,
		Type:            opensearchserverlesstypes.CollectionTypeSearch,
	})
	assertOpenSearchResource(t, resource, ok, "col-123", openSearchServerlessResourceName("collection", "search", "col-123"), openSearchServerlessCollectionResourceType)
	assertOpenSearchAttribute(t, resource, "name", "search")
	assertOpenSearchAttribute(t, resource, "type", "SEARCH")

	if _, ok := newOpenSearchServerlessCollectionResource(opensearchserverlesstypes.CollectionDetail{
		Id:     openSearchTestString("col-123"),
		Name:   openSearchTestString("search"),
		Status: opensearchserverlesstypes.CollectionStatusCreating,
	}); ok {
		t.Fatal("creating collection should be skipped")
	}
}

func TestNewOpenSearchServerlessPolicies(t *testing.T) {
	accessResource, ok := newOpenSearchServerlessAccessPolicyResource(opensearchserverlesstypes.AccessPolicyDetail{
		Description: openSearchTestString("data access"),
		Name:        openSearchTestString("data"),
		Policy:      document.NewLazyDocument(map[string]interface{}{"Rules": []interface{}{}}),
		Type:        opensearchserverlesstypes.AccessPolicyTypeData,
	})
	assertOpenSearchResource(t, accessResource, ok, "data/data", openSearchServerlessResourceName("access-policy", "data", "data"), openSearchServerlessAccessPolicyResourceType)
	assertOpenSearchAttribute(t, accessResource, "policy", "{\"Rules\":[]}")

	securityResource, ok := newOpenSearchServerlessSecurityPolicyResource(opensearchserverlesstypes.SecurityPolicyDetail{
		Name:   openSearchTestString("network"),
		Policy: document.NewLazyDocument(map[string]interface{}{"Rules": []interface{}{}}),
		Type:   opensearchserverlesstypes.SecurityPolicyTypeNetwork,
	})
	assertOpenSearchResource(t, securityResource, ok, "network/network", openSearchServerlessResourceName("security-policy", "network", "network"), openSearchServerlessSecurityPolicyResourceType)

	lifecycleResource, ok := newOpenSearchServerlessLifecyclePolicyResource(opensearchserverlesstypes.LifecyclePolicyDetail{
		Name:   openSearchTestString("retention"),
		Policy: document.NewLazyDocument(map[string]interface{}{"Rules": []interface{}{}}),
		Type:   opensearchserverlesstypes.LifecyclePolicyTypeRetention,
	})
	assertOpenSearchResource(t, lifecycleResource, ok, "retention/retention", openSearchServerlessResourceName("lifecycle-policy", "retention", "retention"), openSearchServerlessLifecyclePolicyResourceType)
}

func TestNewOpenSearchServerlessSecurityConfigResource(t *testing.T) {
	resource, ok := newOpenSearchServerlessSecurityConfigResource(opensearchserverlesstypes.SecurityConfigDetail{
		Description: openSearchTestString("sso"),
		Id:          openSearchTestString("saml/123456789012/sso"),
		SamlOptions: &opensearchserverlesstypes.SamlConfigOptions{
			GroupAttribute: openSearchTestString("group"),
			Metadata:       openSearchTestString("<xml/>"),
			SessionTimeout: openSearchTestInt32(120),
			UserAttribute:  openSearchTestString("user"),
		},
		Type: opensearchserverlesstypes.SecurityConfigTypeSaml,
	})
	assertOpenSearchResource(t, resource, ok, "saml/123456789012/sso", openSearchServerlessResourceName("security-config", "saml", "sso"), openSearchServerlessSecurityConfigResourceType)
	assertOpenSearchAttribute(t, resource, "name", "sso")
	assertOpenSearchAttribute(t, resource, "saml_options.0.metadata", "<xml/>")

	if _, ok := newOpenSearchServerlessSecurityConfigResource(opensearchserverlesstypes.SecurityConfigDetail{
		Id:          openSearchTestString("saml/123456789012/sso"),
		SamlOptions: &opensearchserverlesstypes.SamlConfigOptions{},
		Type:        opensearchserverlesstypes.SecurityConfigTypeSaml,
	}); ok {
		t.Fatal("SAML security config without metadata should be skipped")
	}
	if got := openSearchServerlessSamlOptionsAttributes("saml_options", nil)["saml_options.#"]; got != "0" {
		t.Fatalf("nil SAML options count = %q, want 0", got)
	}
}

func TestNewOpenSearchServerlessVPCEndpointResource(t *testing.T) {
	resource, ok := newOpenSearchServerlessVPCEndpointResource(opensearchserverlesstypes.VpcEndpointDetail{
		Id:               openSearchTestString("vpce-123"),
		Name:             openSearchTestString("private-search"),
		SecurityGroupIds: []string{"sg-from-serverless"},
		Status:           opensearchserverlesstypes.VpcEndpointStatusActive,
		SubnetIds:        []string{"subnet-1", "subnet-2"},
		VpcId:            openSearchTestString("vpc-123"),
	}, []string{"sg-from-ec2"})
	assertOpenSearchResource(t, resource, ok, "vpce-123", openSearchServerlessResourceName("vpc-endpoint", "private-search", "vpce-123"), openSearchServerlessVPCEndpointResourceType)
	assertOpenSearchAttribute(t, resource, "name", "private-search")
	assertOpenSearchAttribute(t, resource, "subnet_ids.#", "2")
	assertOpenSearchAttribute(t, resource, "security_group_ids.0", "sg-from-ec2")

	if _, ok := newOpenSearchServerlessVPCEndpointResource(opensearchserverlesstypes.VpcEndpointDetail{
		Id:        openSearchTestString("vpce-456"),
		Name:      openSearchTestString("private-search"),
		Status:    opensearchserverlesstypes.VpcEndpointStatusActive,
		SubnetIds: []string{"subnet-1"},
		VpcId:     openSearchTestString("vpc-123"),
	}, nil); ok {
		t.Fatal("VPC endpoint without EC2 security group IDs should be skipped")
	}

	if _, ok := newOpenSearchServerlessVPCEndpointResource(opensearchserverlesstypes.VpcEndpointDetail{
		Id:        openSearchTestString("vpce-123"),
		Name:      openSearchTestString("private-search"),
		Status:    opensearchserverlesstypes.VpcEndpointStatusPending,
		SubnetIds: []string{"subnet-1"},
		VpcId:     openSearchTestString("vpc-123"),
	}, []string{"sg-1"}); ok {
		t.Fatal("pending VPC endpoint should be skipped")
	}
}

func TestOpenSearchServerlessCollectionIDs(t *testing.T) {
	got := openSearchServerlessCollectionIDs([]opensearchserverlesstypes.CollectionSummary{
		{Id: openSearchTestString("col-1")},
		{},
		{Id: openSearchTestString("col-2")},
	})
	want := []string{"col-1", "col-2"}
	if len(got) != len(want) {
		t.Fatalf("collection IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("collection IDs = %v, want %v", got, want)
		}
	}
	if chunks := openSearchServerlessStringChunks(got, 1); len(chunks) != 2 || len(chunks[0]) != 1 || chunks[0][0] != "col-1" {
		t.Fatalf("collection chunks = %v, want one ID per chunk", chunks)
	}
}

func TestOpenSearchServerlessEC2SecurityGroupIDs(t *testing.T) {
	got := openSearchServerlessEC2SecurityGroupIDs([]ec2types.SecurityGroupIdentifier{
		{GroupId: openSearchTestString("sg-1")},
		{},
		{GroupId: openSearchTestString("sg-2")},
	})
	want := []string{"sg-1", "sg-2"}
	if len(got) != len(want) {
		t.Fatalf("EC2 security group IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("EC2 security group IDs = %v, want %v", got, want)
		}
	}
}

func TestOpenSearchServerlessVPCEndpointIDsFiltersInactiveSummaries(t *testing.T) {
	got := openSearchServerlessVPCEndpointIDs([]opensearchserverlesstypes.VpcEndpointSummary{
		{Id: openSearchTestString("vpce-active"), Status: opensearchserverlesstypes.VpcEndpointStatusActive},
		{Id: openSearchTestString("vpce-pending"), Status: opensearchserverlesstypes.VpcEndpointStatusPending},
		{Id: openSearchTestString("vpce-deleting"), Status: opensearchserverlesstypes.VpcEndpointStatusDeleting},
		{Status: opensearchserverlesstypes.VpcEndpointStatusActive},
	})
	want := []string{"vpce-active"}
	if len(got) != len(want) {
		t.Fatalf("VPC endpoint IDs = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("VPC endpoint IDs = %v, want %v", got, want)
		}
	}
}

func TestOpenSearchServerlessEC2VPCEndpointSecurityGroupsRetriesStaleIDs(t *testing.T) {
	notFound := &smithy.GenericAPIError{Code: "InvalidVpcEndpointId.NotFound"}
	describer := &fakeOpenSearchServerlessEC2VPCEndpointDescriber{
		batchErr: notFound,
		errors: map[string]error{
			"vpce-stale": notFound,
		},
		outputs: map[string]*ec2.DescribeVpcEndpointsOutput{
			"vpce-active": {
				VpcEndpoints: []ec2types.VpcEndpoint{{
					VpcEndpointId: openSearchTestString("vpce-active"),
					Groups: []ec2types.SecurityGroupIdentifier{
						{GroupId: openSearchTestString("sg-1")},
					},
				}},
			},
		},
	}
	got, err := openSearchServerlessEC2VPCEndpointSecurityGroups(context.Background(), describer, []string{"vpce-active", "vpce-stale"})
	if err != nil {
		t.Fatalf("EC2 VPC endpoint security group lookup returned error: %v", err)
	}
	if got["vpce-active"][0] != "sg-1" {
		t.Fatalf("active endpoint security groups = %v, want sg-1", got["vpce-active"])
	}
	if _, exists := got["vpce-stale"]; exists {
		t.Fatal("stale endpoint should not produce security group state")
	}
	if len(describer.calls) != 3 {
		t.Fatalf("EC2 calls = %v, want batch call plus two single-ID retries", describer.calls)
	}
}

func TestOpenSearchServerlessOptionalResourceErrorSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "resource not found", err: &opensearchserverlesstypes.ResourceNotFoundException{}, want: true},
		{name: "generic access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "unexpected generic error", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openSearchServerlessOptionalResourceErrorSkippable(tt.err); got != tt.want {
				t.Fatalf("openSearchServerlessOptionalResourceErrorSkippable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestOpenSearchServerlessEC2ErrorSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "unauthorized operation", err: &smithy.GenericAPIError{Code: "UnauthorizedOperation"}, want: true},
		{name: "not found", err: &smithy.GenericAPIError{Code: "InvalidVpcEndpointId.NotFound"}, want: false},
		{name: "throttling", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openSearchServerlessEC2ErrorSkippable(tt.err); got != tt.want {
				t.Fatalf("openSearchServerlessEC2ErrorSkippable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenSearchServerlessEC2NotFound(t *testing.T) {
	if !openSearchServerlessEC2NotFound(&smithy.GenericAPIError{Code: "InvalidVpcEndpointId.NotFound"}) {
		t.Fatal("InvalidVpcEndpointId.NotFound should be detected as EC2 not found")
	}
	if openSearchServerlessEC2NotFound(&smithy.GenericAPIError{Code: "AccessDeniedException"}) {
		t.Fatal("AccessDeniedException should not be detected as EC2 not found")
	}
}

func TestOpenSearchServerlessOptionalResourceLoaderErrors(t *testing.T) {
	g := OpenSearchServerlessGenerator{}
	calledNext := false
	err := g.loadOptionalResources([]openSearchServerlessOptionalResourceLoader{
		{name: "collections", load: func() error { return &opensearchserverlesstypes.ResourceNotFoundException{} }},
		{name: "access policies", load: func() error {
			calledNext = true
			return nil
		}},
	})
	if err != nil {
		t.Fatalf("loadOptionalResources() error = %v, want nil", err)
	}
	if !calledNext {
		t.Fatal("loadOptionalResources() should continue after skippable error")
	}
	boom := errors.New("boom")
	calledNext = false
	err = g.loadOptionalResources([]openSearchServerlessOptionalResourceLoader{
		{name: "collections", load: func() error { return boom }},
		{name: "access policies", load: func() error {
			calledNext = true
			return nil
		}},
	})
	if !errors.Is(err, boom) {
		t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
	}
	if calledNext {
		t.Fatal("loadOptionalResources() should stop after unexpected error")
	}
}

func TestOpenSearchServerlessPolicyHeredoc(t *testing.T) {
	g := &OpenSearchServerlessGenerator{}
	resource := terraformutils.Resource{Item: map[string]interface{}{"policy": "{\"Rules\":[]}"}}
	wrapOpenSearchServerlessPolicyHeredoc(g, &resource)
	if got := resource.Item["policy"]; got != "<<POLICY\n{\"Rules\":[]}\nPOLICY" {
		t.Fatalf("policy heredoc = %q", got)
	}
}

func assertOpenSearchResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatalf("resource ok = false, want true")
	}
	if resource.InstanceState == nil {
		t.Fatal("resource InstanceState is nil")
	}
	if resource.InstanceState.ID != wantID {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	if resource.ResourceName != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, terraformutils.TfSanitize(wantName))
	}
	if resource.InstanceInfo == nil {
		t.Fatal("resource InstanceInfo is nil")
	}
	if resource.InstanceInfo.Type != wantType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, wantType)
	}
}

func assertOpenSearchAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if resource.InstanceState == nil {
		t.Fatal("resource InstanceState is nil")
	}
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("attribute %s = %q, want %q", key, got, want)
	}
}

func openSearchTestString(value string) *string {
	return &value
}

func openSearchTestInt32(value int32) *int32 {
	return &value
}

type fakeOpenSearchServerlessEC2VPCEndpointDescriber struct {
	batchErr error
	errors   map[string]error
	outputs  map[string]*ec2.DescribeVpcEndpointsOutput
	calls    [][]string
}

func (f *fakeOpenSearchServerlessEC2VPCEndpointDescriber) DescribeVpcEndpoints(_ context.Context, input *ec2.DescribeVpcEndpointsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcEndpointsOutput, error) {
	ids := append([]string{}, input.VpcEndpointIds...)
	f.calls = append(f.calls, ids)
	if len(ids) > 1 && f.batchErr != nil {
		return nil, f.batchErr
	}
	if len(ids) == 1 {
		id := ids[0]
		if err := f.errors[id]; err != nil {
			return nil, err
		}
		if output := f.outputs[id]; output != nil {
			return output, nil
		}
	}
	return &ec2.DescribeVpcEndpointsOutput{}, nil
}

func openSearchTestDomainInfo(domainName, ownerID, region string) *opensearchtypes.DomainInformationContainer {
	return &opensearchtypes.DomainInformationContainer{
		AWSDomainInformation: &opensearchtypes.AWSDomainInformation{
			DomainName: openSearchTestString(domainName),
			OwnerId:    openSearchTestString(ownerID),
			Region:     openSearchTestString(region),
		},
	}
}
