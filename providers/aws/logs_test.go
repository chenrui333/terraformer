// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func TestLogsAccountPolicyTypes(t *testing.T) {
	want := []types.PolicyType{
		types.PolicyTypeDataProtectionPolicy,
		types.PolicyTypeSubscriptionFilterPolicy,
		types.PolicyTypeFieldIndexPolicy,
		types.PolicyTypeTransformerPolicy,
	}
	if len(logsAccountPolicyTypes) != len(want) {
		t.Fatalf("logsAccountPolicyTypes length = %d, want %d", len(logsAccountPolicyTypes), len(want))
	}
	for i, policyType := range logsAccountPolicyTypes {
		if policyType != want[i] {
			t.Fatalf("logsAccountPolicyTypes[%d] = %q, want %q", i, policyType, want[i])
		}
	}
}

func TestLogsResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "joins parts", parts: []string{"group", "filter"}, want: "group_filter"},
		{name: "omits empty parts", parts: []string{"", "group", "", "filter"}, want: "group_filter"},
		{name: "empty", parts: []string{"", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logsResourceName(tt.parts...); got != tt.want {
				t.Fatalf("logsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLogsResourceNotFound(t *testing.T) {
	if !logsResourceNotFound(&types.ResourceNotFoundException{}) {
		t.Fatal("logsResourceNotFound() = false for ResourceNotFoundException, want true")
	}
	if logsResourceNotFound(errors.New("boom")) {
		t.Fatal("logsResourceNotFound() = true for generic error, want false")
	}
	if logsResourceNotFound(nil) {
		t.Fatal("logsResourceNotFound() = true for nil, want false")
	}
}

func TestLogsResourcePolicyResource(t *testing.T) {
	policyName := "account-policy"
	policyDocument := "{}"
	resourceArn := "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/example"

	tests := []struct {
		name           string
		policy         types.ResourcePolicy
		wantID         string
		wantName       string
		wantAttributes map[string]string
	}{
		{
			name: "account scoped policy uses policy name",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeAccount,
			},
			wantID:   policyName,
			wantName: policyName,
			wantAttributes: map[string]string{
				"policy_name": policyName,
			},
		},
		{
			name: "resource scoped policy uses resource ARN",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeResource,
				ResourceArn:    &resourceArn,
			},
			wantID:   resourceArn,
			wantName: logsResourceName(policyName, resourceArn),
			wantAttributes: map[string]string{
				"policy_scope": string(types.PolicyScopeResource),
				"resource_arn": resourceArn,
			},
		},
		{
			name: "resource scoped policy without ARN is skipped",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyName:     &policyName,
				PolicyScope:    types.PolicyScopeResource,
			},
		},
		{
			name: "account scoped policy without name is skipped",
			policy: types.ResourcePolicy{
				PolicyDocument: &policyDocument,
				PolicyScope:    types.PolicyScopeAccount,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotName, gotAttributes := logsResourcePolicyResource(tt.policy)
			if gotID != tt.wantID {
				t.Fatalf("logsResourcePolicyResource() id = %q, want %q", gotID, tt.wantID)
			}
			if gotName != tt.wantName {
				t.Fatalf("logsResourcePolicyResource() name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotAttributes) != len(tt.wantAttributes) {
				t.Fatalf("logsResourcePolicyResource() attributes = %#v, want %#v", gotAttributes, tt.wantAttributes)
			}
			for key, want := range tt.wantAttributes {
				if gotAttributes[key] != want {
					t.Fatalf("logsResourcePolicyResource() attributes[%q] = %q, want %q", key, gotAttributes[key], want)
				}
			}
		})
	}
}

func TestNewLogsDestinationPolicyResource(t *testing.T) {
	destinationName := "central-logs"
	accessPolicy := "{}"

	tests := []struct {
		name        string
		destination types.Destination
		wantOK      bool
	}{
		{
			name: "destination with access policy",
			destination: types.Destination{
				AccessPolicy:    &accessPolicy,
				DestinationName: &destinationName,
			},
			wantOK: true,
		},
		{
			name: "destination without access policy is skipped",
			destination: types.Destination{
				DestinationName: &destinationName,
			},
		},
		{
			name: "destination without name is skipped",
			destination: types.Destination{
				AccessPolicy: &accessPolicy,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDestinationPolicyResource(tt.destination)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDestinationPolicyResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != destinationName {
				t.Fatalf("resource ID = %q, want %q", got, destinationName)
			}
			if got := resource.InstanceInfo.Type; got != logsDestinationPolicyResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDestinationPolicyResourceType)
			}
			if got := resource.InstanceState.Attributes["destination_name"]; got != destinationName {
				t.Fatalf("destination_name = %q, want %q", got, destinationName)
			}
		})
	}
}

