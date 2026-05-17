// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

func TestACLImportIDConstruction(t *testing.T) {
	acl := ACL{
		Principal:                 "User:producer",
		Host:                      "*",
		Operation:                 "Write",
		PermissionType:            "Allow",
		ResourceType:              "Topic",
		ResourceName:              "orders",
		ResourcePatternTypeFilter: "Literal",
	}
	want := "User:producer|*|Write|Allow|Topic|orders|Literal"
	if got := acl.ImportID(); got != want {
		t.Fatalf("ImportID() = %q, want %q", got, want)
	}
	parsed, err := parseKafkaACLImportID(want)
	if err != nil {
		t.Fatalf("parseKafkaACLImportID() error = %v", err)
	}
	if !reflect.DeepEqual(parsed, acl) {
		t.Fatalf("parsed ACL = %#v, want %#v", parsed, acl)
	}
}

func TestACLImportIDHandlesWildcardValues(t *testing.T) {
	acl := ACL{
		Principal:                 "User:*",
		Host:                      "*",
		Operation:                 "All",
		PermissionType:            "Allow",
		ResourceType:              "Cluster",
		ResourceName:              "*",
		ResourcePatternTypeFilter: "Literal",
	}
	resource := ACLGenerator{}.createResources([]ACL{acl})[0]
	if resource.InstanceState.ID != "User:*|*|All|Allow|Cluster|*|Literal" {
		t.Fatalf("resource ID = %q", resource.InstanceState.ID)
	}
	for key, want := range acl.attributes() {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}

func TestACLResourceNamesAvoidCollisions(t *testing.T) {
	base := ACL{
		Principal:                 "User:svc/prod",
		Host:                      "*",
		Operation:                 "Read",
		PermissionType:            "Allow",
		ResourceType:              "Topic",
		ResourceName:              "payments.events/v1",
		ResourcePatternTypeFilter: "Literal",
	}
	other := base
	other.Operation = "Write"

	first := kafkaACLResourceName(base)
	second := kafkaACLResourceName(other)
	if first == second {
		t.Fatalf("resource names collided: %q", first)
	}
	if kafkaACLResourceName(base) != first {
		t.Fatalf("resource name was not stable: %q", first)
	}
	for _, disallowed := range []string{".", "/", ":", "|", "*"} {
		if strings.Contains(first, disallowed) {
			t.Fatalf("resource name = %q contains %q", first, disallowed)
		}
	}
}

func TestACLInitResourcesImportsMultipleACLsForSamePrincipalResource(t *testing.T) {
	admin := &mockAdmin{
		acls: []sarama.ResourceAcls{{
			Resource: sarama.Resource{
				ResourceType:        sarama.AclResourceTopic,
				ResourceName:        "orders",
				ResourcePatternType: sarama.AclPatternLiteral,
			},
			Acls: []*sarama.Acl{
				{
					Principal:      "User:consumer",
					Host:           "*",
					Operation:      sarama.AclOperationRead,
					PermissionType: sarama.AclPermissionAllow,
				},
				{
					Principal:      "User:consumer",
					Host:           "*",
					Operation:      sarama.AclOperationDescribe,
					PermissionType: sarama.AclPermissionAllow,
				},
			},
		}},
	}
	generator := &ACLGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("resources len = %d, want 2", len(generator.Resources))
	}
	ids := []string{generator.Resources[0].InstanceState.ID, generator.Resources[1].InstanceState.ID}
	want := []string{
		"User:consumer|*|Describe|Allow|Topic|orders|Literal",
		"User:consumer|*|Read|Allow|Topic|orders|Literal",
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("resource IDs = %#v, want %#v", ids, want)
	}
}

func TestACLInitResourcesHandlesEmptyACLList(t *testing.T) {
	admin := &mockAdmin{}
	generator := &ACLGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("resources len = %d, want 0", len(generator.Resources))
	}
	if len(admin.listACLFilters) != 5 {
		t.Fatalf("ListAcls calls = %d, want one per Kafka ACL resource type", len(admin.listACLFilters))
	}
}

