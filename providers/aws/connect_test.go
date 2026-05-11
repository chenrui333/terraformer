// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	connecttypes "github.com/aws/aws-sdk-go-v2/service/connect/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestConnectImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "instance", got: connectInstanceImportID("instance-123"), want: "instance-123"},
		{name: "storage config", got: connectInstanceStorageConfigImportID("instance-123", "assoc-456", "CALL_RECORDINGS"), want: "instance-123:assoc-456:CALL_RECORDINGS"},
		{name: "lambda association", got: connectLambdaFunctionAssociationImportID("instance-123", "arn:aws:lambda:us-east-1:123456789012:function:handler"), want: "instance-123,arn:aws:lambda:us-east-1:123456789012:function:handler"},
		{name: "bot association", got: connectBotAssociationImportID("instance-123", "support", "us-east-1"), want: "instance-123:support:us-east-1"},
		{name: "two part", got: connectTwoPartImportID("instance-123", "queue-456"), want: "instance-123:queue-456"},
		{name: "phone number", got: connectPhoneNumberImportID("phone-123"), want: "phone-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestConnectResourceNameAvoidsSanitizedCollisions(t *testing.T) {
	tests := []struct {
		name   string
		first  []string
		second []string
	}{
		{name: "separator boundary", first: []string{"queue", "a_b", "c"}, second: []string{"queue", "a", "b_c"}},
		{name: "colon encoding", first: []string{"lambda", "a:b"}, second: []string{"lambda", "a-003A-b"}},
		{name: "slash encoding", first: []string{"bot", "a/b"}, second: []string{"bot", "a-002F-b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := terraformutils.TfSanitize(connectResourceName(tt.first...))
			second := terraformutils.TfSanitize(connectResourceName(tt.second...))
			if first == second {
				t.Fatalf("connectResourceName() generated duplicate sanitized name %q", first)
			}
		})
	}
}

func TestNewConnectInstanceResource(t *testing.T) {
	resource, ok := newConnectInstanceResource(connecttypes.InstanceSummary{
		Id:                     aws.String("instance-123"),
		InstanceAlias:          aws.String("support"),
		InstanceStatus:         connecttypes.InstanceStatusActive,
		IdentityManagementType: connecttypes.DirectoryTypeConnectManaged,
		InboundCallsEnabled:    aws.Bool(true),
		OutboundCallsEnabled:   aws.Bool(false),
	})
	assertConnectResource(t, resource, ok, connectInstanceResourceType, "instance-123", map[string]string{
		"identity_management_type": "CONNECT_MANAGED",
		"inbound_calls_enabled":    "true",
		"instance_alias":           "support",
		"outbound_calls_enabled":   "false",
	})
	if _, ok := newConnectInstanceResource(connecttypes.InstanceSummary{Id: aws.String("instance-123"), InstanceStatus: connecttypes.InstanceStatusCreationInProgress}); ok {
		t.Fatal("non-active instance should be skipped")
	}
	if _, ok := newConnectInstanceResource(connecttypes.InstanceSummary{Id: aws.String("instance-123"), InstanceStatus: connecttypes.InstanceStatusActive}); ok {
		t.Fatal("instance without reconstructable alias should be skipped")
	}
	if _, ok := newConnectInstanceResource(connecttypes.InstanceSummary{}); ok {
		t.Fatal("instance with empty ID should be skipped")
	}
}

func TestNewConnectInstanceReference(t *testing.T) {
	ref, ok := newConnectInstanceReference(connecttypes.InstanceSummary{
		Id:             aws.String("instance-123"),
		InstanceStatus: connecttypes.InstanceStatusActive,
	})
	if !ok {
		t.Fatal("active aliasless instance should be retained for child discovery")
	}
	if ref.id != "instance-123" {
		t.Fatalf("instance reference ID = %q, want %q", ref.id, "instance-123")
	}
	if _, ok := newConnectInstanceReference(connecttypes.InstanceSummary{Id: aws.String("instance-123"), InstanceStatus: connecttypes.InstanceStatusCreationInProgress}); ok {
		t.Fatal("non-active instance should be skipped for child discovery")
	}
	if _, ok := newConnectInstanceReference(connecttypes.InstanceSummary{}); ok {
		t.Fatal("instance reference with empty ID should be skipped")
	}
}