func TestNewLogsIndexPolicyResource(t *testing.T) {
	logGroupName := "/aws/lambda/example"
	policyDocument := "{}"

	tests := []struct {
		name         string
		logGroupName string
		policy       types.IndexPolicy
		wantOK       bool
	}{
		{
			name:         "log group policy",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceLogGroup,
			},
			wantOK: true,
		},
		{
			name:         "policy without source is accepted",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
			},
			wantOK: true,
		},
		{
			name:         "account policy is skipped",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceAccount,
			},
		},
		{
			name: "empty log group is skipped",
			policy: types.IndexPolicy{
				PolicyDocument: &policyDocument,
				Source:         types.IndexSourceLogGroup,
			},
		},
		{
			name:         "policy without document is skipped",
			logGroupName: logGroupName,
			policy: types.IndexPolicy{
				Source: types.IndexSourceLogGroup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsIndexPolicyResource(tt.logGroupName, tt.policy)
			if ok != tt.wantOK {
				t.Fatalf("newLogsIndexPolicyResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != tt.logGroupName {
				t.Fatalf("resource ID = %q, want %q", got, tt.logGroupName)
			}
			if got := resource.InstanceInfo.Type; got != logsIndexPolicyResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsIndexPolicyResourceType)
			}
			if got := resource.InstanceState.Attributes["log_group_name"]; got != tt.logGroupName {
				t.Fatalf("log_group_name = %q, want %q", got, tt.logGroupName)
			}
		})
	}
}

func TestNewLogsDeliverySourceResource(t *testing.T) {
	sourceName := "api-delivery-source"
	logType := "ACCESS_LOGS"
	resourceArn := "arn:aws:apigateway:us-east-1::/restapis/api-id"

	tests := []struct {
		name   string
		source types.DeliverySource
		wantOK bool
	}{
		{
			name: "active delivery source",
			source: types.DeliverySource{
				LogType:      &logType,
				Name:         &sourceName,
				ResourceArns: []string{resourceArn},
				Status:       types.DeliverySourceStatusActive,
			},
			wantOK: true,
		},
		{
			name: "source without name is skipped",
			source: types.DeliverySource{
				LogType:      &logType,
				ResourceArns: []string{resourceArn},
			},
		},
		{
			name: "source without log type is skipped",
			source: types.DeliverySource{
				Name:         &sourceName,
				ResourceArns: []string{resourceArn},
			},
		},
		{
			name: "source without resource ARN is skipped",
			source: types.DeliverySource{
				LogType: &logType,
				Name:    &sourceName,
			},
		},
		{
			name: "source with deleted backing resource is skipped",
			source: types.DeliverySource{
				LogType:      &logType,
				Name:         &sourceName,
				ResourceArns: []string{resourceArn},
				StatusReason: types.DeliverySourceStatusReasonResourceDeleted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDeliverySourceResource(tt.source)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDeliverySourceResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != sourceName {
				t.Fatalf("resource ID = %q, want %q", got, sourceName)
			}
			if got := resource.InstanceInfo.Type; got != logsDeliverySourceResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDeliverySourceResourceType)
			}
			if got := resource.InstanceState.Attributes["name"]; got != sourceName {
				t.Fatalf("name = %q, want %q", got, sourceName)
			}
			if got := resource.InstanceState.Attributes["log_type"]; got != logType {
				t.Fatalf("log_type = %q, want %q", got, logType)
			}
			if got := resource.InstanceState.Attributes["resource_arn"]; got != resourceArn {
				t.Fatalf("resource_arn = %q, want %q", got, resourceArn)
			}
		})
	}
}

func TestLogsDeliveryResourceNamesPreservePartBoundaries(t *testing.T) {
	logType := "ACCESS_LOGS"
	resourceArn := "arn:aws:apigateway:us-east-1::/restapis/api-id"
	leftName := "a/b"
	rightName := "a-002F-b"

	left, ok := newLogsDeliverySourceResource(types.DeliverySource{
		LogType:      &logType,
		Name:         &leftName,
		ResourceArns: []string{resourceArn},
	})
	if !ok {
		t.Fatal("newLogsDeliverySourceResource() should create left resource")
	}
	right, ok := newLogsDeliverySourceResource(types.DeliverySource{
		LogType:      &logType,
		Name:         &rightName,
		ResourceArns: []string{resourceArn},
	})
	if !ok {
		t.Fatal("newLogsDeliverySourceResource() should create right resource")
	}
	if left.ResourceName == right.ResourceName {
		t.Fatalf("delivery source resource names collide: %q", left.ResourceName)
	}
}

