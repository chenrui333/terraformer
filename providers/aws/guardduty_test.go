// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	guarddutytypes "github.com/aws/aws-sdk-go-v2/service/guardduty/types"
)

const (
	testGuardDutyDetectorID       = "12abc34d567e8fa901bc2d34e56789f0"
	testGuardDutyFilterName       = "CriticalFindings"
	testGuardDutyIPSetID          = "ipset-1234567890abcdef"
	testGuardDutyMemberAccountID  = "123456789012"
	testGuardDutyOrganizationID   = "210987654321"
	testGuardDutyPlanID           = "mpl-1234567890abcdef"
	testGuardDutyPublishingID     = "publishing-1234567890abcdef"
	testGuardDutyThreatIntelSetID = "threatintelset-1234567890abcdef"
)

func TestGuardDutyResourceIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "detector",
			got:  testGuardDutyDetectorID,
			want: testGuardDutyDetectorID,
		},
		{
			name: "filter",
			got:  guardDutyChildResourceID(testGuardDutyDetectorID, testGuardDutyFilterName),
			want: testGuardDutyDetectorID + ":" + testGuardDutyFilterName,
		},
		{
			name: "ip set",
			got:  guardDutyChildResourceID(testGuardDutyDetectorID, testGuardDutyIPSetID),
			want: testGuardDutyDetectorID + ":" + testGuardDutyIPSetID,
		},
		{
			name: "malware protection plan",
			got:  guardDutyMalwareProtectionPlanResourceID(testGuardDutyPlanID),
			want: testGuardDutyPlanID,
		},
		{
			name: "member",
			got:  guardDutyMemberResourceID(testGuardDutyDetectorID, testGuardDutyMemberAccountID),
			want: testGuardDutyDetectorID + ":" + testGuardDutyMemberAccountID,
		},
		{
			name: "organization admin account",
			got:  guardDutyOrganizationAdminAccountResourceID(testGuardDutyOrganizationID),
			want: testGuardDutyOrganizationID,
		},
		{
			name: "organization configuration",
			got:  guardDutyOrganizationConfigurationResourceID(testGuardDutyDetectorID),
			want: testGuardDutyDetectorID,
		},
		{
			name: "publishing destination",
			got:  guardDutyPublishingDestinationResourceID(testGuardDutyDetectorID, testGuardDutyPublishingID),
			want: testGuardDutyDetectorID + ":" + testGuardDutyPublishingID,
		},
		{
			name: "threat intel set",
			got:  guardDutyChildResourceID(testGuardDutyDetectorID, testGuardDutyThreatIntelSetID),
			want: testGuardDutyDetectorID + ":" + testGuardDutyThreatIntelSetID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("resource ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestGuardDutyDetectorResource(t *testing.T) {
	resource, ok := newGuardDutyDetectorResource(testGuardDutyDetectorID, &guardduty.GetDetectorOutput{
		FindingPublishingFrequency: guarddutytypes.FindingPublishingFrequencyFifteenMinutes,
		Status:                     guarddutytypes.DetectorStatusEnabled,
		DataSources: &guarddutytypes.DataSourceConfigurationsResult{
			S3Logs: &guarddutytypes.S3LogsConfigurationResult{Status: guarddutytypes.DataSourceStatusEnabled},
			Kubernetes: &guarddutytypes.KubernetesConfigurationResult{
				AuditLogs: &guarddutytypes.KubernetesAuditLogsConfigurationResult{Status: guarddutytypes.DataSourceStatusDisabled},
			},
			MalwareProtection: &guarddutytypes.MalwareProtectionConfigurationResult{
				ScanEc2InstanceWithFindings: &guarddutytypes.ScanEc2InstanceWithFindingsResult{
					EbsVolumes: &guarddutytypes.EbsVolumesResult{Status: guarddutytypes.DataSourceStatusEnabled},
				},
			},
		},
	})
	if !ok {
		t.Fatal("expected detector resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyDetectorResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"enable":                                         "true",
		"finding_publishing_frequency":                   "FIFTEEN_MINUTES",
		"datasources.#":                                  "1",
		"datasources.0.s3_logs.#":                        "1",
		"datasources.0.s3_logs.0.enable":                 "true",
		"datasources.0.kubernetes.#":                     "1",
		"datasources.0.kubernetes.0.audit_logs.#":        "1",
		"datasources.0.kubernetes.0.audit_logs.0.enable": "false",
		"datasources.0.malware_protection.#":             "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.#":                      "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.#":        "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.0.enable": "true",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestGuardDutyFilterResource(t *testing.T) {
	rank := int32(7)
	greaterThan := int64(1710000000123)
	lessThan := int64(5)
	resource, ok := newGuardDutyFilterResource(testGuardDutyDetectorID, testGuardDutyFilterName, &guardduty.GetFilterOutput{
		Action:      guarddutytypes.FilterActionArchive,
		Description: aws.String("archive known findings"),
		FindingCriteria: &guarddutytypes.FindingCriteria{
			Criterion: map[string]guarddutytypes.Condition{
				"severity": {
					GreaterThanOrEqual: &lessThan,
					NotEquals:          []string{"1"},
				},
				"updatedAt": {
					GreaterThan: &greaterThan,
				},
			},
		},
		Name: aws.String(testGuardDutyFilterName),
		Rank: &rank,
	})
	if !ok {
		t.Fatal("expected filter resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyFilterName; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyFilterResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"action":                               "ARCHIVE",
		"description":                          "archive known findings",
		"detector_id":                          testGuardDutyDetectorID,
		"name":                                 testGuardDutyFilterName,
		"rank":                                 "7",
		"finding_criteria.#":                   "1",
		"finding_criteria.0.criterion.#":       "2",
		"finding_criteria.0.criterion.0.field": "severity",
		"finding_criteria.0.criterion.0.greater_than_or_equal": "5",
		"finding_criteria.0.criterion.0.not_equals.#":          "1",
		"finding_criteria.0.criterion.0.not_equals.0":          "1",
		"finding_criteria.0.criterion.1.field":                 "updatedAt",
		"finding_criteria.0.criterion.1.greater_than":          "2024-03-09T16:00:00.123Z",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	updatedAtCriteria, ok := resource.AdditionalFields[guardDutyFilterUpdatedAtCriteriaAdditionalField].(map[string]string)
	if !ok {
		t.Fatalf("missing updatedAt criteria metadata in %#v", resource.AdditionalFields)
	}
	if got, want := updatedAtCriteria["greater_than"], "2024-03-09T16:00:00.123Z"; got != want {
		t.Fatalf("updatedAt greater_than metadata = %q, want %q", got, want)
	}
}

func TestGuardDutyPostConvertHookPreservesUpdatedAtMillisecondsAfterRefresh(t *testing.T) {
	rank := int32(7)
	greaterThan := int64(1710000000123)
	resource, ok := newGuardDutyFilterResource(testGuardDutyDetectorID, testGuardDutyFilterName, &guardduty.GetFilterOutput{
		Action: guarddutytypes.FilterActionArchive,
		FindingCriteria: &guarddutytypes.FindingCriteria{
			Criterion: map[string]guarddutytypes.Condition{
				"updatedAt": {
					GreaterThan: &greaterThan,
				},
			},
		},
		Name: aws.String(testGuardDutyFilterName),
		Rank: &rank,
	})
	if !ok {
		t.Fatal("expected filter resource")
	}
	resource.InstanceState.Attributes["finding_criteria.0.criterion.0.greater_than"] = "2024-03-09T16:00:00Z"
	resource.InstanceState.SetTypedAttributes(json.RawMessage(`{"finding_criteria":[{"criterion":[{"field":"updatedAt","greater_than":"2024-03-09T16:00:00Z"}]}]}`))
	resource.Item = map[string]interface{}{
		guardDutyFilterUpdatedAtCriteriaAdditionalField: resource.AdditionalFields[guardDutyFilterUpdatedAtCriteriaAdditionalField],
		"finding_criteria": []interface{}{
			map[string]interface{}{
				"criterion": []interface{}{
					map[string]interface{}{
						"field":        "updatedAt",
						"greater_than": "2024-03-09T16:00:00Z",
					},
				},
			},
		},
	}

	g := GuardDutyGenerator{}
	g.Resources = append(g.Resources, resource)
	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	if _, ok := g.Resources[0].Item[guardDutyFilterUpdatedAtCriteriaAdditionalField]; ok {
		t.Fatalf("unexpected metadata in item: %#v", g.Resources[0].Item)
	}
	findingCriteria := g.Resources[0].Item["finding_criteria"].([]interface{})[0].(map[string]interface{})
	criterion := findingCriteria["criterion"].([]interface{})[0].(map[string]interface{})
	if got, want := criterion["greater_than"], "2024-03-09T16:00:00.123Z"; got != want {
		t.Fatalf("greater_than = %q, want %q", got, want)
	}
	if got, want := g.Resources[0].InstanceState.Attributes["finding_criteria.0.criterion.0.greater_than"], "2024-03-09T16:00:00.123Z"; got != want {
		t.Fatalf("state greater_than = %q, want %q", got, want)
	}
	var typedAttributes map[string]interface{}
	if err := json.Unmarshal(g.Resources[0].InstanceState.TypedAttributes, &typedAttributes); err != nil {
		t.Fatalf("TypedAttributes unmarshal error: %v", err)
	}
	typedFindingCriteria := typedAttributes["finding_criteria"].([]interface{})[0].(map[string]interface{})
	typedCriterion := typedFindingCriteria["criterion"].([]interface{})[0].(map[string]interface{})
	if got, want := typedCriterion["greater_than"], "2024-03-09T16:00:00.123Z"; got != want {
		t.Fatalf("typed greater_than = %q, want %q", got, want)
	}
	if !g.Resources[0].InstanceState.HasCurrentTypedAttributes() {
		t.Fatal("typed attributes should track patched flat state")
	}
}

func TestGuardDutyIPSetResource(t *testing.T) {
	resource, ok := newGuardDutyIPSetResource(testGuardDutyDetectorID, testGuardDutyIPSetID, &guardduty.GetIPSetOutput{
		Format:   guarddutytypes.IpSetFormatTxt,
		Location: aws.String("s3://example-bucket/ipset.txt"),
		Name:     aws.String("trusted-ipset"),
		Status:   guarddutytypes.IpSetStatusActive,
	})
	if !ok {
		t.Fatal("expected ipset resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyIPSetID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyIPSetResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"activate":    "true",
		"detector_id": testGuardDutyDetectorID,
		"format":      "TXT",
		"location":    "s3://example-bucket/ipset.txt",
		"name":        "trusted-ipset",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if _, ok := attributes["ip_set_id"]; ok {
		t.Fatalf("unexpected ip_set_id attribute in %#v", attributes)
	}
}

func TestGuardDutyMalwareProtectionPlanResource(t *testing.T) {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	resource, ok := newGuardDutyMalwareProtectionPlanResource(testGuardDutyPlanID, &guardduty.GetMalwareProtectionPlanOutput{
		Actions: &guarddutytypes.MalwareProtectionPlanActions{
			Tagging: &guarddutytypes.MalwareProtectionPlanTaggingAction{Status: guarddutytypes.MalwareProtectionPlanTaggingActionStatusEnabled},
		},
		Arn:       aws.String("arn:aws:guardduty:us-east-1:123456789012:malware-protection-plan/" + testGuardDutyPlanID),
		CreatedAt: &createdAt,
		ProtectedResource: &guarddutytypes.CreateProtectedResource{
			S3Bucket: &guarddutytypes.CreateS3BucketResource{
				BucketName:     aws.String("scan-bucket"),
				ObjectPrefixes: []string{"incoming/", "archive/"},
			},
		},
		Role:   aws.String("arn:aws:iam::123456789012:role/GuardDutyS3MalwareProtection"),
		Status: guarddutytypes.MalwareProtectionPlanStatusActive,
	})
	if !ok {
		t.Fatal("expected malware protection plan resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyPlanID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyMalwareProtectionPlanResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"actions.#":                        "1",
		"actions.0.tagging.#":              "1",
		"actions.0.tagging.0.status":       "ENABLED",
		"arn":                              "arn:aws:guardduty:us-east-1:123456789012:malware-protection-plan/" + testGuardDutyPlanID,
		"created_at":                       "2024-01-02T03:04:05Z",
		"protected_resource.#":             "1",
		"protected_resource.0.s3_bucket.#": "1",
		"protected_resource.0.s3_bucket.0.bucket_name":       "scan-bucket",
		"protected_resource.0.s3_bucket.0.object_prefixes.#": "2",
		"protected_resource.0.s3_bucket.0.object_prefixes.0": "incoming/",
		"protected_resource.0.s3_bucket.0.object_prefixes.1": "archive/",
		"role":   "arn:aws:iam::123456789012:role/GuardDutyS3MalwareProtection",
		"status": "ACTIVE",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestGuardDutyMemberResource(t *testing.T) {
	resource, ok := newGuardDutyMemberResource(testGuardDutyDetectorID, testGuardDutyMemberAccountID, &guarddutytypes.Member{
		AccountId:          aws.String(testGuardDutyMemberAccountID),
		Email:              aws.String("member@example.com"),
		RelationshipStatus: aws.String(guardDutyMemberRelationshipStatusEnabled),
	})
	if !ok {
		t.Fatal("expected member resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyMemberAccountID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyMemberResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"account_id":  testGuardDutyMemberAccountID,
		"detector_id": testGuardDutyDetectorID,
		"email":       "member@example.com",
		"invite":      "true",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}

	resource, ok = newGuardDutyMemberResource(testGuardDutyDetectorID, testGuardDutyMemberAccountID, &guarddutytypes.Member{
		AccountId:          aws.String(testGuardDutyMemberAccountID),
		Email:              aws.String("member@example.com"),
		RelationshipStatus: aws.String(guardDutyMemberRelationshipStatusCreated),
	})
	if !ok {
		t.Fatal("expected created member resource")
	}
	if got, want := resource.InstanceState.Attributes["invite"], "false"; got != want {
		t.Fatalf("created member invite = %q, want %q", got, want)
	}
}

func TestGuardDutyOrganizationAdminAccountResource(t *testing.T) {
	resource, ok := newGuardDutyOrganizationAdminAccountResource(guarddutytypes.AdminAccount{
		AdminAccountId: aws.String(testGuardDutyOrganizationID),
		AdminStatus:    guarddutytypes.AdminStatusEnabled,
	})
	if !ok {
		t.Fatal("expected organization admin account resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyOrganizationID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyOrganizationAdminAccountResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.Attributes["admin_account_id"], testGuardDutyOrganizationID; got != want {
		t.Fatalf("admin_account_id = %q, want %q", got, want)
	}

	if _, ok := newGuardDutyOrganizationAdminAccountResource(guarddutytypes.AdminAccount{
		AdminAccountId: aws.String(testGuardDutyOrganizationID),
		AdminStatus:    guarddutytypes.AdminStatusDisableInProgress,
	}); ok {
		t.Fatal("expected disabled organization admin account to skip")
	}
}

func TestGuardDutyOrganizationConfigurationResource(t *testing.T) {
	resource, ok := newGuardDutyOrganizationConfigurationResource(testGuardDutyDetectorID, &guardduty.DescribeOrganizationConfigurationOutput{
		AutoEnableOrganizationMembers: guarddutytypes.AutoEnableMembersAll,
		DataSources: &guarddutytypes.OrganizationDataSourceConfigurationsResult{
			S3Logs: &guarddutytypes.OrganizationS3LogsConfigurationResult{AutoEnable: aws.Bool(true)},
			Kubernetes: &guarddutytypes.OrganizationKubernetesConfigurationResult{
				AuditLogs: &guarddutytypes.OrganizationKubernetesAuditLogsConfigurationResult{AutoEnable: aws.Bool(false)},
			},
			MalwareProtection: &guarddutytypes.OrganizationMalwareProtectionConfigurationResult{
				ScanEc2InstanceWithFindings: &guarddutytypes.OrganizationScanEc2InstanceWithFindingsResult{
					EbsVolumes: &guarddutytypes.OrganizationEbsVolumesResult{AutoEnable: aws.Bool(true)},
				},
			},
		},
	})
	if !ok {
		t.Fatal("expected organization configuration resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyOrganizationConfigurationResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"auto_enable_organization_members":                                     "ALL",
		"detector_id":                                                          testGuardDutyDetectorID,
		"datasources.#":                                                        "1",
		"datasources.0.s3_logs.#":                                              "1",
		"datasources.0.s3_logs.0.auto_enable":                                  "true",
		"datasources.0.kubernetes.#":                                           "1",
		"datasources.0.kubernetes.0.audit_logs.#":                              "1",
		"datasources.0.kubernetes.0.audit_logs.0.enable":                       "false",
		"datasources.0.malware_protection.#":                                   "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.#": "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.#":             "1",
		"datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.0.auto_enable": "true",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestGuardDutyPublishingDestinationResource(t *testing.T) {
	resource, ok := newGuardDutyPublishingDestinationResource(testGuardDutyDetectorID, testGuardDutyPublishingID, &guardduty.DescribePublishingDestinationOutput{
		DestinationId:   aws.String(testGuardDutyPublishingID),
		DestinationType: guarddutytypes.DestinationTypeS3,
		DestinationProperties: &guarddutytypes.DestinationProperties{
			DestinationArn: aws.String("arn:aws:s3:::guardduty-findings"),
			KmsKeyArn:      aws.String("arn:aws:kms:us-east-1:123456789012:key/00000000-0000-0000-0000-000000000000"),
		},
		Status: guarddutytypes.PublishingStatusPublishing,
	})
	if !ok {
		t.Fatal("expected publishing destination resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyPublishingID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyPublishingDestinationResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"destination_arn":  "arn:aws:s3:::guardduty-findings",
		"destination_id":   testGuardDutyPublishingID,
		"destination_type": "S3",
		"detector_id":      testGuardDutyDetectorID,
		"kms_key_arn":      "arn:aws:kms:us-east-1:123456789012:key/00000000-0000-0000-0000-000000000000",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}

	resource, ok = newGuardDutyPublishingDestinationResource(testGuardDutyDetectorID, testGuardDutyPublishingID, &guardduty.DescribePublishingDestinationOutput{
		DestinationId:   aws.String(testGuardDutyPublishingID),
		DestinationType: guarddutytypes.DestinationTypeS3,
		DestinationProperties: &guarddutytypes.DestinationProperties{
			DestinationArn: aws.String("arn:aws:s3:::guardduty-findings"),
			KmsKeyArn:      aws.String("arn:aws:kms:us-east-1:123456789012:key/00000000-0000-0000-0000-000000000000"),
		},
		Status: guarddutytypes.PublishingStatusPendingVerification,
	})
	if !ok {
		t.Fatal("expected pending publishing destination resource")
	}
	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyPublishingID; got != want {
		t.Fatalf("pending destination resource ID = %q, want %q", got, want)
	}
}

func TestGuardDutyThreatIntelSetResource(t *testing.T) {
	resource, ok := newGuardDutyThreatIntelSetResource(testGuardDutyDetectorID, testGuardDutyThreatIntelSetID, &guardduty.GetThreatIntelSetOutput{
		Format:   guarddutytypes.ThreatIntelSetFormatStix,
		Location: aws.String("s3://example-bucket/threats.stix"),
		Name:     aws.String("known-threats"),
		Status:   guarddutytypes.ThreatIntelSetStatusInactive,
	})
	if !ok {
		t.Fatal("expected threat intel set resource")
	}

	if got, want := resource.InstanceState.ID, testGuardDutyDetectorID+":"+testGuardDutyThreatIntelSetID; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.InstanceInfo.Type, guardDutyThreatIntelSetResourceType; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"activate":    "false",
		"detector_id": testGuardDutyDetectorID,
		"format":      "STIX",
		"location":    "s3://example-bucket/threats.stix",
		"name":        "known-threats",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if _, ok := attributes["threat_intel_set_id"]; ok {
		t.Fatalf("unexpected threat_intel_set_id attribute in %#v", attributes)
	}
}

func TestGuardDutyEmptyIdentifierSkipBehavior(t *testing.T) {
	if _, ok := newGuardDutyDetectorResource("", &guardduty.GetDetectorOutput{}); ok {
		t.Fatal("expected empty detector ID to skip")
	}
	if _, ok := newGuardDutyFilterResource(testGuardDutyDetectorID, "", &guardduty.GetFilterOutput{}); ok {
		t.Fatal("expected empty filter to skip")
	}
	if _, ok := newGuardDutyIPSetResource(testGuardDutyDetectorID, "", &guardduty.GetIPSetOutput{}); ok {
		t.Fatal("expected empty ipset ID to skip")
	}
	if _, ok := newGuardDutyMalwareProtectionPlanResource("", guardDutyTestMalwareProtectionPlan("scan-bucket")); ok {
		t.Fatal("expected empty malware protection plan ID to skip")
	}
	if _, ok := newGuardDutyMemberResource("", testGuardDutyMemberAccountID, guardDutyTestMember(testGuardDutyMemberAccountID)); ok {
		t.Fatal("expected empty member detector ID to skip")
	}
	if _, ok := newGuardDutyMemberResource(testGuardDutyDetectorID, "", guardDutyTestMember(testGuardDutyMemberAccountID)); ok {
		t.Fatal("expected empty member account ID to skip")
	}
	if _, ok := newGuardDutyOrganizationAdminAccountResource(guarddutytypes.AdminAccount{}); ok {
		t.Fatal("expected empty organization admin account ID to skip")
	}
	if _, ok := newGuardDutyOrganizationConfigurationResource("", &guardduty.DescribeOrganizationConfigurationOutput{AutoEnableOrganizationMembers: guarddutytypes.AutoEnableMembersAll}); ok {
		t.Fatal("expected empty organization configuration detector ID to skip")
	}
	if _, ok := newGuardDutyPublishingDestinationResource(testGuardDutyDetectorID, "", guardDutyTestPublishingDestination(testGuardDutyPublishingID)); ok {
		t.Fatal("expected empty publishing destination ID to skip")
	}
	if _, ok := newGuardDutyThreatIntelSetResource(testGuardDutyDetectorID, "", &guardduty.GetThreatIntelSetOutput{}); ok {
		t.Fatal("expected empty threat intel set ID to skip")
	}
}

func TestGuardDutyImportabilityPredicates(t *testing.T) {
	if _, ok := newGuardDutyMalwareProtectionPlanResource(testGuardDutyPlanID, &guardduty.GetMalwareProtectionPlanOutput{
		ProtectedResource: &guarddutytypes.CreateProtectedResource{
			S3Bucket: &guarddutytypes.CreateS3BucketResource{BucketName: aws.String("scan-bucket")},
		},
		Role: aws.String("arn:aws:iam::123456789012:role/GuardDutyS3MalwareProtection"),
	}); ok {
		t.Fatal("expected malware protection plan with empty status to skip")
	}
	if _, ok := newGuardDutyMalwareProtectionPlanResource(testGuardDutyPlanID, &guardduty.GetMalwareProtectionPlanOutput{
		ProtectedResource: &guarddutytypes.CreateProtectedResource{},
		Role:              aws.String("arn:aws:iam::123456789012:role/GuardDutyS3MalwareProtection"),
		Status:            guarddutytypes.MalwareProtectionPlanStatusActive,
	}); ok {
		t.Fatal("expected malware protection plan without S3 bucket to skip")
	}
	if _, ok := newGuardDutyMemberResource(testGuardDutyDetectorID, testGuardDutyMemberAccountID, &guarddutytypes.Member{
		AccountId:          aws.String(testGuardDutyMemberAccountID),
		RelationshipStatus: aws.String(guardDutyMemberRelationshipStatusEnabled),
	}); ok {
		t.Fatal("expected member without email to skip")
	}
	if _, ok := newGuardDutyMemberResource(testGuardDutyDetectorID, testGuardDutyMemberAccountID, &guarddutytypes.Member{
		AccountId:          aws.String(testGuardDutyMemberAccountID),
		Email:              aws.String("member@example.com"),
		RelationshipStatus: aws.String("Removed"),
	}); ok {
		t.Fatal("expected member with unsupported relationship to skip")
	}
	if _, ok := newGuardDutyMemberResource(testGuardDutyDetectorID, testGuardDutyMemberAccountID, &guarddutytypes.Member{
		AccountId:          aws.String("999999999999"),
		Email:              aws.String("member@example.com"),
		RelationshipStatus: aws.String(guardDutyMemberRelationshipStatusEnabled),
	}); ok {
		t.Fatal("expected member with mismatched account ID to skip")
	}
	if _, ok := newGuardDutyOrganizationConfigurationResource(testGuardDutyDetectorID, &guardduty.DescribeOrganizationConfigurationOutput{}); ok {
		t.Fatal("expected organization configuration without auto-enable mode to skip")
	}
	if _, ok := newGuardDutyPublishingDestinationResource(testGuardDutyDetectorID, testGuardDutyPublishingID, &guardduty.DescribePublishingDestinationOutput{
		DestinationId:   aws.String(testGuardDutyPublishingID),
		DestinationType: guarddutytypes.DestinationTypeS3,
		Status:          guarddutytypes.PublishingStatusPublishing,
	}); ok {
		t.Fatal("expected publishing destination without properties to skip")
	}
}

func TestGuardDutyDeletedResourcesAreSkipped(t *testing.T) {
	if _, ok := newGuardDutyIPSetResource(testGuardDutyDetectorID, testGuardDutyIPSetID, &guardduty.GetIPSetOutput{
		Format:   guarddutytypes.IpSetFormatTxt,
		Location: aws.String("s3://example-bucket/ipset.txt"),
		Name:     aws.String("trusted-ipset"),
		Status:   guarddutytypes.IpSetStatusDeleted,
	}); ok {
		t.Fatal("expected deleted ipset to skip")
	}
	if _, ok := newGuardDutyThreatIntelSetResource(testGuardDutyDetectorID, testGuardDutyThreatIntelSetID, &guardduty.GetThreatIntelSetOutput{
		Format:   guarddutytypes.ThreatIntelSetFormatStix,
		Location: aws.String("s3://example-bucket/threats.stix"),
		Name:     aws.String("known-threats"),
		Status:   guarddutytypes.ThreatIntelSetStatusDeleted,
	}); ok {
		t.Fatal("expected deleted threat intel set to skip")
	}
}

func TestGuardDutyResourceNamesDoNotCollapseJoinedParts(t *testing.T) {
	one := guardDutyResourceName("filter", "detector_a", "b")
	two := guardDutyResourceName("filter", "detector", "a_b")
	if one == two {
		t.Fatalf("resource names collapsed before sanitize: %q", one)
	}
	resourceOne, ok := newGuardDutyFilterResource("detector_a", "b", guardDutyTestFilter("b"))
	if !ok {
		t.Fatal("expected first filter resource")
	}
	resourceTwo, ok := newGuardDutyFilterResource("detector", "a_b", guardDutyTestFilter("a_b"))
	if !ok {
		t.Fatal("expected second filter resource")
	}
	if resourceOne.ResourceName == resourceTwo.ResourceName {
		t.Fatalf("resource names collapsed after sanitize: %q", resourceOne.ResourceName)
	}
	memberOne, ok := newGuardDutyMemberResource("detector_a", "b", guardDutyTestMember("b"))
	if !ok {
		t.Fatal("expected first member resource")
	}
	memberTwo, ok := newGuardDutyMemberResource("detector", "a_b", guardDutyTestMember("a_b"))
	if !ok {
		t.Fatal("expected second member resource")
	}
	if memberOne.ResourceName == memberTwo.ResourceName {
		t.Fatalf("member resource names collapsed after sanitize: %q", memberOne.ResourceName)
	}
}

func TestGuardDutyResourceNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no such resource",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected since no such resource found.")},
			want: true,
		},
		{
			name: "detector not owned",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the input detectorId is not owned by the current account.")},
			want: true,
		},
		{
			name: "other bad request",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because a parameter is invalid.")},
			want: false,
		},
		{
			name: "resource not found exception",
			err:  &guarddutytypes.ResourceNotFoundException{Message: aws.String("The requested resource does not exist.")},
			want: true,
		},
		{
			name: "other error type",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardDutyResourceNotFound(tt.err); got != tt.want {
				t.Fatalf("not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGuardDutyPublishingDestinationNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "shared not found",
			err:  &guarddutytypes.ResourceNotFoundException{Message: aws.String("The requested resource does not exist.")},
			want: true,
		},
		{
			name: "stale destination invalid parameter",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because one or more input parameters have invalid values.")},
			want: true,
		},
		{
			name: "other bad request",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because a parameter is invalid.")},
			want: false,
		},
		{
			name: "other error type",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardDutyPublishingDestinationNotFound(tt.err); got != tt.want {
				t.Fatalf("publishing destination not found = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGuardDutyMalwareProtectionPlansUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "access denied",
			err:  &guarddutytypes.AccessDeniedException{Message: aws.String("User is not authorized to perform guardduty:ListMalwareProtectionPlans.")},
			want: true,
		},
		{
			name: "bad request missing malware feature",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("Malware Protection plan APIs are not supported in this account.")},
			want: true,
		},
		{
			name: "other bad request",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because a parameter is invalid.")},
			want: false,
		},
		{
			name: "other error type",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardDutyMalwareProtectionPlansUnavailable(tt.err); got != tt.want {
				t.Fatalf("malware protection plans unavailable = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGuardDutyMemberListingUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "access denied",
			err:  &guarddutytypes.AccessDeniedException{Message: aws.String("User is not authorized to perform guardduty:ListMembers.")},
			want: true,
		},
		{
			name: "not administrator",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the current account is not a GuardDuty administrator account.")},
			want: true,
		},
		{
			name: "administrator only",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("This API can only be called by the GuardDuty administrator account.")},
			want: true,
		},
		{
			name: "legacy not master",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the current account is not the master account.")},
			want: true,
		},
		{
			name: "member account cannot list",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("This operation cannot be called by a member account.")},
			want: true,
		},
		{
			name: "detector not owned is still not member listing",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the input detectorId is not owned by the current account.")},
			want: false,
		},
		{
			name: "other bad request",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because a parameter is invalid.")},
			want: false,
		},
		{
			name: "other error type",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardDutyMemberListingUnavailable(tt.err); got != tt.want {
				t.Fatalf("member listing unavailable = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestGuardDutyOrganizationResourceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "access denied",
			err:  &guarddutytypes.AccessDeniedException{Message: aws.String("User is not authorized to perform guardduty:ListOrganizationAdminAccounts.")},
			want: true,
		},
		{
			name: "not management account",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("Only the organization's management account can run this API operation.")},
			want: true,
		},
		{
			name: "not delegated administrator",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the account is not the delegated administrator for this organization.")},
			want: true,
		},
		{
			name: "delegated administrator not enabled",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("delegated administrator account has not been enabled")},
			want: true,
		},
		{
			name: "not organization member",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because the account is not a member of an organization.")},
			want: true,
		},
		{
			name: "not organization error",
			err:  &guarddutytypes.BadRequestException{Message: aws.String("The request is rejected because a parameter is invalid.")},
			want: false,
		},
		{
			name: "other error type",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardDutyOrganizationResourceUnavailable(tt.err); got != tt.want {
				t.Fatalf("organization unavailable = %t, want %t", got, tt.want)
			}
		})
	}
}

func guardDutyTestFilter(name string) *guardduty.GetFilterOutput {
	rank := int32(1)
	return &guardduty.GetFilterOutput{
		Action: guarddutytypes.FilterActionNoop,
		FindingCriteria: &guarddutytypes.FindingCriteria{
			Criterion: map[string]guarddutytypes.Condition{
				"severity": {Equals: []string{"8"}},
			},
		},
		Name: aws.String(name),
		Rank: &rank,
	}
}

func guardDutyTestMalwareProtectionPlan(bucketName string) *guardduty.GetMalwareProtectionPlanOutput {
	return &guardduty.GetMalwareProtectionPlanOutput{
		ProtectedResource: &guarddutytypes.CreateProtectedResource{
			S3Bucket: &guarddutytypes.CreateS3BucketResource{BucketName: aws.String(bucketName)},
		},
		Role:   aws.String("arn:aws:iam::123456789012:role/GuardDutyS3MalwareProtection"),
		Status: guarddutytypes.MalwareProtectionPlanStatusActive,
	}
}

func guardDutyTestMember(accountID string) *guarddutytypes.Member {
	return &guarddutytypes.Member{
		AccountId:          aws.String(accountID),
		Email:              aws.String("member@example.com"),
		RelationshipStatus: aws.String(guardDutyMemberRelationshipStatusEnabled),
	}
}

func guardDutyTestPublishingDestination(destinationID string) *guardduty.DescribePublishingDestinationOutput {
	return &guardduty.DescribePublishingDestinationOutput{
		DestinationId:   aws.String(destinationID),
		DestinationType: guarddutytypes.DestinationTypeS3,
		DestinationProperties: &guarddutytypes.DestinationProperties{
			DestinationArn: aws.String("arn:aws:s3:::guardduty-findings"),
			KmsKeyArn:      aws.String("arn:aws:kms:us-east-1:123456789012:key/00000000-0000-0000-0000-000000000000"),
		},
		Status: guarddutytypes.PublishingStatusPublishing,
	}
}