func TestNewConnectRelationshipResources(t *testing.T) {
	instanceID := "instance-123"
	tests := []struct {
		name       string
		resource   terraformutils.Resource
		ok         bool
		wantType   string
		wantID     string
		wantAttrs  map[string]string
		wantNameID string
	}{
		{
			name:     "storage config",
			resource: mustConnectResource(newConnectInstanceStorageConfigResource(instanceID, connecttypes.InstanceStorageResourceTypeCallRecordings, connecttypes.InstanceStorageConfig{AssociationId: aws.String("assoc-456")})),
			ok:       true,
			wantType: connectInstanceStorageConfigResourceType,
			wantID:   "instance-123:assoc-456:CALL_RECORDINGS",
			wantAttrs: map[string]string{
				"association_id": "assoc-456",
				"instance_id":    instanceID,
				"resource_type":  "CALL_RECORDINGS",
			},
		},
		{
			name:     "lambda function association",
			resource: mustConnectResource(newConnectLambdaFunctionAssociationResource(instanceID, "arn:aws:lambda:us-east-1:123456789012:function:handler")),
			ok:       true,
			wantType: connectLambdaFunctionAssociationResourceType,
			wantID:   "instance-123,arn:aws:lambda:us-east-1:123456789012:function:handler",
			wantAttrs: map[string]string{
				"function_arn": "arn:aws:lambda:us-east-1:123456789012:function:handler",
				"instance_id":  instanceID,
			},
		},
		{
			name:     "bot association",
			resource: mustConnectResource(newConnectBotAssociationResource(instanceID, connecttypes.LexBotConfig{LexBot: &connecttypes.LexBot{Name: aws.String("support"), LexRegion: aws.String("us-east-1")}})),
			ok:       true,
			wantType: connectBotAssociationResourceType,
			wantID:   "instance-123:support:us-east-1",
			wantAttrs: map[string]string{
				"instance_id": instanceID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertConnectResource(t, tt.resource, tt.ok, tt.wantType, tt.wantID, tt.wantAttrs)
		})
	}
	if _, ok := newConnectInstanceStorageConfigResource(instanceID, connecttypes.InstanceStorageResourceTypeCallRecordings, connecttypes.InstanceStorageConfig{}); ok {
		t.Fatal("storage config with empty association ID should be skipped")
	}
	if _, ok := newConnectLambdaFunctionAssociationResource(instanceID, ""); ok {
		t.Fatal("lambda association with empty function ARN should be skipped")
	}
	if _, ok := newConnectBotAssociationResource(instanceID, connecttypes.LexBotConfig{}); ok {
		t.Fatal("bot association with empty Lex bot should be skipped")
	}
}