func TestNewLogsDeliveryDestinationResource(t *testing.T) {
	destinationName := "central-delivery-destination"

	tests := []struct {
		name        string
		destination types.DeliveryDestination
		wantOK      bool
	}{
		{
			name: "delivery destination",
			destination: types.DeliveryDestination{
				Name: &destinationName,
			},
			wantOK: true,
		},
		{
			name: "destination without name is skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDeliveryDestinationResource(tt.destination)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDeliveryDestinationResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != destinationName {
				t.Fatalf("resource ID = %q, want %q", got, destinationName)
			}
			if got := resource.InstanceInfo.Type; got != logsDeliveryDestinationResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDeliveryDestinationResourceType)
			}
			if got := resource.InstanceState.Attributes["name"]; got != destinationName {
				t.Fatalf("name = %q, want %q", got, destinationName)
			}
		})
	}
}

func TestNewLogsDeliveryDestinationPolicyResource(t *testing.T) {
	destinationName := "central-delivery-destination"
	policyDocument := "{}"

	tests := []struct {
		name            string
		destinationName string
		policy          types.Policy
		wantOK          bool
	}{
		{
			name:            "delivery destination policy",
			destinationName: destinationName,
			policy: types.Policy{
				DeliveryDestinationPolicy: &policyDocument,
			},
			wantOK: true,
		},
		{
			name: "policy without destination name is skipped",
			policy: types.Policy{
				DeliveryDestinationPolicy: &policyDocument,
			},
		},
		{
			name:            "policy without document is skipped",
			destinationName: destinationName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDeliveryDestinationPolicyResource(tt.destinationName, tt.policy)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDeliveryDestinationPolicyResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != destinationName {
				t.Fatalf("resource ID = %q, want %q", got, destinationName)
			}
			if got := resource.InstanceInfo.Type; got != logsDeliveryDestinationPolicyResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDeliveryDestinationPolicyResourceType)
			}
			if got := resource.InstanceState.Attributes["delivery_destination_name"]; got != destinationName {
				t.Fatalf("delivery_destination_name = %q, want %q", got, destinationName)
			}
			if got := resource.InstanceState.Attributes["delivery_destination_policy"]; got != policyDocument {
				t.Fatalf("delivery_destination_policy = %q, want %q", got, policyDocument)
			}
		})
	}
}

func TestNewLogsDeliveryResource(t *testing.T) {
	deliveryID := "delivery-1234567890"
	sourceName := "api-delivery-source"
	destinationArn := "arn:aws:logs:us-east-1:123456789012:delivery-destination:central"

	tests := []struct {
		name     string
		delivery types.Delivery
		wantOK   bool
	}{
		{
			name: "delivery",
			delivery: types.Delivery{
				DeliveryDestinationArn: &destinationArn,
				DeliverySourceName:     &sourceName,
				Id:                     &deliveryID,
			},
			wantOK: true,
		},
		{
			name: "delivery without ID is skipped",
			delivery: types.Delivery{
				DeliveryDestinationArn: &destinationArn,
				DeliverySourceName:     &sourceName,
			},
		},
		{
			name: "delivery without source is skipped",
			delivery: types.Delivery{
				DeliveryDestinationArn: &destinationArn,
				Id:                     &deliveryID,
			},
		},
		{
			name: "delivery without destination is skipped",
			delivery: types.Delivery{
				DeliverySourceName: &sourceName,
				Id:                 &deliveryID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsDeliveryResource(tt.delivery)
			if ok != tt.wantOK {
				t.Fatalf("newLogsDeliveryResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != deliveryID {
				t.Fatalf("resource ID = %q, want %q", got, deliveryID)
			}
			if got := resource.InstanceInfo.Type; got != logsDeliveryResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsDeliveryResourceType)
			}
			if got := resource.InstanceState.Attributes["delivery_source_name"]; got != sourceName {
				t.Fatalf("delivery_source_name = %q, want %q", got, sourceName)
			}
			if got := resource.InstanceState.Attributes["delivery_destination_arn"]; got != destinationArn {
				t.Fatalf("delivery_destination_arn = %q, want %q", got, destinationArn)
			}
		})
	}
}