func TestACLInitResourcesAppliesIDFilter(t *testing.T) {
	admin := &mockAdmin{
		acls: []sarama.ResourceAcls{{
			Resource: sarama.Resource{
				ResourceType:        sarama.AclResourceTopic,
				ResourceName:        "orders",
				ResourcePatternType: sarama.AclPatternLiteral,
			},
			Acls: []*sarama.Acl{
				{
					Principal:      "User:producer",
					Host:           "*",
					Operation:      sarama.AclOperationWrite,
					PermissionType: sarama.AclPermissionAllow,
				},
				{
					Principal:      "User:auditor",
					Host:           "*",
					Operation:      sarama.AclOperationRead,
					PermissionType: sarama.AclPermissionAllow,
				},
			},
		}},
	}
	generator := &ACLGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.ParseFilters([]string{"acls=User:producer|*|Write|Allow|Topic|orders|Literal"})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(admin.listACLFilters) != 1 {
		t.Fatalf("ListAcls calls = %d, want 1", len(admin.listACLFilters))
	}
	filter := admin.listACLFilters[0]
	if filter.ResourceType != sarama.AclResourceTopic {
		t.Fatalf("filter resource type = %v, want Topic", filter.ResourceType)
	}
	if filter.ResourceName == nil || *filter.ResourceName != "orders" {
		t.Fatalf("filter resource name = %#v, want orders", filter.ResourceName)
	}
	if filter.Principal == nil || *filter.Principal != "User:producer" {
		t.Fatalf("filter principal = %#v, want User:producer", filter.Principal)
	}
	if filter.Operation != sarama.AclOperationWrite {
		t.Fatalf("filter operation = %v, want Write", filter.Operation)
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceState.ID; got != "User:producer|*|Write|Allow|Topic|orders|Literal" {
		t.Fatalf("resource ID = %q", got)
	}
}

func TestACLIDFilterSyntaxKeepsPrincipalColon(t *testing.T) {
	generator := &ACLGenerator{}
	generator.ParseFilters([]string{"Type=acl;Name=id;Value=User:producer|*|Write|Allow|Topic|orders|Literal"})
	if len(generator.Filter) != 1 {
		t.Fatalf("filter len = %d, want 1", len(generator.Filter))
	}
	filter := generator.Filter[0]
	if filter.ServiceName != "acl" {
		t.Fatalf("filter service = %q, want acl", filter.ServiceName)
	}
	if filter.FieldPath != "id" {
		t.Fatalf("filter field = %q, want id", filter.FieldPath)
	}
	want := []string{"User:producer|*|Write|Allow|Topic|orders|Literal"}
	if !reflect.DeepEqual(filter.AcceptableValues, want) {
		t.Fatalf("filter values = %#v, want %#v", filter.AcceptableValues, want)
	}

	acls, err := generator.explicitlyRequestedACLs()
	if err != nil {
		t.Fatalf("explicitlyRequestedACLs() error = %v", err)
	}
	if len(acls) != 1 {
		t.Fatalf("explicit ACLs len = %d, want 1", len(acls))
	}
	if acls[0].Principal != "User:producer" {
		t.Fatalf("principal = %q, want User:producer", acls[0].Principal)
	}
}

func TestACLPreservesRequiredFieldsAfterImportFallback(t *testing.T) {
	acl := ACL{
		Principal:                 "User:ANONYMOUS",
		Host:                      "*",
		Operation:                 "Read",
		PermissionType:            "Deny",
		ResourceType:              "Group",
		ResourceName:              "payments.events/v1",
		ResourcePatternTypeFilter: "Prefixed",
	}
	resource := ACLGenerator{}.createResources([]ACL{acl})[0]
	resource.InstanceState.Attributes = map[string]string{"id": acl.ImportID()}
	parser := terraformutils.NewFlatmapParser(
		resource.InstanceState.Attributes,
		[]*regexp.Regexp{regexp.MustCompile("^id$")},
		nil,
	)
	if err := resource.ParseTFstate(parser, cty.Object(map[string]cty.Type{
		"id":                           cty.String,
		"acl_principal":                cty.String,
		"acl_host":                     cty.String,
		"acl_operation":                cty.String,
		"acl_permission_type":          cty.String,
		"resource_type":                cty.String,
		"resource_name":                cty.String,
		"resource_pattern_type_filter": cty.String,
	})); err != nil {
		t.Fatalf("ParseTFstate() error = %v", err)
	}

	for key, want := range acl.attributes() {
		if got := resource.Item[key]; got != want {
			t.Fatalf("Item[%q] = %#v, want %#v", key, got, want)
		}
	}
	if _, ok := resource.Item["id"]; ok {
		t.Fatal("id attribute was not filtered from generated config item")
	}
}

func TestACLSkipsPipeDelimitedUnencodableFields(t *testing.T) {
	admin := &mockAdmin{
		acls: []sarama.ResourceAcls{{
			Resource: sarama.Resource{
				ResourceType:        sarama.AclResourceTopic,
				ResourceName:        "orders|archive",
				ResourcePatternType: sarama.AclPatternLiteral,
			},
			Acls: []*sarama.Acl{{
				Principal:      "User:producer",
				Host:           "*",
				Operation:      sarama.AclOperationWrite,
				PermissionType: sarama.AclPermissionAllow,
			}},
		}},
	}
	generator := &ACLGenerator{}
	generator.SetArgs(map[string]interface{}{
		"config": Config{BootstrapServers: []string{"broker1.example.com:9092"}},
	})
	generator.newAdmin = func(Config) (adminClient, error) { return admin, nil }

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("resources len = %d, want 0", len(generator.Resources))
	}
}

func TestACLDescribeResponseErrorsFailImport(t *testing.T) {
	message := "ACL authorization failed"
	_, err := resourceACLsFromDescribeResponse(&sarama.DescribeAclsResponse{
		Err:    sarama.ErrClusterAuthorizationFailed,
		ErrMsg: &message,
	})
	if err == nil {
		t.Fatal("expected describe ACLs error")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("error = %q, want authorization context", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("error = %q, want broker message %q", err, message)
	}

	_, err = resourceACLsFromDescribeResponse(&sarama.DescribeAclsResponse{
		Err: sarama.ErrSecurityDisabled,
	})
	if err == nil {
		t.Fatal("expected security-disabled describe ACLs error")
	}
	if !strings.Contains(err.Error(), "Security features are disabled") {
		t.Fatalf("error = %q, want security-disabled context", err)
	}
}

func TestACLDescribeResponseResources(t *testing.T) {
	resources, err := resourceACLsFromDescribeResponse(&sarama.DescribeAclsResponse{
		Err: sarama.ErrNoError,
		ResourceAcls: []*sarama.ResourceAcls{
			nil,
			{
				Resource: sarama.Resource{
					ResourceType:        sarama.AclResourceTopic,
					ResourceName:        "orders",
					ResourcePatternType: sarama.AclPatternLiteral,
				},
				Acls: []*sarama.Acl{{
					Principal:      "User:producer",
					Host:           "*",
					Operation:      sarama.AclOperationWrite,
					PermissionType: sarama.AclPermissionAllow,
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("resourceACLsFromDescribeResponse() error = %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources len = %d, want 1", len(resources))
	}
	if resources[0].ResourceName != "orders" {
		t.Fatalf("resource name = %q, want orders", resources[0].ResourceName)
	}
}
