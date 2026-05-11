// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	guarddutytypes "github.com/aws/aws-sdk-go-v2/service/guardduty/types"
)

const (
	testGuardDutyDetectorID       = "12abc34d567e8fa901bc2d34e56789f0"
	testGuardDutyFilterName       = "CriticalFindings"
	testGuardDutyIPSetID          = "ipset-1234567890abcdef"
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
	greaterThan := int64(1710000000000)
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
		"finding_criteria.0.criterion.1.greater_than":          "2024-03-09T16:00:00Z",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
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
	if _, ok := newGuardDutyThreatIntelSetResource(testGuardDutyDetectorID, "", &guardduty.GetThreatIntelSetOutput{}); ok {
		t.Fatal("expected empty threat intel set ID to skip")
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
