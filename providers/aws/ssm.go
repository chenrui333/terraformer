// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var ssmAllowEmptyValues = []string{"tags."}

var ssmServiceSettingIDs = []string{
	"/ssm/appmanager/appmanager-enabled",
	"/ssm/automation/customer-script-log-destination",
	"/ssm/automation/customer-script-log-group-name",
	"/ssm/automation/enable-adaptive-concurrency",
	"/ssm/documents/console/public-sharing-permission",
	"/ssm/managed-instance/activation-tier",
	"/ssm/managed-instance/default-ec2-instance-management-role",
	"/ssm/opsinsights/opscenter",
	"/ssm/parameter-store/default-parameter-tier",
	"/ssm/parameter-store/high-throughput-enabled",
}

type SsmGenerator struct {
	AWSService
}

type ssmOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *SsmGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ssm.NewFromConfig(config)

	if err := g.addParameters(svc); err != nil {
		return err
	}

	g.loadOptionalResources([]ssmOptionalResourceLoader{
		{name: "documents", load: func() error { return g.addDocuments(svc) }},
		{name: "associations", load: func() error { return g.addAssociations(svc) }},
		{name: "maintenance windows", load: func() error { return g.addMaintenanceWindows(svc) }},
		{name: "patch baselines", load: func() error { return g.addPatchBaselines(svc) }},
		{name: "patch groups", load: func() error { return g.addPatchGroups(svc) }},
		{name: "resource data syncs", load: func() error { return g.addResourceDataSyncs(svc) }},
		{name: "activations", load: func() error { return g.addActivations(svc) }},
		{name: "service settings", load: func() error { return g.addServiceSettings(svc) }},
	})

	return nil
}