func TestNewLogsAnomalyDetectorResource(t *testing.T) {
	detectorARN := "arn:aws:logs:us-east-1:123456789012:anomaly-detector:detector-1"
	logGroupARN := "arn:aws:logs:us-east-1:123456789012:log-group:/aws/lambda/example"
	detectorName := "lambda-errors"

	tests := []struct {
		name        string
		detector    types.AnomalyDetector
		wantOK      bool
		wantEnabled string
	}{
		{
			name: "analyzing detector",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusAnalyzing,
				DetectorName:          &detectorName,
				LogGroupArnList:       []string{logGroupARN},
			},
			wantOK:      true,
			wantEnabled: "true",
		},
		{
			name: "paused detector",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusPaused,
				DetectorName:          &detectorName,
				LogGroupArnList:       []string{logGroupARN},
			},
			wantOK:      true,
			wantEnabled: "false",
		},
		{
			name: "failed detector remains importable",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusFailed,
				DetectorName:          &detectorName,
				LogGroupArnList:       []string{logGroupARN},
			},
			wantOK:      true,
			wantEnabled: "true",
		},
		{
			name: "detector without ARN is skipped",
			detector: types.AnomalyDetector{
				AnomalyDetectorStatus: types.AnomalyDetectorStatusAnalyzing,
				LogGroupArnList:       []string{logGroupARN},
			},
		},
		{
			name: "detector without log group ARN is skipped",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusAnalyzing,
			},
		},
		{
			name: "detector with empty log group ARN is skipped",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusAnalyzing,
				LogGroupArnList:       []string{""},
			},
		},
		{
			name: "detector without status is skipped",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn: &detectorARN,
				LogGroupArnList:    []string{logGroupARN},
			},
		},
		{
			name: "deleted detector is skipped",
			detector: types.AnomalyDetector{
				AnomalyDetectorArn:    &detectorARN,
				AnomalyDetectorStatus: types.AnomalyDetectorStatusDeleted,
				LogGroupArnList:       []string{logGroupARN},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newLogsAnomalyDetectorResource(tt.detector)
			if ok != tt.wantOK {
				t.Fatalf("newLogsAnomalyDetectorResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := resource.InstanceState.ID; got != detectorARN {
				t.Fatalf("resource ID = %q, want %q", got, detectorARN)
			}
			if got := resource.InstanceInfo.Type; got != logsAnomalyDetectorResourceType {
				t.Fatalf("resource type = %q, want %q", got, logsAnomalyDetectorResourceType)
			}
			for key, want := range map[string]string{
				"arn":                  detectorARN,
				"enabled":              tt.wantEnabled,
				"log_group_arn_list.#": "1",
				"log_group_arn_list.0": logGroupARN,
			} {
				if got := resource.InstanceState.Attributes[key]; got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestLogsAnomalyDetectorResourceNamesPreservePartBoundaries(t *testing.T) {
	left := logsAnomalyDetectorResourceName("a/b", "arn:aws:logs:us-east-1:123456789012:anomaly-detector:left")
	right := logsAnomalyDetectorResourceName("a-002F-b", "arn:aws:logs:us-east-1:123456789012:anomaly-detector:left")
	if left == right {
		t.Fatalf("anomaly detector resource names collide: %q", left)
	}
}

func TestLogsAnomalyDetectorEnabledValue(t *testing.T) {
	tests := []struct {
		name        string
		status      types.AnomalyDetectorStatus
		wantEnabled bool
		wantOK      bool
	}{
		{name: "analyzing", status: types.AnomalyDetectorStatusAnalyzing, wantEnabled: true, wantOK: true},
		{name: "initializing", status: types.AnomalyDetectorStatusInitializing, wantEnabled: true, wantOK: true},
		{name: "training", status: types.AnomalyDetectorStatusTraining, wantEnabled: true, wantOK: true},
		{name: "failed", status: types.AnomalyDetectorStatusFailed, wantEnabled: true, wantOK: true},
		{name: "paused", status: types.AnomalyDetectorStatusPaused, wantEnabled: false, wantOK: true},
		{name: "deleted", status: types.AnomalyDetectorStatusDeleted, wantOK: false},
		{name: "empty", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnabled, gotOK := logsAnomalyDetectorEnabledValue(tt.status)
			if gotOK != tt.wantOK {
				t.Fatalf("logsAnomalyDetectorEnabledValue() ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotEnabled != tt.wantEnabled {
				t.Fatalf("logsAnomalyDetectorEnabledValue() enabled = %t, want %t", gotEnabled, tt.wantEnabled)
			}
		})
	}
}
