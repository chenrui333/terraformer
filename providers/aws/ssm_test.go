// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSsmImportID(t *testing.T) {
	got := ssmImportID("mw-1", "target-1")
	want := "mw-1/target-1"
	if got != want {
		t.Fatalf("ssmImportID() = %q, want %q", got, want)
	}
}

func TestSsmPatchGroupImportID(t *testing.T) {
	got := ssmPatchGroupImportID("prod", "pb-123")
	want := "prod,pb-123"
	if got != want {
		t.Fatalf("ssmPatchGroupImportID() = %q, want %q", got, want)
	}
}

func TestSsmResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "window", "", "target"}, want: "window_target"},
		{name: "fallback", parts: nil, want: "ssm_resource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ssmResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("ssmResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSsmBaselineName(t *testing.T) {
	tests := []struct {
		name     string
		baseline ssmtypes.PatchBaselineIdentity
		want     string
	}{
		{
			name: "uses baseline name",
			baseline: ssmtypes.PatchBaselineIdentity{
				BaselineId:   aws.String("pb-123"),
				BaselineName: aws.String("baseline-name"),
			},
			want: "baseline-name",
		},
		{
			name: "falls back to baseline id",
			baseline: ssmtypes.PatchBaselineIdentity{
				BaselineId: aws.String("pb-123"),
			},
			want: "pb-123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := baselineName(tt.baseline)
			if got != tt.want {
				t.Fatalf("baselineName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSsmPatchGroupBaselineID(t *testing.T) {
	if got := patchGroupBaselineID(ssmtypes.PatchGroupPatchBaselineMapping{}); got != "" {
		t.Fatalf("patchGroupBaselineID() = %q, want empty", got)
	}

	got := patchGroupBaselineID(ssmtypes.PatchGroupPatchBaselineMapping{
		BaselineIdentity: &ssmtypes.PatchBaselineIdentity{BaselineId: aws.String("pb-123")},
	})
	want := "pb-123"
	if got != want {
		t.Fatalf("patchGroupBaselineID() = %q, want %q", got, want)
	}
}

func TestSsmResourceDataSyncImportable(t *testing.T) {
	tests := []struct {
		name string
		sync ssmtypes.ResourceDataSyncItem
		want bool
	}{
		{name: "empty", sync: ssmtypes.ResourceDataSyncItem{}, want: false},
		{name: "s3 destination", sync: ssmtypes.ResourceDataSyncItem{S3Destination: &ssmtypes.ResourceDataSyncS3Destination{}}, want: true},
		{name: "sync to destination", sync: ssmtypes.ResourceDataSyncItem{SyncType: aws.String("SyncToDestination")}, want: true},
		{name: "sync from source", sync: ssmtypes.ResourceDataSyncItem{SyncType: aws.String("SyncFromSource")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ssmResourceDataSyncImportable(tt.sync)
			if got != tt.want {
				t.Fatalf("ssmResourceDataSyncImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSsmServiceSettingImportID(t *testing.T) {
	if got := ssmServiceSettingImportID(nil, "/ssm/example"); got != "/ssm/example" {
		t.Fatalf("ssmServiceSettingImportID(nil) = %q, want fallback", got)
	}

	if got := ssmServiceSettingImportID(&ssmtypes.ServiceSetting{}, "/ssm/example"); got != "/ssm/example" {
		t.Fatalf("ssmServiceSettingImportID(empty) = %q, want fallback", got)
	}

	got := ssmServiceSettingImportID(&ssmtypes.ServiceSetting{ARN: aws.String("arn:aws:ssm:us-east-1:123456789012:servicesetting/ssm/example")}, "/ssm/example")
	want := "arn:aws:ssm:us-east-1:123456789012:servicesetting/ssm/example"
	if got != want {
		t.Fatalf("ssmServiceSettingImportID() = %q, want %q", got, want)
	}
}

func TestSsmServiceSettingImportable(t *testing.T) {
	tests := []struct {
		name    string
		setting *ssmtypes.ServiceSetting
		want    bool
	}{
		{name: "nil", setting: nil, want: false},
		{name: "default", setting: &ssmtypes.ServiceSetting{Status: aws.String("Default")}, want: false},
		{name: "customized", setting: &ssmtypes.ServiceSetting{Status: aws.String("Customized")}, want: true},
		{name: "pending update", setting: &ssmtypes.ServiceSetting{Status: aws.String("PendingUpdate")}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ssmServiceSettingImportable(tt.setting)
			if got != tt.want {
				t.Fatalf("ssmServiceSettingImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSsmServiceSettingMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "generic", err: errors.New("boom"), want: false},
		{name: "service setting missing", err: &ssmtypes.ServiceSettingNotFound{}, want: true},
		{name: "wrapped service setting missing", err: errors.Join(errors.New("lookup failed"), &ssmtypes.ServiceSettingNotFound{}), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ssmServiceSettingMissing(tt.err)
			if got != tt.want {
				t.Fatalf("ssmServiceSettingMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSsmPostConvertHookLinksScopedResources(t *testing.T) {
	maintenanceWindow := terraformutils.NewSimpleResource("mw-1", "window", "aws_ssm_maintenance_window", "aws", ssmAllowEmptyValues)
	windowTarget := terraformutils.NewResource(
		"mw-1/target-1",
		"target",
		"aws_ssm_maintenance_window_target",
		"aws",
		map[string]string{"window_id": "mw-1"},
		ssmAllowEmptyValues,
		map[string]interface{}{},
	)
	windowTarget.Item = map[string]interface{}{"window_id": "mw-1"}
	patchBaseline := terraformutils.NewSimpleResource("pb-1", "baseline", "aws_ssm_patch_baseline", "aws", ssmAllowEmptyValues)
	patchGroup := terraformutils.NewResource(
		"prod,pb-1",
		"prod",
		"aws_ssm_patch_group",
		"aws",
		map[string]string{"baseline_id": "pb-1"},
		ssmAllowEmptyValues,
		map[string]interface{}{},
	)
	patchGroup.Item = map[string]interface{}{"baseline_id": "pb-1"}

	g := SsmGenerator{}
	g.Resources = []terraformutils.Resource{maintenanceWindow, windowTarget, patchBaseline, patchGroup}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() returned error: %v", err)
	}

	wantWindowRef := "$" + "{aws_ssm_maintenance_window." + maintenanceWindow.ResourceName + ".id}"
	if got := g.Resources[1].Item["window_id"]; got != wantWindowRef {
		t.Fatalf("window_id = %q, want %q", got, wantWindowRef)
	}

	wantBaselineRef := "$" + "{aws_ssm_patch_baseline." + patchBaseline.ResourceName + ".id}"
	if got := g.Resources[3].Item["baseline_id"]; got != wantBaselineRef {
		t.Fatalf("baseline_id = %q, want %q", got, wantBaselineRef)
	}
}
