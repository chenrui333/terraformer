// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	backuptypes "github.com/aws/aws-sdk-go-v2/service/backup/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	backupVaultResourceType                   = "backup_vault"
	backupLogicallyAirGappedVaultResourceType = "backup_logically_air_gapped_vault"
	backupVaultPolicyResourceType             = "backup_vault_policy"
	backupVaultNotificationsResourceType      = "backup_vault_notifications"
	backupVaultLockConfigurationResourceType  = "backup_vault_lock_configuration"
	backupPlanResourceType                    = "backup_plan"
	backupSelectionResourceType               = "backup_selection"
	backupRegionSettingsResourceType          = "backup_region_settings"
	backupGlobalSettingsResourceType          = "backup_global_settings"
	backupFrameworkResourceType               = "backup_framework"
	backupReportPlanResourceType              = "backup_report_plan"
	backupRestoreTestingPlanResourceType      = "backup_restore_testing_plan"
	backupRestoreTestingSelectionResourceType = "backup_restore_testing_selection"
)

var backupAllowEmptyValues = []string{"tags.", "resource_type_opt_in_preference"}

var backupResourceTypes = []string{
	backupVaultResourceType,
	backupLogicallyAirGappedVaultResourceType,
	backupVaultPolicyResourceType,
	backupVaultNotificationsResourceType,
	backupVaultLockConfigurationResourceType,
	backupPlanResourceType,
	backupSelectionResourceType,
	backupRegionSettingsResourceType,
	backupGlobalSettingsResourceType,
	backupFrameworkResourceType,
	backupReportPlanResourceType,
	backupRestoreTestingPlanResourceType,
	backupRestoreTestingSelectionResourceType,
}

var backupVaultResourceTypes = []string{
	backupVaultResourceType,
	backupLogicallyAirGappedVaultResourceType,
	backupVaultPolicyResourceType,
	backupVaultNotificationsResourceType,
	backupVaultLockConfigurationResourceType,
}

var backupPlanResourceTypes = []string{
	backupPlanResourceType,
	backupSelectionResourceType,
}

var backupRestoreTestingResourceTypes = []string{
	backupRestoreTestingPlanResourceType,
	backupRestoreTestingSelectionResourceType,
}

type BackupGenerator struct {
	AWSService
}

type backupOptionalResourceLoader struct {
	name         string
	serviceNames []string
	load         func() error
}