func TestNewConnectChildResources(t *testing.T) {
	instanceID := "instance-123"
	tests := []struct {
		name      string
		resource  terraformutils.Resource
		ok        bool
		wantType  string
		wantID    string
		wantAttrs map[string]string
	}{
		{
			name:     "hours of operation",
			resource: mustConnectResource(newConnectHoursOfOperationResource(instanceID, connecttypes.HoursOfOperationSummary{Id: aws.String("hours-123"), Name: aws.String("business")})),
			ok:       true,
			wantType: connectHoursOfOperationResourceType,
			wantID:   "instance-123:hours-123",
			wantAttrs: map[string]string{
				"hours_of_operation_id": "hours-123",
				"instance_id":           instanceID,
				"name":                  "business",
			},
		},
		{
			name:     "queue",
			resource: mustConnectResource(newConnectQueueResource(instanceID, connecttypes.QueueSummary{Id: aws.String("queue-123"), Name: aws.String("support"), QueueType: connecttypes.QueueTypeStandard})),
			ok:       true,
			wantType: connectQueueResourceType,
			wantID:   "instance-123:queue-123",
			wantAttrs: map[string]string{
				"instance_id": instanceID,
				"name":        "support",
				"queue_id":    "queue-123",
			},
		},
		{
			name:     "quick connect",
			resource: mustConnectResource(newConnectQuickConnectResource(instanceID, connecttypes.QuickConnectSummary{Id: aws.String("quick-123"), Name: aws.String("operator")})),
			ok:       true,
			wantType: connectQuickConnectResourceType,
			wantID:   "instance-123:quick-123",
			wantAttrs: map[string]string{
				"instance_id":      instanceID,
				"name":             "operator",
				"quick_connect_id": "quick-123",
			},
		},
		{
			name:     "routing profile",
			resource: mustConnectResource(newConnectRoutingProfileResource(instanceID, connecttypes.RoutingProfileSummary{Id: aws.String("routing-123"), Name: aws.String("default")})),
			ok:       true,
			wantType: connectRoutingProfileResourceType,
			wantID:   "instance-123:routing-123",
			wantAttrs: map[string]string{
				"instance_id":        instanceID,
				"name":               "default",
				"routing_profile_id": "routing-123",
			},
		},
		{
			name:     "security profile",
			resource: mustConnectResource(newConnectSecurityProfileResource(instanceID, connecttypes.SecurityProfileSummary{Id: aws.String("security-123"), Name: aws.String("agent")})),
			ok:       true,
			wantType: connectSecurityProfileResourceType,
			wantID:   "instance-123:security-123",
			wantAttrs: map[string]string{
				"instance_id":         instanceID,
				"name":                "agent",
				"security_profile_id": "security-123",
			},
		},
		{
			name:     "user",
			resource: mustConnectResource(newConnectUserResource(instanceID, connecttypes.UserSummary{Id: aws.String("user-123"), Username: aws.String("alice")})),
			ok:       true,
			wantType: connectUserResourceType,
			wantID:   "instance-123:user-123",
			wantAttrs: map[string]string{
				"instance_id": instanceID,
				"name":        "alice",
				"user_id":     "user-123",
			},
		},
		{
			name:     "hierarchy group",
			resource: mustConnectResource(newConnectUserHierarchyGroupResource(instanceID, connecttypes.HierarchyGroup{Id: aws.String("group-123"), Name: aws.String("sales")})),
			ok:       true,
			wantType: connectUserHierarchyGroupResourceType,
			wantID:   "instance-123:group-123",
			wantAttrs: map[string]string{
				"hierarchy_group_id": "group-123",
				"instance_id":        instanceID,
				"name":               "sales",
			},
		},
		{
			name: "nested hierarchy group",
			resource: mustConnectResource(newConnectUserHierarchyGroupResource(instanceID, connecttypes.HierarchyGroup{
				Id:   aws.String("child-123"),
				Name: aws.String("east"),
				HierarchyPath: &connecttypes.HierarchyPath{
					LevelOne: &connecttypes.HierarchyGroupSummary{Id: aws.String("parent-123")},
					LevelTwo: &connecttypes.HierarchyGroupSummary{Id: aws.String("child-123")},
				},
			})),
			ok:       true,
			wantType: connectUserHierarchyGroupResourceType,
			wantID:   "instance-123:child-123",
			wantAttrs: map[string]string{
				"hierarchy_group_id": "child-123",
				"instance_id":        instanceID,
				"name":               "east",
				"parent_group_id":    "parent-123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertConnectResource(t, tt.resource, tt.ok, tt.wantType, tt.wantID, tt.wantAttrs)
		})
	}
	if _, ok := newConnectQueueResource(instanceID, connecttypes.QueueSummary{Id: aws.String("queue-123"), QueueType: connecttypes.QueueTypeAgent}); ok {
		t.Fatal("agent queue should be skipped")
	}
	if _, ok := newConnectUserResource(instanceID, connecttypes.UserSummary{}); ok {
		t.Fatal("child resource with empty ID should be skipped")
	}
}

func TestNewConnectUserHierarchyStructureResource(t *testing.T) {
	resource, ok := newConnectUserHierarchyStructureResource("instance-123", connecttypes.HierarchyStructure{
		LevelOne: &connecttypes.HierarchyLevel{Name: aws.String("Region")},
	})
	assertConnectResource(t, resource, ok, connectUserHierarchyStructureResourceType, "instance-123", map[string]string{
		"instance_id": "instance-123",
	})
	if _, ok := newConnectUserHierarchyStructureResource("instance-123", connecttypes.HierarchyStructure{}); ok {
		t.Fatal("empty hierarchy structure should be skipped")
	}
}

