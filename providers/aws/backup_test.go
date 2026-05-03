// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	backuptypes "github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestBackupVaultLockConfigured(t *testing.T) {
	minRetention := int64(7)
	maxRetention := int64(30)
	lockDate := time.Now()

	tests := []struct {
		name  string
		vault backuptypes.BackupVaultListMember
		want  bool
	}{
		{name: "empty vault has no lock config"},
		{name: "locked vault has lock config", vault: backuptypes.BackupVaultListMember{Locked: aws.Bool(true)}, want: true},
		{name: "unlocked vault alone is not lock config", vault: backuptypes.BackupVaultListMember{Locked: aws.Bool(false)}},
		{name: "lock date is lock config", vault: backuptypes.BackupVaultListMember{LockDate: &lockDate}, want: true},
		{name: "minimum retention is lock config", vault: backuptypes.BackupVaultListMember{MinRetentionDays: &minRetention}, want: true},
		{name: "maximum retention is lock config", vault: backuptypes.BackupVaultListMember{MaxRetentionDays: &maxRetention}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backupVaultLockConfigured(tt.vault); got != tt.want {
				t.Fatalf("backupVaultLockConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBackupImportIDs(t *testing.T) {
	if got := backupSelectionImportID("plan-1", "selection-1"); got != "plan-1|selection-1" {
		t.Fatalf("backupSelectionImportID() = %q", got)
	}
	if !backupSelectionIDFilterCanMatch([]string{"plan-1|selection-1"}, "plan-1", "selection-1") {
		t.Fatal("backupSelectionIDFilterCanMatch() should match composite selection ID")
	}
	if !backupSelectionIDFilterCanMatch([]string{"selection-1"}, "plan-1", "selection-1") {
		t.Fatal("backupSelectionIDFilterCanMatch() should match provider state selection ID")
	}
	if !backupSelectionIDFilterCanMatch([]string{"selection-1"}, "plan-1", "") {
		t.Fatal("backupSelectionIDFilterCanMatch() should let raw selection ID filters scan candidate plans")
	}
	if got := backupRestoreTestingSelectionImportID("selection_1", "plan_1"); got != "selection_1:plan_1" {
		t.Fatalf("backupRestoreTestingSelectionImportID() = %q", got)
	}
}

func TestBackupAttributeHelpers(t *testing.T) {
	stringMap := backupStringMapAttributes("global_settings", map[string]string{
		"isCrossAccountBackupEnabled": "true",
		"isMpaEnabled":                "false",
	})
	if stringMap["global_settings.%"] != "2" {
		t.Fatalf("global_settings.%% = %q, want 2", stringMap["global_settings.%"])
	}
	if stringMap["global_settings.isMpaEnabled"] != "false" {
		t.Fatalf("global_settings.isMpaEnabled = %q", stringMap["global_settings.isMpaEnabled"])
	}

	boolMap := backupBoolMapAttributes("resource_type_opt_in_preference", map[string]bool{"EBS": true})
	if boolMap["resource_type_opt_in_preference.%"] != "1" {
		t.Fatalf("resource_type_opt_in_preference.%% = %q, want 1", boolMap["resource_type_opt_in_preference.%"])
	}
	if boolMap["resource_type_opt_in_preference.EBS"] != "true" {
		t.Fatalf("resource_type_opt_in_preference.EBS = %q", boolMap["resource_type_opt_in_preference.EBS"])
	}
	emptyBoolMap := backupBoolMapAttributes("resource_type_opt_in_preference", nil)
	if emptyBoolMap["resource_type_opt_in_preference.%"] != "0" {
		t.Fatalf("empty resource_type_opt_in_preference.%% = %q, want 0", emptyBoolMap["resource_type_opt_in_preference.%"])
	}

	slice := backupStringSliceAttributes("backup_vault_events", []string{"BACKUP_JOB_COMPLETED", "RESTORE_JOB_COMPLETED"})
	if slice["backup_vault_events.#"] != "2" {
		t.Fatalf("backup_vault_events.# = %q, want 2", slice["backup_vault_events.#"])
	}
	if slice["backup_vault_events.1"] != "RESTORE_JOB_COMPLETED" {
		t.Fatalf("backup_vault_events.1 = %q", slice["backup_vault_events.1"])
	}
}

func TestBackupRegionSettingsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		output *backup.DescribeRegionSettingsOutput
		want   bool
	}{
		{name: "nil output"},
		{name: "empty maps", output: &backup.DescribeRegionSettingsOutput{}},
		{
			name: "opt-in settings",
			output: &backup.DescribeRegionSettingsOutput{
				ResourceTypeOptInPreference: map[string]bool{"EBS": true},
			},
			want: true,
		},
		{
			name: "management-only settings",
			output: &backup.DescribeRegionSettingsOutput{
				ResourceTypeManagementPreference: map[string]bool{"DynamoDB": true},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backupRegionSettingsConfigured(tt.output); got != tt.want {
				t.Fatalf("backupRegionSettingsConfigured() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBackupResourceMissing(t *testing.T) {
	if !backupResourceMissing(fmt.Errorf("wrapped: %w", &backuptypes.ResourceNotFoundException{})) {
		t.Fatal("backupResourceMissing() should match wrapped ResourceNotFoundException")
	}
	if backupResourceMissing(errors.New("boom")) {
		t.Fatal("backupResourceMissing() should reject unrelated errors")
	}
}

func TestBackupLogicallyAirGappedVaultResource(t *testing.T) {
	minRetention := int64(7)
	maxRetention := int64(30)
	vault := backuptypes.BackupVaultListMember{
		BackupVaultName:  aws.String("lag-vault"),
		VaultType:        backuptypes.VaultTypeLogicallyAirGappedBackupVault,
		MinRetentionDays: &minRetention,
		MaxRetentionDays: &maxRetention,
	}

	if !backupLogicallyAirGappedVaultImportable(vault) {
		t.Fatal("backupLogicallyAirGappedVaultImportable() = false, want true")
	}
	resource, ok := newBackupLogicallyAirGappedVaultResource(vault)
	if !ok {
		t.Fatal("newBackupLogicallyAirGappedVaultResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != "aws_backup_logically_air_gapped_vault" {
		t.Fatalf("resource type = %q", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.Attributes["min_retention_days"] != "7" {
		t.Fatalf("min_retention_days = %q", resource.InstanceState.Attributes["min_retention_days"])
	}

	vault.MaxRetentionDays = nil
	if backupLogicallyAirGappedVaultImportable(vault) {
		t.Fatal("backupLogicallyAirGappedVaultImportable() should reject missing max retention")
	}
	if _, ok := newBackupLogicallyAirGappedVaultResource(vault); ok {
		t.Fatal("newBackupLogicallyAirGappedVaultResource() should reject missing max retention")
	}
}

func TestBackupFilterGatesVaultChildren(t *testing.T) {
	vault := newBackupVaultResource("vault-a")
	policy := newBackupVaultChildReferenceResource(backupVaultPolicyResourceType, "vault-a")
	otherPolicy := newBackupVaultChildReferenceResource(backupVaultPolicyResourceType, "vault-b")

	tests := []struct {
		name            string
		filters         []terraformutils.ResourceFilter
		appendVault     bool
		appendPolicy    bool
		loadPolicy      bool
		loadOtherPolicy bool
	}{
		{
			name:        "typed vault filter excludes children",
			filters:     []terraformutils.ResourceFilter{{ServiceName: backupVaultResourceType, FieldPath: "id", AcceptableValues: []string{"vault-a"}}},
			appendVault: true,
		},
		{
			name:         "typed child id filter excludes parent and unrelated children",
			filters:      []terraformutils.ResourceFilter{{ServiceName: backupVaultPolicyResourceType, FieldPath: "id", AcceptableValues: []string{"vault-a"}}},
			appendPolicy: true,
			loadPolicy:   true,
		},
		{
			name:            "typed child non-id filter loads child type for all parents",
			filters:         []terraformutils.ResourceFilter{{ServiceName: backupVaultPolicyResourceType, FieldPath: "policy", AcceptableValues: []string{"{}"}}},
			appendPolicy:    true,
			loadPolicy:      true,
			loadOtherPolicy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := BackupGenerator{}
			g.Filter = tt.filters
			if got := g.shouldAppendBackupResource(backupVaultResourceType, vault); got != tt.appendVault {
				t.Fatalf("shouldAppendBackupResource(vault) = %t, want %t", got, tt.appendVault)
			}
			if got := g.shouldAppendBackupResource(backupVaultPolicyResourceType, policy); got != tt.appendPolicy {
				t.Fatalf("shouldAppendBackupResource(policy) = %t, want %t", got, tt.appendPolicy)
			}
			if got := g.shouldLoadBackupVaultChildResource(backupVaultPolicyResourceType, "vault-a"); got != tt.loadPolicy {
				t.Fatalf("shouldLoadBackupVaultChildResource(vault-a) = %t, want %t", got, tt.loadPolicy)
			}
			if got := g.shouldLoadBackupVaultChildResource(backupVaultPolicyResourceType, "vault-b"); got != tt.loadOtherPolicy {
				t.Fatalf("shouldLoadBackupVaultChildResource(vault-b) = %t, want %t", got, tt.loadOtherPolicy)
			}
			if got := g.shouldAppendBackupResource(backupVaultPolicyResourceType, otherPolicy); got && !tt.loadOtherPolicy {
				t.Fatalf("shouldAppendBackupResource(other policy) = true, want false")
			}
		})
	}
}

func TestBackupFilterGatesSelections(t *testing.T) {
	g := BackupGenerator{}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      backupSelectionResourceType,
		FieldPath:        "id",
		AcceptableValues: []string{backupSelectionImportID("plan-a", "selection-a")},
	}}

	if !g.shouldLoadBackupSelections("plan-a") {
		t.Fatal("shouldLoadBackupSelections(plan-a) = false, want true")
	}
	if g.shouldLoadBackupSelections("plan-b") {
		t.Fatal("shouldLoadBackupSelections(plan-b) = true, want false")
	}

	selection := newBackupSelectionResource("plan-a", backuptypes.BackupSelectionsListMember{
		SelectionId:   aws.String("selection-a"),
		SelectionName: aws.String("daily"),
	})
	if selection.InstanceState.ID != "selection-a" {
		t.Fatalf("selection InstanceState.ID = %q, want provider read ID selection-a", selection.InstanceState.ID)
	}
	if !g.shouldAppendBackupResource(backupSelectionResourceType, selection) {
		t.Fatal("shouldAppendBackupResource(selection) = false, want true")
	}
	g.Resources = []terraformutils.Resource{selection}
	g.InitialCleanup()
	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	plan := terraformutils.NewSimpleResource("plan-a", "plan-a", "aws_backup_plan", "aws", backupAllowEmptyValues)
	if g.shouldAppendBackupResource(backupPlanResourceType, plan) {
		t.Fatal("shouldAppendBackupResource(plan) = true, want false for child-only filter")
	}
}

func TestBackupFilterGatesRestoreTestingSelections(t *testing.T) {
	g := BackupGenerator{}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      backupRestoreTestingSelectionResourceType,
		FieldPath:        "id",
		AcceptableValues: []string{backupRestoreTestingSelectionImportID("selection_a", "plan_a")},
	}}

	if !g.shouldLoadBackupRestoreTestingSelections("plan_a") {
		t.Fatal("shouldLoadBackupRestoreTestingSelections(plan_a) = false, want true")
	}
	if g.shouldLoadBackupRestoreTestingSelections("plan_b") {
		t.Fatal("shouldLoadBackupRestoreTestingSelections(plan_b) = true, want false")
	}
}

func TestBackupPostConvertHookWrapsVaultPolicy(t *testing.T) {
	g := BackupGenerator{}
	g.Resources = []terraformutils.Resource{
		terraformutils.NewResource(
			"vault-a",
			"vault-a-policy",
			"aws_backup_vault_policy",
			"aws",
			map[string]string{},
			backupAllowEmptyValues,
			map[string]interface{}{},
		),
	}
	g.Resources[0].Item = map[string]interface{}{
		"policy": "{\"Condition\":{\"StringEquals\":{\"aws:PrincipalTag/name\":\"${aws:username}\"}}}",
	}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	policy, ok := g.Resources[0].Item["policy"].(string)
	if !ok {
		t.Fatal("policy is not a string")
	}
	if !strings.HasPrefix(policy, "<<POLICY\n") || !strings.HasSuffix(policy, "\nPOLICY") {
		t.Fatalf("policy was not wrapped in heredoc: %q", policy)
	}
	if !strings.Contains(policy, "$${aws:username}") {
		t.Fatalf("policy interpolation was not escaped: %q", policy)
	}
}