func (g *BackupGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	var filteredResources []terraformutils.Resource
	for _, resource := range g.Resources {
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath == "id" {
				allPredicatesTrue = allPredicatesTrue && backupInitialIDFilterMatchesResource(filter, resource)
			}
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *BackupGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := backup.NewFromConfig(config)

	if g.shouldLoadAnyBackupResourceType(backupVaultResourceTypes...) {
		if err := g.loadBackupVaults(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadAnyBackupResourceType(backupPlanResourceTypes...) {
		if err := g.loadBackupPlans(svc); err != nil {
			return err
		}
	}

	var optionalLoaders []backupOptionalResourceLoader
	if g.shouldLoadAnyBackupResourceType(backupFrameworkResourceType) {
		optionalLoaders = append(optionalLoaders, backupOptionalResourceLoader{
			name:         "frameworks",
			serviceNames: []string{backupFrameworkResourceType},
			load:         func() error { return g.loadBackupFrameworks(svc) },
		})
	}
	if g.shouldLoadAnyBackupResourceType(backupReportPlanResourceType) {
		optionalLoaders = append(optionalLoaders, backupOptionalResourceLoader{
			name:         "report plans",
			serviceNames: []string{backupReportPlanResourceType},
			load:         func() error { return g.loadBackupReportPlans(svc) },
		})
	}
	if g.shouldLoadAnyBackupResourceType(backupRestoreTestingResourceTypes...) {
		optionalLoaders = append(optionalLoaders, backupOptionalResourceLoader{
			name:         "restore testing plans",
			serviceNames: backupRestoreTestingResourceTypes,
			load:         func() error { return g.loadBackupRestoreTestingPlans(svc) },
		})
	}
	if g.shouldLoadAnyBackupResourceType(backupRegionSettingsResourceType) {
		optionalLoaders = append(optionalLoaders, backupOptionalResourceLoader{
			name:         "region settings",
			serviceNames: []string{backupRegionSettingsResourceType},
			load:         func() error { return g.addBackupRegionSettings(svc, config.Region) },
		})
	}
	if g.shouldLoadAnyBackupResourceType(backupGlobalSettingsResourceType) {
		optionalLoaders = append(optionalLoaders, backupOptionalResourceLoader{
			name:         "global settings",
			serviceNames: []string{backupGlobalSettingsResourceType},
			load:         func() error { return g.addBackupGlobalSettings(svc, config) },
		})
	}

	return g.loadOptionalResources(optionalLoaders)
}

func (g *BackupGenerator) loadBackupVaults(svc *backup.Client) error {
	p := backup.NewListBackupVaultsPaginator(svc, &backup.ListBackupVaultsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, vault := range page.BackupVaultList {
			if err := g.addBackupVaultResources(svc, vault); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *BackupGenerator) addBackupVaultResources(svc *backup.Client, vault backuptypes.BackupVaultListMember) error {
	name := StringValue(vault.BackupVaultName)
	if name == "" {
		return nil
	}

	switch vault.VaultType {
	case backuptypes.VaultTypeLogicallyAirGappedBackupVault:
		resource, ok := newBackupLogicallyAirGappedVaultResource(vault)
		if ok && g.shouldAppendBackupResource(backupLogicallyAirGappedVaultResourceType, resource) {
			g.Resources = append(g.Resources, resource)
		}
	case backuptypes.VaultTypeRestoreAccessBackupVault:
		// Restore access backup vaults are AWS-managed vault views, not a Terraform resource shape.
		return nil
	default:
		resource := newBackupVaultResource(name)
		if g.shouldAppendBackupResource(backupVaultResourceType, resource) {
			g.Resources = append(g.Resources, resource)
		}
	}

	if g.shouldLoadBackupVaultChildResource(backupVaultPolicyResourceType, name) {
		if err := g.addBackupVaultPolicy(svc, name); err != nil {
			return err
		}
	}
	if g.shouldLoadBackupVaultChildResource(backupVaultNotificationsResourceType, name) {
		if err := g.addBackupVaultNotifications(svc, name); err != nil {
			return err
		}
	}
	if vault.VaultType != backuptypes.VaultTypeLogicallyAirGappedBackupVault && backupVaultLockConfigured(vault) && g.shouldLoadBackupVaultChildResource(backupVaultLockConfigurationResourceType, name) {
		resource := newBackupVaultLockConfigurationResource(vault)
		if g.shouldAppendBackupResource(backupVaultLockConfigurationResourceType, resource) {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func newBackupVaultResource(name string) terraformutils.Resource {
	return terraformutils.NewResource(
		name,
		name,
		"aws_backup_vault",
		"aws",
		map[string]string{"name": name},
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newBackupLogicallyAirGappedVaultResource(vault backuptypes.BackupVaultListMember) (terraformutils.Resource, bool) {
	name := StringValue(vault.BackupVaultName)
	if !backupLogicallyAirGappedVaultImportable(vault) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":               name,
		"min_retention_days": strconv.FormatInt(*vault.MinRetentionDays, 10),
		"max_retention_days": strconv.FormatInt(*vault.MaxRetentionDays, 10),
	}
	if encryptionKeyARN := StringValue(vault.EncryptionKeyArn); encryptionKeyARN != "" {
		attributes["encryption_key_arn"] = encryptionKeyARN
	}
	return terraformutils.NewResource(
		name,
		name,
		"aws_backup_logically_air_gapped_vault",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBackupVaultChildReferenceResource(serviceName, vaultName string) terraformutils.Resource {
	resourceType := "aws_" + serviceName
	resourceName := vaultName
	if serviceName != backupVaultResourceType {
		resourceName = backupResourceName(vaultName, strings.TrimPrefix(serviceName, "backup_vault_"))
	}
	return terraformutils.NewResource(
		vaultName,
		resourceName,
		resourceType,
		"aws",
		map[string]string{"backup_vault_name": vaultName},
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *BackupGenerator) addBackupVaultPolicy(svc *backup.Client, vaultName string) error {
	output, err := svc.GetBackupVaultAccessPolicy(context.TODO(), &backup.GetBackupVaultAccessPolicyInput{
		BackupVaultName: aws.String(vaultName),
	})
	if err != nil {
		return g.handleOptionalResourceError(backupVaultPolicyResourceType, fmt.Sprintf("vault policy for %s", vaultName), err)
	}
	if output == nil {
		return nil
	}
	policy := StringValue(output.Policy)
	if policy == "" {
		return nil
	}
	resource := terraformutils.NewResource(
		vaultName,
		backupResourceName(vaultName, "policy"),
		"aws_backup_vault_policy",
		"aws",
		map[string]string{
			"backup_vault_name": vaultName,
			"policy":            policy,
		},
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
	if g.shouldAppendBackupResource(backupVaultPolicyResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *BackupGenerator) addBackupVaultNotifications(svc *backup.Client, vaultName string) error {
	output, err := svc.GetBackupVaultNotifications(context.TODO(), &backup.GetBackupVaultNotificationsInput{
		BackupVaultName: aws.String(vaultName),
	})
	if err != nil {
		return g.handleOptionalResourceError(backupVaultNotificationsResourceType, fmt.Sprintf("vault notifications for %s", vaultName), err)
	}
	if output == nil {
		return nil
	}
	if !backupVaultNotificationsConfigured(output) {
		return nil
	}
	attributes := map[string]string{
		"backup_vault_name": vaultName,
		"sns_topic_arn":     StringValue(output.SNSTopicArn),
	}
	for key, value := range backupStringSliceAttributes("backup_vault_events", backupVaultEvents(output.BackupVaultEvents)) {
		attributes[key] = value
	}
	resource := terraformutils.NewResource(
		vaultName,
		backupResourceName(vaultName, "notifications"),
		"aws_backup_vault_notifications",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
	if g.shouldAppendBackupResource(backupVaultNotificationsResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newBackupVaultLockConfigurationResource(vault backuptypes.BackupVaultListMember) terraformutils.Resource {
	name := StringValue(vault.BackupVaultName)
	attributes := map[string]string{"backup_vault_name": name}
	if vault.MinRetentionDays != nil {
		attributes["min_retention_days"] = strconv.FormatInt(*vault.MinRetentionDays, 10)
	}
	if vault.MaxRetentionDays != nil {
		attributes["max_retention_days"] = strconv.FormatInt(*vault.MaxRetentionDays, 10)
	}
	return terraformutils.NewResource(
		name,
		backupResourceName(name, "lock_configuration"),
		"aws_backup_vault_lock_configuration",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *BackupGenerator) loadBackupPlans(svc *backup.Client) error {
	p := backup.NewListBackupPlansPaginator(svc, &backup.ListBackupPlansInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, plan := range page.BackupPlansList {
			planID := StringValue(plan.BackupPlanId)
			if planID == "" {
				continue
			}
			resource := newBackupPlanResource(plan)
			if g.shouldAppendBackupResource(backupPlanResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
			if g.shouldLoadBackupSelections(planID) {
				if err := g.loadBackupSelections(svc, planID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func newBackupPlanResource(plan backuptypes.BackupPlansListMember) terraformutils.Resource {
	planID := StringValue(plan.BackupPlanId)
	name := StringValue(plan.BackupPlanName)
	attributes := map[string]string{}
	if name != "" {
		attributes["name"] = name
	}
	return terraformutils.NewResource(
		planID,
		backupPlanResourceName(planID, name),
		"aws_backup_plan",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *BackupGenerator) loadBackupSelections(svc *backup.Client, planID string) error {
	p := backup.NewListBackupSelectionsPaginator(svc, &backup.ListBackupSelectionsInput{
		BackupPlanId: aws.String(planID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return g.handleOptionalResourceError(backupSelectionResourceType, fmt.Sprintf("selections for plan %s", planID), err)
		}
		for _, selection := range page.BackupSelectionsList {
			selectionID := StringValue(selection.SelectionId)
			if selectionID == "" {
				continue
			}
			resource := newBackupSelectionResource(planID, selection)
			if g.shouldAppendBackupResource(backupSelectionResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newBackupSelectionResource(planID string, selection backuptypes.BackupSelectionsListMember) terraformutils.Resource {
	selectionID := StringValue(selection.SelectionId)
	attributes := map[string]string{
		"plan_id": planID,
	}
	if selectionName := StringValue(selection.SelectionName); selectionName != "" {
		attributes["name"] = selectionName
	}
	if iamRoleARN := StringValue(selection.IamRoleArn); iamRoleARN != "" {
		attributes["iam_role_arn"] = iamRoleARN
	}
	return terraformutils.NewResource(
		selectionID,
		backupSelectionImportID(planID, selectionID),
		"aws_backup_selection",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *BackupGenerator) loadBackupFrameworks(svc *backup.Client) error {
	p := backup.NewListFrameworksPaginator(svc, &backup.ListFrameworksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, framework := range page.Frameworks {
			name := StringValue(framework.FrameworkName)
			if name == "" {
				continue
			}
			resource := terraformutils.NewResource(
				name,
				name,
				"aws_backup_framework",
				"aws",
				map[string]string{"name": name},
				backupAllowEmptyValues,
				map[string]interface{}{},
			)
			if g.shouldAppendBackupResource(backupFrameworkResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *BackupGenerator) loadBackupReportPlans(svc *backup.Client) error {
	p := backup.NewListReportPlansPaginator(svc, &backup.ListReportPlansInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, reportPlan := range page.ReportPlans {
			name := StringValue(reportPlan.ReportPlanName)
			if name == "" {
				continue
			}
			resource := terraformutils.NewResource(
				name,
				name,
				"aws_backup_report_plan",
				"aws",
				map[string]string{"name": name},
				backupAllowEmptyValues,
				map[string]interface{}{},
			)
			if g.shouldAppendBackupResource(backupReportPlanResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *BackupGenerator) loadBackupRestoreTestingPlans(svc *backup.Client) error {
	p := backup.NewListRestoreTestingPlansPaginator(svc, &backup.ListRestoreTestingPlansInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, plan := range page.RestoreTestingPlans {
			name := StringValue(plan.RestoreTestingPlanName)
			if name == "" {
				continue
			}
			attributes := map[string]string{"name": name}
			if scheduleExpression := StringValue(plan.ScheduleExpression); scheduleExpression != "" {
				attributes["schedule_expression"] = scheduleExpression
			}
			if scheduleExpressionTimezone := StringValue(plan.ScheduleExpressionTimezone); scheduleExpressionTimezone != "" {
				attributes["schedule_expression_timezone"] = scheduleExpressionTimezone
			}
			if plan.StartWindowHours != 0 {
				attributes["start_window_hours"] = strconv.FormatInt(int64(plan.StartWindowHours), 10)
			}
			resource := terraformutils.NewResource(
				name,
				name,
				"aws_backup_restore_testing_plan",
				"aws",
				attributes,
				backupAllowEmptyValues,
				map[string]interface{}{},
			)
			if g.shouldAppendBackupResource(backupRestoreTestingPlanResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
			if g.shouldLoadBackupRestoreTestingSelections(name) {
				if err := g.loadBackupRestoreTestingSelections(svc, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *BackupGenerator) loadBackupRestoreTestingSelections(svc *backup.Client, planName string) error {
	p := backup.NewListRestoreTestingSelectionsPaginator(svc, &backup.ListRestoreTestingSelectionsInput{
		RestoreTestingPlanName: aws.String(planName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return g.handleOptionalResourceError(backupRestoreTestingSelectionResourceType, fmt.Sprintf("restore testing selections for plan %s", planName), err)
		}
		for _, selection := range page.RestoreTestingSelections {
			selectionName := StringValue(selection.RestoreTestingSelectionName)
			if selectionName == "" {
				continue
			}
			resource := newBackupRestoreTestingSelectionResource(planName, selection)
			if g.shouldAppendBackupResource(backupRestoreTestingSelectionResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newBackupRestoreTestingSelectionResource(planName string, selection backuptypes.RestoreTestingSelectionForList) terraformutils.Resource {
	selectionName := StringValue(selection.RestoreTestingSelectionName)
	attributes := map[string]string{
		"name":                      selectionName,
		"restore_testing_plan_name": planName,
	}
	if iamRoleARN := StringValue(selection.IamRoleArn); iamRoleARN != "" {
		attributes["iam_role_arn"] = iamRoleARN
	}
	if protectedResourceType := StringValue(selection.ProtectedResourceType); protectedResourceType != "" {
		attributes["protected_resource_type"] = protectedResourceType
	}
	if selection.ValidationWindowHours != 0 {
		attributes["validation_window_hours"] = strconv.FormatInt(int64(selection.ValidationWindowHours), 10)
	}
	importID := backupRestoreTestingSelectionImportID(selectionName, planName)
	return terraformutils.NewResource(
		importID,
		importID,
		"aws_backup_restore_testing_selection",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *BackupGenerator) addBackupRegionSettings(svc *backup.Client, region string) error {
	output, err := svc.DescribeRegionSettings(context.TODO(), &backup.DescribeRegionSettingsInput{})
	if err != nil {
		return err
	}
	if region == "" || !backupRegionSettingsConfigured(output) {
		return nil
	}
	attributes := backupBoolMapAttributes("resource_type_opt_in_preference", output.ResourceTypeOptInPreference)
	for key, value := range backupBoolMapAttributes("resource_type_management_preference", output.ResourceTypeManagementPreference) {
		attributes[key] = value
	}
	resource := terraformutils.NewResource(
		region,
		region,
		"aws_backup_region_settings",
		"aws",
		attributes,
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
	if g.shouldAppendBackupResource(backupRegionSettingsResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *BackupGenerator) addBackupGlobalSettings(svc *backup.Client, config aws.Config) error {
	output, err := svc.DescribeGlobalSettings(context.TODO(), &backup.DescribeGlobalSettingsInput{})
	if err != nil {
		return err
	}
	if len(output.GlobalSettings) == 0 {
		return nil
	}
	accountID, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}
	if StringValue(accountID) == "" {
		return nil
	}
	resource := terraformutils.NewResource(
		StringValue(accountID),
		StringValue(accountID),
		"aws_backup_global_settings",
		"aws",
		backupStringMapAttributes("global_settings", output.GlobalSettings),
		backupAllowEmptyValues,
		map[string]interface{}{},
	)
	if g.shouldAppendBackupResource(backupGlobalSettingsResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *BackupGenerator) loadOptionalResources(loaders []backupOptionalResourceLoader) error {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if g.hasTypedFilterForAny(loader.serviceNames...) {
				return fmt.Errorf("loading Backup %s: %w", loader.name, err)
			}
			log.Printf("Skipping Backup %s: %v", loader.name, err)
		}
	}
	return nil
}

func (g *BackupGenerator) handleOptionalResourceError(serviceName, name string, err error) error {
	if backupResourceMissing(err) {
		return nil
	}
	if g.hasTypedFilterFor(serviceName) {
		return fmt.Errorf("loading Backup %s: %w", name, err)
	}
	log.Printf("Skipping Backup %s: %v", name, err)
	return nil
}

func (g *BackupGenerator) shouldLoadBackupVaultChildResource(serviceName, vaultName string) bool {
	return g.shouldAppendBackupResource(serviceName, newBackupVaultChildReferenceResource(serviceName, vaultName))
}

func (g *BackupGenerator) shouldLoadBackupSelections(planID string) bool {
	if g.hasTypedBackupFilter() && !g.hasTypedFilterFor(backupSelectionResourceType) {
		return g.hasUntypedIDFilter() && g.initialIDFiltersCanMatchBackupSelection(planID, "")
	}
	return g.initialIDFiltersCanMatchBackupSelection(planID, "")
}

func (g *BackupGenerator) shouldLoadBackupRestoreTestingSelections(planName string) bool {
	if g.hasTypedBackupFilter() && !g.hasTypedFilterFor(backupRestoreTestingSelectionResourceType) {
		return g.hasUntypedIDFilter() && g.initialIDFiltersCanMatchBackupRestoreTestingSelection(planName, "")
	}
	return g.initialIDFiltersCanMatchBackupRestoreTestingSelection(planName, "")
}

func (g *BackupGenerator) shouldLoadAnyBackupResourceType(serviceNames ...string) bool {
	if !g.hasTypedBackupFilter() {
		return true
	}
	return g.hasUntypedIDFilter() || g.hasTypedFilterForAny(serviceNames...)
}

func (g *BackupGenerator) shouldAppendBackupResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedBackupFilter() && !g.hasTypedFilterFor(serviceName) {
		if !g.hasUntypedIDFilter() || !g.resourceMatchesUntypedIDFilters(resource) {
			return false
		}
	}
	return g.resourceMatchesInitialIDFilters(serviceName, resource)
}

func (g *BackupGenerator) resourceMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !backupInitialIDFilterMatchesResource(filter, resource) {
			return false
		}
	}
	return true
}

func (g *BackupGenerator) resourceMatchesUntypedIDFilters(resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName != "" || filter.FieldPath != "id" {
			continue
		}
		if !backupInitialIDFilterMatchesResource(filter, resource) {
			return false
		}
	}
	return true
}

func (g *BackupGenerator) initialIDFiltersCanMatchBackupSelection(planID, selectionID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(backupSelectionResourceType) {
			continue
		}
		if !backupSelectionIDFilterCanMatch(filter.AcceptableValues, planID, selectionID) {
			return false
		}
	}
	return true
}

func (g *BackupGenerator) initialIDFiltersCanMatchBackupRestoreTestingSelection(planName, selectionName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(backupRestoreTestingSelectionResourceType) {
			continue
		}
		if !backupRestoreTestingSelectionIDFilterCanMatch(filter.AcceptableValues, planName, selectionName) {
			return false
		}
	}
	return true
}

func (g *BackupGenerator) hasTypedBackupFilter() bool {
	return g.hasTypedFilterForAny(backupResourceTypes...)
}

func (g *BackupGenerator) hasTypedFilterForAny(serviceNames ...string) bool {
	for _, serviceName := range serviceNames {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *BackupGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *BackupGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}

func backupVaultLockConfigured(vault backuptypes.BackupVaultListMember) bool {
	return aws.ToBool(vault.Locked) || vault.LockDate != nil || vault.MinRetentionDays != nil || vault.MaxRetentionDays != nil
}

func backupVaultNotificationsConfigured(output *backup.GetBackupVaultNotificationsOutput) bool {
	return output != nil && StringValue(output.SNSTopicArn) != "" && len(output.BackupVaultEvents) > 0
}

func backupLogicallyAirGappedVaultImportable(vault backuptypes.BackupVaultListMember) bool {
	return vault.VaultType == backuptypes.VaultTypeLogicallyAirGappedBackupVault && StringValue(vault.BackupVaultName) != "" && vault.MinRetentionDays != nil && vault.MaxRetentionDays != nil
}

func backupRegionSettingsConfigured(output *backup.DescribeRegionSettingsOutput) bool {
	return output != nil && (len(output.ResourceTypeOptInPreference) > 0 || len(output.ResourceTypeManagementPreference) > 0)
}

func backupVaultEvents(events []backuptypes.BackupVaultEvent) []string {
	values := make([]string, 0, len(events))
	for _, event := range events {
		if event != "" {
			values = append(values, string(event))
		}
	}
	return values
}

func backupStringSliceAttributes(prefix string, values []string) map[string]string {
	attributes := map[string]string{
		prefix + ".#": strconv.Itoa(len(values)),
	}
	for i, value := range values {
		attributes[fmt.Sprintf("%s.%d", prefix, i)] = value
	}
	return attributes
}

func backupStringMapAttributes(prefix string, values map[string]string) map[string]string {
	attributes := map[string]string{
		prefix + ".%": strconv.Itoa(len(values)),
	}
	for key, value := range values {
		attributes[prefix+"."+key] = value
	}
	return attributes
}

func backupBoolMapAttributes(prefix string, values map[string]bool) map[string]string {
	attributes := map[string]string{
		prefix + ".%": strconv.Itoa(len(values)),
	}
	for key, value := range values {
		attributes[prefix+"."+key] = strconv.FormatBool(value)
	}
	return attributes
}

func backupResourceName(parts ...string) string {
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, "-002F-")
}

func backupPlanResourceName(planID, planName string) string {
	if planName == "" {
		return planID
	}
	return backupResourceName(planName, planID)
}

func backupSelectionImportID(planID, selectionID string) string {
	return planID + "|" + selectionID
}

func backupRestoreTestingSelectionImportID(selectionName, planName string) string {
	return selectionName + ":" + planName
}

func backupSelectionIDFilterCanMatch(values []string, planID, selectionID string) bool {
	for _, value := range values {
		if selectionID != "" {
			if value == selectionID || value == backupSelectionImportID(planID, selectionID) {
				return true
			}
			continue
		}
		if !strings.Contains(value, "|") {
			return true
		}
		if strings.HasPrefix(value, planID+"|") {
			return true
		}
	}
	return false
}

func backupRestoreTestingSelectionIDFilterCanMatch(values []string, planName, selectionName string) bool {
	for _, value := range values {
		if selectionName != "" {
			if value == backupRestoreTestingSelectionImportID(selectionName, planName) {
				return true
			}
			continue
		}
		if strings.HasSuffix(value, ":"+planName) {
			return true
		}
	}
	return false
}

func backupResourceMissing(err error) bool {
	var resourceNotFound *backuptypes.ResourceNotFoundException
	return errors.As(err, &resourceNotFound)
}

func backupInitialIDFilterMatchesResource(filter terraformutils.ResourceFilter, resource terraformutils.Resource) bool {
	serviceName := strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")
	if !filter.IsApplicable(serviceName) {
		return true
	}
	if serviceName == backupSelectionResourceType {
		return backupSelectionIDFilterCanMatch(filter.AcceptableValues, resource.InstanceState.Attributes["plan_id"], resource.InstanceState.ID)
	}
	return filter.Filter(resource)
}

func (g *BackupGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_backup_vault_policy" {
			g.wrapBackupPolicy(i)
		}
	}
	return nil
}

func (g *BackupGenerator) wrapBackupPolicy(resourceIndex int) {
	policy, ok := g.Resources[resourceIndex].Item["policy"].(string)
	if !ok || policy == "" {
		return
	}
	g.Resources[resourceIndex].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