func (g *SsmGenerator) addParameters(svc *ssm.Client) error {
	p := ssm.NewDescribeParametersPaginator(svc, &ssm.DescribeParametersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, parameter := range page.Parameters {
			name := StringValue(parameter.Name)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				name,
				name,
				"aws_ssm_parameter",
				"aws",
				ssmAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addDocuments(svc *ssm.Client) error {
	p := ssm.NewListDocumentsPaginator(svc, &ssm.ListDocumentsInput{
		Filters: []ssmtypes.DocumentKeyValuesFilter{
			{
				Key:    aws.String("Owner"),
				Values: []string{"Self"},
			},
		},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, document := range page.DocumentIdentifiers {
			name := StringValue(document.Name)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				name,
				name,
				"aws_ssm_document",
				"aws",
				map[string]string{
					"name": name,
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addAssociations(svc *ssm.Client) error {
	p := ssm.NewListAssociationsPaginator(svc, &ssm.ListAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.Associations {
			associationID := StringValue(association.AssociationId)
			if associationID == "" {
				continue
			}
			documentName := StringValue(association.Name)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				associationID,
				ssmResourceName("association", StringValue(association.AssociationName), documentName, associationID),
				"aws_ssm_association",
				"aws",
				map[string]string{
					"association_id": associationID,
					"name":           documentName,
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addMaintenanceWindows(svc *ssm.Client) error {
	p := ssm.NewDescribeMaintenanceWindowsPaginator(svc, &ssm.DescribeMaintenanceWindowsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, window := range page.WindowIdentities {
			windowID := StringValue(window.WindowId)
			if windowID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				windowID,
				ssmResourceName(StringValue(window.Name), windowID),
				"aws_ssm_maintenance_window",
				"aws",
				map[string]string{
					"name": StringValue(window.Name),
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
			if err := g.addMaintenanceWindowTargets(svc, windowID); err != nil {
				log.Printf("Skipping AWS SSM maintenance window targets for %s: %v", windowID, err)
			}
			if err := g.addMaintenanceWindowTasks(svc, windowID); err != nil {
				log.Printf("Skipping AWS SSM maintenance window tasks for %s: %v", windowID, err)
			}
		}
	}
	return nil
}

func (g *SsmGenerator) addMaintenanceWindowTargets(svc *ssm.Client, windowID string) error {
	p := ssm.NewDescribeMaintenanceWindowTargetsPaginator(svc, &ssm.DescribeMaintenanceWindowTargetsInput{
		WindowId: aws.String(windowID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, target := range page.Targets {
			targetID := StringValue(target.WindowTargetId)
			if targetID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				targetID,
				ssmResourceName(windowID, StringValue(target.Name), targetID),
				"aws_ssm_maintenance_window_target",
				"aws",
				map[string]string{
					"window_id":        windowID,
					"window_target_id": targetID,
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addMaintenanceWindowTasks(svc *ssm.Client, windowID string) error {
	p := ssm.NewDescribeMaintenanceWindowTasksPaginator(svc, &ssm.DescribeMaintenanceWindowTasksInput{
		WindowId: aws.String(windowID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, task := range page.Tasks {
			taskID := StringValue(task.WindowTaskId)
			if taskID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				taskID,
				ssmResourceName(windowID, StringValue(task.Name), taskID),
				"aws_ssm_maintenance_window_task",
				"aws",
				map[string]string{
					"window_id":      windowID,
					"window_task_id": taskID,
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addPatchBaselines(svc *ssm.Client) error {
	p := ssm.NewDescribePatchBaselinesPaginator(svc, &ssm.DescribePatchBaselinesInput{
		Filters: []ssmtypes.PatchOrchestratorFilter{
			{
				Key:    aws.String("OWNER"),
				Values: []string{"Self"},
			},
		},
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, baseline := range page.BaselineIdentities {
			baselineID := StringValue(baseline.BaselineId)
			if baselineID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				baselineID,
				ssmResourceName(StringValue(baseline.BaselineName), baselineID),
				"aws_ssm_patch_baseline",
				"aws",
				map[string]string{
					"name": baselineName(baseline),
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
			if baseline.DefaultBaseline {
				operatingSystem := string(baseline.OperatingSystem)
				if operatingSystem == "" {
					continue
				}
				g.Resources = append(g.Resources, terraformutils.NewResource(
					operatingSystem,
					ssmResourceName("default", operatingSystem, baselineID),
					"aws_ssm_default_patch_baseline",
					"aws",
					map[string]string{
						"baseline_id":      baselineID,
						"operating_system": operatingSystem,
					},
					ssmAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}

func (g *SsmGenerator) addPatchGroups(svc *ssm.Client) error {
	p := ssm.NewDescribePatchGroupsPaginator(svc, &ssm.DescribePatchGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, patchGroup := range page.Mappings {
			name := StringValue(patchGroup.PatchGroup)
			baselineID := patchGroupBaselineID(patchGroup)
			if name == "" || baselineID == "" {
				continue
			}
			importID := ssmPatchGroupImportID(name, baselineID)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				importID,
				ssmResourceName(name, baselineID),
				"aws_ssm_patch_group",
				"aws",
				map[string]string{
					"baseline_id": baselineID,
					"patch_group": name,
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addResourceDataSyncs(svc *ssm.Client) error {
	p := ssm.NewListResourceDataSyncPaginator(svc, &ssm.ListResourceDataSyncInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, sync := range page.ResourceDataSyncItems {
			name := StringValue(sync.SyncName)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				name,
				name,
				"aws_ssm_resource_data_sync",
				"aws",
				ssmAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addActivations(svc *ssm.Client) error {
	p := ssm.NewDescribeActivationsPaginator(svc, &ssm.DescribeActivationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, activation := range page.ActivationList {
			activationID := StringValue(activation.ActivationId)
			if activationID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				activationID,
				ssmResourceName(StringValue(activation.DefaultInstanceName), activationID),
				"aws_ssm_activation",
				"aws",
				map[string]string{
					"name": StringValue(activation.DefaultInstanceName),
				},
				ssmAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *SsmGenerator) addServiceSettings(svc *ssm.Client) error {
	for _, settingID := range ssmServiceSettingIDs {
		output, err := svc.GetServiceSetting(context.TODO(), &ssm.GetServiceSettingInput{
			SettingId: aws.String(settingID),
		})
		if err != nil {
			if ssmServiceSettingMissing(err) {
				continue
			}
			log.Printf("Skipping AWS SSM service setting %s: %v", settingID, err)
			continue
		}
		setting := output.ServiceSetting
		if !ssmServiceSettingImportable(setting) {
			continue
		}
		importID := ssmServiceSettingImportID(setting, settingID)
		g.Resources = append(g.Resources, terraformutils.NewResource(
			importID,
			ssmResourceName("service_setting", settingID),
			"aws_ssm_service_setting",
			"aws",
			map[string]string{
				"setting_id": importID,
			},
			ssmAllowEmptyValues,
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *SsmGenerator) loadOptionalResources(loaders []ssmOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping AWS SSM %s: %v", loader.name, err)
		}
	}
}

func (g *SsmGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_ssm_maintenance_window_target", "aws_ssm_maintenance_window_task":
			g.linkResourceByID(i, "window_id", "aws_ssm_maintenance_window", "id")
		case "aws_ssm_patch_group", "aws_ssm_default_patch_baseline":
			g.linkResourceByID(i, "baseline_id", "aws_ssm_patch_baseline", "id")
		}
	}
	return nil
}

func (g *SsmGenerator) linkResourceByID(index int, fieldName, resourceType, outputField string) {
	value, ok := g.Resources[index].Item[fieldName].(string)
	if !ok || value == "" {
		return
	}
	for _, candidate := range g.Resources {
		if candidate.InstanceInfo.Type != resourceType || candidate.InstanceState.ID != value {
			continue
		}
		g.Resources[index].Item[fieldName] = fmt.Sprintf("$"+"{%s.%s.%s}", resourceType, candidate.ResourceName, outputField)
		return
	}
}

func ssmImportID(parts ...string) string {
	return strings.Join(parts, "/")
}

func ssmPatchGroupImportID(patchGroup, baselineID string) string {
	return fmt.Sprintf("%s,%s", patchGroup, baselineID)
}

func ssmResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) == 0 {
		return "ssm_resource"
	}
	return strings.Join(cleanParts, "_")
}

func baselineName(baseline ssmtypes.PatchBaselineIdentity) string {
	name := StringValue(baseline.BaselineName)
	if name != "" {
		return name
	}
	return StringValue(baseline.BaselineId)
}

func patchGroupBaselineID(patchGroup ssmtypes.PatchGroupPatchBaselineMapping) string {
	if patchGroup.BaselineIdentity == nil {
		return ""
	}
	return StringValue(patchGroup.BaselineIdentity.BaselineId)
}

func ssmServiceSettingImportID(setting *ssmtypes.ServiceSetting, fallback string) string {
	if setting == nil {
		return fallback
	}
	arn := StringValue(setting.ARN)
	if arn != "" {
		return arn
	}
	return fallback
}

func ssmServiceSettingImportable(setting *ssmtypes.ServiceSetting) bool {
	if setting == nil {
		return false
	}
	status := StringValue(setting.Status)
	return status == "Customized" || status == "PendingUpdate"
}

func ssmServiceSettingMissing(err error) bool {
	var notFound *ssmtypes.ServiceSettingNotFound
	return errors.As(err, &notFound)
}