func TestNewConnectPhoneNumberResource(t *testing.T) {
	resource, ok := newConnectPhoneNumberResource(connecttypes.ClaimedPhoneNumberSummary{
		PhoneNumberCountryCode: connecttypes.PhoneNumberCountryCodeUs,
		PhoneNumberDescription: aws.String("support line"),
		PhoneNumberId:          aws.String("phone-123"),
		PhoneNumberStatus:      &connecttypes.PhoneNumberStatus{Status: connecttypes.PhoneNumberWorkflowStatusClaimed},
		PhoneNumberType:        connecttypes.PhoneNumberTypeTollFree,
		TargetArn:              aws.String("arn:aws:connect:us-east-1:123456789012:instance/instance-123"),
	})
	assertConnectResource(t, resource, ok, connectPhoneNumberResourceType, "phone-123", map[string]string{
		"country_code": "US",
		"description":  "support line",
		"target_arn":   "arn:aws:connect:us-east-1:123456789012:instance/instance-123",
		"type":         "TOLL_FREE",
	})
	if _, ok := newConnectPhoneNumberResource(connecttypes.ClaimedPhoneNumberSummary{
		PhoneNumberId:     aws.String("phone-123"),
		PhoneNumberStatus: &connecttypes.PhoneNumberStatus{Status: connecttypes.PhoneNumberWorkflowStatusInProgress},
	}); ok {
		t.Fatal("in-progress phone number should be skipped")
	}
	if _, ok := newConnectPhoneNumberResource(connecttypes.ClaimedPhoneNumberSummary{
		PhoneNumberCountryCode: connecttypes.PhoneNumberCountryCodeUs,
		PhoneNumberId:          aws.String("phone-123"),
		PhoneNumberStatus:      &connecttypes.PhoneNumberStatus{Status: connecttypes.PhoneNumberWorkflowStatusClaimed},
		PhoneNumberType:        connecttypes.PhoneNumberTypeTollFree,
	}); ok {
		t.Fatal("phone number with empty target ARN should be skipped")
	}
	if _, ok := newConnectPhoneNumberResource(connecttypes.ClaimedPhoneNumberSummary{}); ok {
		t.Fatal("phone number with empty ID should be skipped")
	}
}

func TestConnectTrafficDistributionGroupTargetARN(t *testing.T) {
	tests := []struct {
		name  string
		group connecttypes.TrafficDistributionGroupSummary
		want  string
	}{
		{
			name: "active group ARN",
			group: connecttypes.TrafficDistributionGroupSummary{
				Arn:    aws.String("arn:aws:connect:us-east-1:123456789012:traffic-distribution-group/tdg-123"),
				Status: connecttypes.TrafficDistributionGroupStatusActive,
			},
			want: "arn:aws:connect:us-east-1:123456789012:traffic-distribution-group/tdg-123",
		},
		{
			name: "replica region ID ARN",
			group: connecttypes.TrafficDistributionGroupSummary{
				Id:     aws.String("arn:aws:connect:us-east-1:123456789012:traffic-distribution-group/tdg-123"),
				Status: connecttypes.TrafficDistributionGroupStatusActive,
			},
			want: "arn:aws:connect:us-east-1:123456789012:traffic-distribution-group/tdg-123",
		},
		{
			name: "inactive group",
			group: connecttypes.TrafficDistributionGroupSummary{
				Arn:    aws.String("arn:aws:connect:us-east-1:123456789012:traffic-distribution-group/tdg-123"),
				Status: connecttypes.TrafficDistributionGroupStatusPendingDeletion,
			},
		},
		{
			name: "non ARN ID",
			group: connecttypes.TrafficDistributionGroupSummary{
				Id:     aws.String("tdg-123"),
				Status: connecttypes.TrafficDistributionGroupStatusActive,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectTrafficDistributionGroupTargetARN(tt.group); got != tt.want {
				t.Fatalf("target ARN = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectNotFound(t *testing.T) {
	if !connectNotFound(&connecttypes.ResourceNotFoundException{}) {
		t.Fatal("connectNotFound() = false, want true")
	}
	if connectNotFound(errors.New("other")) {
		t.Fatal("connectNotFound() = true for generic error, want false")
	}
}

func mustConnectResource(resource terraformutils.Resource, ok bool) terraformutils.Resource {
	if !ok {
		panic("resource constructor returned ok=false")
	}
	return resource
}

func assertConnectResource(t *testing.T, resource terraformutils.Resource, ok bool, wantType, wantID string, wantAttrs map[string]string) {
	t.Helper()
	if !ok {
		t.Fatal("resource constructor returned ok=false, want true")
	}
	if resource.InstanceInfo.Type != wantType {
		t.Fatalf("type = %q, want %q", resource.InstanceInfo.Type, wantType)
	}
	if resource.InstanceState.ID != wantID {
		t.Fatalf("ID = %q, want %q", resource.InstanceState.ID, wantID)
	}
	for key, want := range wantAttrs {
		if got := resource.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(resource.AllowEmptyValues) != 1 || resource.AllowEmptyValues[0] != "tags." {
		t.Fatalf("AllowEmptyValues = %#v, want tags", resource.AllowEmptyValues)
	}
}
