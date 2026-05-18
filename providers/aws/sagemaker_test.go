// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSageMakerImportIDs(t *testing.T) {
	userProfileArn := "arn:aws:sagemaker:us-east-1:123456789012:user-profile/d-abc123/alice"
	if got, want := sageMakerUserProfileImportID(userProfileArn), userProfileArn; got != want {
		t.Fatalf("sageMakerUserProfileImportID() = %q, want %q", got, want)
	}
	if got, want := sageMakerImageVersionImportID("studio-image", 7), "studio-image,7"; got != want {
		t.Fatalf("sageMakerImageVersionImportID() = %q, want %q", got, want)
	}
	if got, want := sageMakerServicecatalogPortfolioStatusImportID("us-east-1"), "us-east-1"; got != want {
		t.Fatalf("sageMakerServicecatalogPortfolioStatusImportID() = %q, want %q", got, want)
	}
}

func TestSageMakerResourceNameFallback(t *testing.T) {
	if got := sageMakerResourceName("", ""); got != sageMakerResourceNameFallback {
		t.Fatalf("sageMakerResourceName() = %q, want %q", got, sageMakerResourceNameFallback)
	}
}

func TestSageMakerResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(sageMakerResourceName("ab", "c"))
	second := terraformutils.TfSanitize(sageMakerResourceName("a", "bc"))
	if first == second {
		t.Fatalf("sageMakerResourceName() collision after sanitize: %q", first)
	}

	userProfileFirst := terraformutils.TfSanitize(sageMakerResourceName("user-profile", "d-aaa", "default"))
	userProfileSecond := terraformutils.TfSanitize(sageMakerResourceName("user-profile", "d-bbb", "default"))
	if userProfileFirst == userProfileSecond {
		t.Fatalf("user profile resource names should include parent domain identity: %q", userProfileFirst)
	}

	imageVersionFirst := terraformutils.TfSanitize(sageMakerResourceName("image-version", "image-a", "1"))
	imageVersionSecond := terraformutils.TfSanitize(sageMakerResourceName("image-version", "image-b", "1"))
	if imageVersionFirst == imageVersionSecond {
		t.Fatalf("image version resource names should include parent image identity: %q", imageVersionFirst)
	}

	notebookFirst := terraformutils.TfSanitize(sageMakerResourceName("notebook-instance", "ab", "c"))
	notebookSecond := terraformutils.TfSanitize(sageMakerResourceName("notebook-instance", "a", "bc"))
	if notebookFirst == notebookSecond {
		t.Fatalf("notebook instance resource names should be length-prefixed: %q", notebookFirst)
	}
}

func TestSageMakerPagination(t *testing.T) {
	client := &fakeSageMakerListAlgorithmsClient{
		pages: []*sagemaker.ListAlgorithmsOutput{
			{
				AlgorithmSummaryList: []sagemakertypes.AlgorithmSummary{{AlgorithmName: aws.String("fraud")}},
				NextToken:            aws.String("next"),
			},
			{
				AlgorithmSummaryList: []sagemakertypes.AlgorithmSummary{{AlgorithmName: aws.String("forecast")}},
			},
		},
	}
	algorithms, err := listSageMakerAlgorithms(client)
	if err != nil {
		t.Fatalf("listSageMakerAlgorithms() error = %v", err)
	}
	if len(algorithms) != 2 {
		t.Fatalf("listSageMakerAlgorithms() len = %d, want 2", len(algorithms))
	}
	if client.calls != 2 {
		t.Fatalf("ListAlgorithms calls = %d, want 2", client.calls)
	}
}

func TestSageMakerShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := SageMakerGenerator{}
	for _, serviceName := range sageMakerResourceTypes {
		if !g.shouldLoadSageMakerResource(serviceName) {
			t.Fatalf("without typed filters, %s should be loaded", serviceName)
		}
	}

	for _, typedServiceName := range sageMakerResourceTypes {
		t.Run(typedServiceName, func(t *testing.T) {
			g.Filter = []terraformutils.ResourceFilter{{
				ServiceName:      typedServiceName,
				FieldPath:        "id",
				AcceptableValues: []string{"example-id"},
			}}
			for _, serviceName := range sageMakerResourceTypes {
				got := g.shouldLoadSageMakerResource(serviceName)
				want := serviceName == typedServiceName
				if got != want {
					t.Fatalf("shouldLoadSageMakerResource(%q) = %t, want %t for typed filter %q", serviceName, got, want, typedServiceName)
				}
			}
		})
	}
}

func TestSageMakerShouldLoadResourceAllowsUntypedFilters(t *testing.T) {
	for _, filter := range []terraformutils.ResourceFilter{
		{FieldPath: "id", AcceptableValues: []string{"prod-endpoint"}},
		{FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
	} {
		t.Run(filter.FieldPath, func(t *testing.T) {
			g := SageMakerGenerator{
				AWSService: AWSService{
					Service: terraformutils.Service{
						Filter: []terraformutils.ResourceFilter{
							{
								ServiceName:      "sagemaker_model",
								FieldPath:        "id",
								AcceptableValues: []string{"fraud-model"},
							},
							filter,
						},
					},
				},
			}
			for _, serviceName := range sageMakerResourceTypes {
				if !g.shouldLoadSageMakerResource(serviceName) {
					t.Fatalf("untyped filter should keep %s discovery available", serviceName)
				}
			}
		})
	}
}

func TestNewSageMakerModelServingResources(t *testing.T) {
	model, ok := newSageMakerModelResource(sagemakertypes.ModelSummary{ModelName: aws.String("fraud-model")})
	assertSageMakerResource(t, model, ok, "fraud-model", sageMakerModelResourceType)
	if got := model.InstanceState.Attributes["name"]; got != "fraud-model" {
		t.Fatalf("model name attribute = %q, want fraud-model", got)
	}

	endpointConfig, ok := newSageMakerEndpointConfigurationResource(sagemakertypes.EndpointConfigSummary{EndpointConfigName: aws.String("prod-config")})
	assertSageMakerResource(t, endpointConfig, ok, "prod-config", sageMakerEndpointConfigurationResourceType)
	if !sageMakerTestStringSliceContains(endpointConfig.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("endpoint configuration IgnoreKeys = %v, want ^name_prefix$", endpointConfig.IgnoreKeys)
	}

	endpoint, ok := newSageMakerEndpointResource(sagemakertypes.EndpointSummary{
		EndpointName:   aws.String("prod-endpoint"),
		EndpointStatus: sagemakertypes.EndpointStatusInService,
	})
	assertSageMakerResource(t, endpoint, ok, "prod-endpoint", sageMakerEndpointResourceType)
	if _, ok := newSageMakerEndpointResource(sagemakertypes.EndpointSummary{
		EndpointName:   aws.String("prod-endpoint"),
		EndpointStatus: sagemakertypes.EndpointStatusCreating,
	}); ok {
		t.Fatal("creating endpoint should be skipped")
	}
	if _, ok := newSageMakerModelResource(sagemakertypes.ModelSummary{}); ok {
		t.Fatal("model without name should be skipped")
	}
}

func TestSageMakerPostConvertHookWrapsModelPackageGroupPolicy(t *testing.T) {
	policyText := "{\"Resource\":\"$" + "{aws:PrincipalArn}\"}"
	policy, ok := newSageMakerModelPackageGroupPolicyResource("fraud-packages", aws.String(policyText))
	assertSageMakerResource(t, policy, ok, "fraud-packages", sageMakerModelPackageGroupPolicyResourceType)
	policy.Item = map[string]interface{}{
		"resource_policy": policyText,
	}
	model, ok := newSageMakerModelResource(sagemakertypes.ModelSummary{ModelName: aws.String("fraud-model")})
	assertSageMakerResource(t, model, ok, "fraud-model", sageMakerModelResourceType)
	model.Item = map[string]interface{}{
		"name": "fraud-model",
	}

	generator := SageMakerGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{policy, model},
			},
		},
	}
	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	want := "<<POLICY\n{\"Resource\":\"$" + "$" + "{aws:PrincipalArn}\"}\nPOLICY"
	if got := generator.Resources[0].Item["resource_policy"]; got != want {
		t.Fatalf("resource_policy = %q, want %q", got, want)
	}
	if got := generator.Resources[1].Item["name"]; got != "fraud-model" {
		t.Fatalf("non-policy resource was modified: %q", got)
	}
}

func TestNewSageMakerStudioResources(t *testing.T) {
	domain, ok := newSageMakerDomainResource(sagemakertypes.DomainDetails{
		DomainId:   aws.String("d-abc123"),
		DomainName: aws.String("studio-domain"),
		Status:     sagemakertypes.DomainStatusInService,
	})
	assertSageMakerResource(t, domain, ok, "d-abc123", sageMakerDomainResourceType)
	if got := domain.InstanceState.Attributes["domain_name"]; got != "studio-domain" {
		t.Fatalf("domain_name attribute = %q, want studio-domain", got)
	}

	profileArn := "arn:aws:sagemaker:us-east-1:123456789012:user-profile/d-abc123/alice"
	profile, ok := newSageMakerUserProfileResource(&sagemaker.DescribeUserProfileOutput{
		DomainId:        aws.String("d-abc123"),
		Status:          sagemakertypes.UserProfileStatusInService,
		UserProfileArn:  aws.String(profileArn),
		UserProfileName: aws.String("alice"),
	})
	assertSageMakerResource(t, profile, ok, profileArn, sageMakerUserProfileResourceType)
	if got := profile.InstanceState.Attributes["domain_id"]; got != "d-abc123" {
		t.Fatalf("domain_id attribute = %q, want d-abc123", got)
	}

	space, ok := newSageMakerSpaceResource(&sagemaker.DescribeSpaceOutput{
		DomainId:         aws.String("d-abc123"),
		SpaceArn:         aws.String("arn:aws:sagemaker:us-east-1:123456789012:space/d-abc123/team"),
		SpaceDisplayName: aws.String("Team Space"),
		SpaceName:        aws.String("team"),
		Status:           sagemakertypes.SpaceStatusInService,
	})
	assertSageMakerResource(t, space, ok, "arn:aws:sagemaker:us-east-1:123456789012:space/d-abc123/team", sageMakerSpaceResourceType)
	if got := space.InstanceState.Attributes["space_display_name"]; got != "Team Space" {
		t.Fatalf("space_display_name attribute = %q, want Team Space", got)
	}

	app, ok := newSageMakerAppResource(&sagemaker.DescribeAppOutput{
		AppArn:          aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/alice/JupyterServer/default"),
		AppName:         aws.String("default"),
		AppType:         sagemakertypes.AppTypeJupyterServer,
		DomainId:        aws.String("d-abc123"),
		Status:          sagemakertypes.AppStatusInService,
		UserProfileName: aws.String("alice"),
	})
	assertSageMakerResource(t, app, ok, "arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/alice/JupyterServer/default", sageMakerAppResourceType)
	if got := app.InstanceState.Attributes["user_profile_name"]; got != "alice" {
		t.Fatalf("user_profile_name attribute = %q, want alice", got)
	}

	spaceApp, ok := newSageMakerAppResource(&sagemaker.DescribeAppOutput{
		AppArn:    aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/team/JupyterLab/default"),
		AppName:   aws.String("default"),
		AppType:   sagemakertypes.AppTypeJupyterLab,
		DomainId:  aws.String("d-abc123"),
		SpaceName: aws.String("team"),
		Status:    sagemakertypes.AppStatusInService,
	})
	assertSageMakerResource(t, spaceApp, ok, "arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/team/JupyterLab/default", sageMakerAppResourceType)
	if got := spaceApp.InstanceState.Attributes["space_name"]; got != "team" {
		t.Fatalf("space_name attribute = %q, want team", got)
	}
	userProfileOwnedApp, ok := newSageMakerAppResource(&sagemaker.DescribeAppOutput{
		AppArn:          aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/team/JupyterLab/default"),
		AppName:         aws.String("default"),
		AppType:         sagemakertypes.AppTypeJupyterLab,
		DomainId:        aws.String("d-abc123"),
		Status:          sagemakertypes.AppStatusInService,
		UserProfileName: aws.String("team"),
	})
	assertSageMakerResource(t, userProfileOwnedApp, ok, "arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/team/JupyterLab/default", sageMakerAppResourceType)
	if spaceApp.ResourceName == userProfileOwnedApp.ResourceName {
		t.Fatalf("space-owned and user-profile-owned apps should have distinct resource names: %q", spaceApp.ResourceName)
	}

	if _, ok := newSageMakerDomainResource(sagemakertypes.DomainDetails{
		DomainId:   aws.String("d-abc123"),
		DomainName: aws.String("studio-domain"),
		Status:     sagemakertypes.DomainStatusDeleting,
	}); ok {
		t.Fatal("deleting domain should be skipped")
	}
	if _, ok := newSageMakerUserProfileResource(&sagemaker.DescribeUserProfileOutput{
		DomainId:        aws.String("d-abc123"),
		Status:          sagemakertypes.UserProfileStatusInService,
		UserProfileName: aws.String("alice"),
	}); ok {
		t.Fatal("user profile without ARN should be skipped")
	}
	if _, ok := newSageMakerAppResource(&sagemaker.DescribeAppOutput{
		AppArn:   aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/alice/JupyterServer/default"),
		AppName:  aws.String("default"),
		AppType:  sagemakertypes.AppTypeJupyterServer,
		DomainId: aws.String("d-abc123"),
		Status:   sagemakertypes.AppStatusPending,
	}); ok {
		t.Fatal("pending app should be skipped")
	}
}

func TestNewSageMakerImageAndConfigResources(t *testing.T) {
	appImageConfig, ok := newSageMakerAppImageConfigResource(sagemakertypes.AppImageConfigDetails{
		AppImageConfigName: aws.String("custom-kernel"),
	})
	assertSageMakerResource(t, appImageConfig, ok, "custom-kernel", sageMakerAppImageConfigResourceType)

	image, ok := newSageMakerImageResource(sagemakertypes.Image{
		ImageName:   aws.String("studio-image"),
		ImageStatus: sagemakertypes.ImageStatusCreated,
	})
	assertSageMakerResource(t, image, ok, "studio-image", sageMakerImageResourceType)

	version, ok := newSageMakerImageVersionResource("studio-image", sagemakertypes.ImageVersion{
		ImageVersionStatus: sagemakertypes.ImageVersionStatusCreated,
		Version:            aws.Int32(3),
	})
	assertSageMakerResource(t, version, ok, "studio-image,3", sageMakerImageVersionResourceType)
	if got := version.InstanceState.Attributes["version"]; got != "3" {
		t.Fatalf("version attribute = %q, want 3", got)
	}

	if _, ok := newSageMakerImageResource(sagemakertypes.Image{
		ImageName:   aws.String("studio-image"),
		ImageStatus: sagemakertypes.ImageStatusCreateFailed,
	}); ok {
		t.Fatal("failed image should be skipped")
	}
	if _, ok := newSageMakerImageVersionResource("studio-image", sagemakertypes.ImageVersion{
		ImageVersionStatus: sagemakertypes.ImageVersionStatusCreated,
	}); ok {
		t.Fatal("image version without version number should be skipped")
	}
}

func TestNewSageMakerRegistryWorkflowAndMonitoringResources(t *testing.T) {
	featureGroup, ok := newSageMakerFeatureGroupResource(sagemakertypes.FeatureGroupSummary{
		FeatureGroupName:   aws.String("features"),
		FeatureGroupStatus: sagemakertypes.FeatureGroupStatusCreated,
	})
	assertSageMakerResource(t, featureGroup, ok, "features", sageMakerFeatureGroupResourceType)

	group, ok := newSageMakerModelPackageGroupResource(sagemakertypes.ModelPackageGroupSummary{
		ModelPackageGroupName:   aws.String("fraud-packages"),
		ModelPackageGroupStatus: sagemakertypes.ModelPackageGroupStatusCompleted,
	})
	assertSageMakerResource(t, group, ok, "fraud-packages", sageMakerModelPackageGroupResourceType)

	policy, ok := newSageMakerModelPackageGroupPolicyResource("fraud-packages", aws.String("{\"Version\":\"2012-10-17\",\"Statement\":[]}"))
	assertSageMakerResource(t, policy, ok, "fraud-packages", sageMakerModelPackageGroupPolicyResourceType)
	if got := policy.InstanceState.Attributes["resource_policy"]; got == "" {
		t.Fatal("resource_policy attribute should be seeded")
	}

	pipeline, ok := newSageMakerPipelineResource(&sagemaker.DescribePipelineOutput{
		PipelineName:   aws.String("train-pipeline"),
		PipelineStatus: sagemakertypes.PipelineStatusActive,
	})
	assertSageMakerResource(t, pipeline, ok, "train-pipeline", sageMakerPipelineResourceType)

	project, ok := newSageMakerProjectResource(sagemakertypes.ProjectSummary{
		ProjectName:   aws.String("ml-project"),
		ProjectStatus: sagemakertypes.ProjectStatusUpdateCompleted,
	})
	assertSageMakerResource(t, project, ok, "ml-project", sageMakerProjectResourceType)

	portfolioStatus, ok := newSageMakerServicecatalogPortfolioStatusResource("us-east-1", &sagemaker.GetSagemakerServicecatalogPortfolioStatusOutput{
		Status: sagemakertypes.SagemakerServicecatalogStatusEnabled,
	})
	assertSageMakerResource(t, portfolioStatus, ok, "us-east-1", sageMakerServicecatalogPortfolioStatusType)
	if got := portfolioStatus.InstanceState.Attributes["status"]; got != "Enabled" {
		t.Fatalf("status attribute = %q, want Enabled", got)
	}

	jobDefinition, ok := newSageMakerDataQualityJobDefinitionResource(sagemakertypes.MonitoringJobDefinitionSummary{
		MonitoringJobDefinitionName: aws.String("data-quality"),
	})
	assertSageMakerResource(t, jobDefinition, ok, "data-quality", sageMakerDataQualityJobDefinitionResourceType)

	schedule, ok := newSageMakerMonitoringScheduleResource(sagemakertypes.MonitoringScheduleSummary{
		MonitoringScheduleName:   aws.String("quality-schedule"),
		MonitoringScheduleStatus: sagemakertypes.ScheduleStatusStopped,
	})
	assertSageMakerResource(t, schedule, ok, "quality-schedule", sageMakerMonitoringScheduleResourceType)

	if _, ok := newSageMakerFeatureGroupResource(sagemakertypes.FeatureGroupSummary{
		FeatureGroupName:   aws.String("features"),
		FeatureGroupStatus: sagemakertypes.FeatureGroupStatusCreating,
	}); ok {
		t.Fatal("creating feature group should be skipped")
	}
	if _, ok := newSageMakerPipelineResource(&sagemaker.DescribePipelineOutput{
		PipelineName:   aws.String("train-pipeline"),
		PipelineStatus: sagemakertypes.PipelineStatusDeleting,
	}); ok {
		t.Fatal("deleting pipeline should be skipped")
	}
	if _, ok := newSageMakerServicecatalogPortfolioStatusResource("", &sagemaker.GetSagemakerServicecatalogPortfolioStatusOutput{
		Status: sagemakertypes.SagemakerServicecatalogStatusEnabled,
	}); ok {
		t.Fatal("portfolio status without region should be skipped")
	}
	if _, ok := newSageMakerServicecatalogPortfolioStatusResource("us-east-1", &sagemaker.GetSagemakerServicecatalogPortfolioStatusOutput{}); ok {
		t.Fatal("portfolio status without status should be skipped")
	}
}

func TestNewSageMakerGroundTruthAndLowRiskResources(t *testing.T) {
	lifecycleConfig, ok := newSageMakerStudioLifecycleConfigResource(sagemakertypes.StudioLifecycleConfigDetails{
		StudioLifecycleConfigAppType: sagemakertypes.StudioLifecycleConfigAppTypeJupyterServer,
		StudioLifecycleConfigName:    aws.String("bootstrap"),
	})
	assertSageMakerResource(t, lifecycleConfig, ok, "bootstrap", sageMakerStudioLifecycleConfigResourceType)

	codeRepository, ok := newSageMakerCodeRepositoryResource(sagemakertypes.CodeRepositorySummary{
		CodeRepositoryName: aws.String("notebooks"),
	})
	assertSageMakerResource(t, codeRepository, ok, "notebooks", sageMakerCodeRepositoryResourceType)

	workforce, ok := newSageMakerWorkforceResource(sagemakertypes.Workforce{
		Status:        sagemakertypes.WorkforceStatusActive,
		WorkforceName: aws.String("private"),
	})
	assertSageMakerResource(t, workforce, ok, "private", sageMakerWorkforceResourceType)
	if _, ok := newSageMakerWorkforceResource(sagemakertypes.Workforce{
		OidcConfig:    &sagemakertypes.OidcConfigForResponse{ClientId: aws.String("client")},
		Status:        sagemakertypes.WorkforceStatusActive,
		WorkforceName: aws.String("oidc"),
	}); ok {
		t.Fatal("OIDC workforce should be skipped because client secret is write-only")
	}

	workteam, ok := newSageMakerWorkteamResource(sagemakertypes.Workteam{
		Description:  aws.String("review team"),
		WorkforceArn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:workforce/private"),
		WorkteamName: aws.String("reviewers"),
		MemberDefinitions: []sagemakertypes.MemberDefinition{{
			CognitoMemberDefinition: &sagemakertypes.CognitoMemberDefinition{
				ClientId:  aws.String("client"),
				UserGroup: aws.String("labelers"),
				UserPool:  aws.String("pool"),
			},
		}},
	})
	assertSageMakerResource(t, workteam, ok, "reviewers", sageMakerWorkteamResourceType)
	if got := workteam.InstanceState.Attributes["workforce_name"]; got != "private" {
		t.Fatalf("workforce_name attribute = %q, want private", got)
	}

	flow, ok := newSageMakerFlowDefinitionResource(sagemakertypes.FlowDefinitionSummary{
		FlowDefinitionName:   aws.String("human-loop"),
		FlowDefinitionStatus: sagemakertypes.FlowDefinitionStatusActive,
	})
	assertSageMakerResource(t, flow, ok, "human-loop", sageMakerFlowDefinitionResourceType)

	fleet, ok := newSageMakerDeviceFleetResource(sagemakertypes.DeviceFleetSummary{
		DeviceFleetName: aws.String("edge-devices"),
	})
	assertSageMakerResource(t, fleet, ok, "edge-devices", sageMakerDeviceFleetResourceType)

	if _, ok := newSageMakerWorkforceResource(sagemakertypes.Workforce{
		Status:        sagemakertypes.WorkforceStatusInitializing,
		WorkforceName: aws.String("private"),
	}); ok {
		t.Fatal("initializing workforce should be skipped")
	}
	if _, ok := newSageMakerWorkteamResource(sagemakertypes.Workteam{
		Description:  aws.String("review team"),
		WorkteamName: aws.String("reviewers"),
	}); ok {
		t.Fatal("workteam without member definitions should be skipped")
	}
	if _, ok := newSageMakerFlowDefinitionResource(sagemakertypes.FlowDefinitionSummary{
		FlowDefinitionName:   aws.String("human-loop"),
		FlowDefinitionStatus: sagemakertypes.FlowDefinitionStatusFailed,
	}); ok {
		t.Fatal("failed flow definition should be skipped")
	}
}

func TestNewSageMakerAIControlPlaneResources(t *testing.T) {
	algorithm, ok := newSageMakerAlgorithmResource(sagemakertypes.AlgorithmSummary{
		AlgorithmName:   aws.String("fraud-algorithm"),
		AlgorithmStatus: sagemakertypes.AlgorithmStatusCompleted,
	})
	assertSageMakerResource(t, algorithm, ok, "fraud-algorithm", sageMakerAlgorithmResourceType)
	if got := algorithm.InstanceState.Attributes["algorithm_name"]; got != "fraud-algorithm" {
		t.Fatalf("algorithm_name attribute = %q, want fraud-algorithm", got)
	}

	notebook, ok := newSageMakerNotebookInstanceResource(sagemakertypes.NotebookInstanceSummary{
		NotebookInstanceName:   aws.String("analysis-notebook"),
		NotebookInstanceStatus: sagemakertypes.NotebookInstanceStatusStopped,
	})
	assertSageMakerResource(t, notebook, ok, "analysis-notebook", sageMakerNotebookInstanceResourceType)

	notebookLifecycle, ok := newSageMakerNotebookInstanceLifecycleConfigResource(sagemakertypes.NotebookInstanceLifecycleConfigSummary{
		NotebookInstanceLifecycleConfigName: aws.String("bootstrap-notebook"),
	})
	assertSageMakerResource(t, notebookLifecycle, ok, "bootstrap-notebook", sageMakerNotebookInstanceLifecycleConfigType)

	modelCard, ok := newSageMakerModelCardResource(sagemakertypes.ModelCardSummary{
		ModelCardName:   aws.String("risk-card"),
		ModelCardStatus: sagemakertypes.ModelCardStatusApproved,
	})
	assertSageMakerResource(t, modelCard, ok, "risk-card", sageMakerModelCardResourceType)

	mlflowApp, ok := newSageMakerMLflowAppResource(sagemakertypes.MlflowAppSummary{
		Arn:    aws.String("arn:aws:sagemaker:us-east-1:123456789012:mlflow-app/team/default"),
		Name:   aws.String("default"),
		Status: sagemakertypes.MlflowAppStatusCreated,
	})
	assertSageMakerResource(t, mlflowApp, ok, "arn:aws:sagemaker:us-east-1:123456789012:mlflow-app/team/default", sageMakerMLflowAppResourceType)

	trackingServer, ok := newSageMakerMLflowTrackingServerResource(sagemakertypes.TrackingServerSummary{
		TrackingServerName:   aws.String("team-tracking"),
		TrackingServerStatus: sagemakertypes.TrackingServerStatusStarted,
	})
	assertSageMakerResource(t, trackingServer, ok, "team-tracking", sageMakerMLflowTrackingServerResourceType)

	if _, ok := newSageMakerAlgorithmResource(sagemakertypes.AlgorithmSummary{
		AlgorithmName:   aws.String("fraud-algorithm"),
		AlgorithmStatus: sagemakertypes.AlgorithmStatusInProgress,
	}); ok {
		t.Fatal("in-progress algorithm should be skipped")
	}
	if _, ok := newSageMakerNotebookInstanceResource(sagemakertypes.NotebookInstanceSummary{
		NotebookInstanceName:   aws.String("analysis-notebook"),
		NotebookInstanceStatus: sagemakertypes.NotebookInstanceStatusUpdating,
	}); ok {
		t.Fatal("updating notebook should be skipped")
	}
	if _, ok := newSageMakerMLflowTrackingServerResource(sagemakertypes.TrackingServerSummary{
		TrackingServerName:   aws.String("team-tracking"),
		TrackingServerStatus: sagemakertypes.TrackingServerStatusMaintenanceInProgress,
	}); ok {
		t.Fatal("maintenance-in-progress tracking server should be skipped")
	}
}

func TestSageMakerWorkforceNameFromARN(t *testing.T) {
	if got, want := sageMakerWorkforceNameFromARN("arn:aws:sagemaker:us-east-1:123456789012:workforce/private"), "private"; got != want {
		t.Fatalf("sageMakerWorkforceNameFromARN() = %q, want %q", got, want)
	}
	if got := sageMakerWorkforceNameFromARN("arn:aws:sagemaker:us-east-1:123456789012:workteam/reviewers"); got != "" {
		t.Fatalf("sageMakerWorkforceNameFromARN() = %q, want empty", got)
	}
}

func TestSageMakerImportableStatuses(t *testing.T) {
	if !sageMakerAlgorithmImportable(sagemakertypes.AlgorithmStatusCompleted) || sageMakerAlgorithmImportable(sagemakertypes.AlgorithmStatusInProgress) {
		t.Fatal("algorithm importability should allow Completed only")
	}
	if !sageMakerEndpointImportable(sagemakertypes.EndpointStatusInService) || sageMakerEndpointImportable(sagemakertypes.EndpointStatusFailed) {
		t.Fatal("endpoint importability should allow InService only")
	}
	if !sageMakerDomainImportable(sagemakertypes.DomainStatusInService) || sageMakerDomainImportable(sagemakertypes.DomainStatusUpdating) {
		t.Fatal("domain importability should allow InService only")
	}
	if !sageMakerUserProfileImportable(sagemakertypes.UserProfileStatusInService) || sageMakerUserProfileImportable(sagemakertypes.UserProfileStatusPending) {
		t.Fatal("user profile importability should allow InService only")
	}
	if !sageMakerSpaceImportable(sagemakertypes.SpaceStatusInService) || sageMakerSpaceImportable(sagemakertypes.SpaceStatusFailed) {
		t.Fatal("space importability should allow InService only")
	}
	if !sageMakerAppImportable(sagemakertypes.AppStatusInService) || sageMakerAppImportable(sagemakertypes.AppStatusPending) {
		t.Fatal("app importability should allow InService only")
	}
	if !sageMakerImageImportable(sagemakertypes.ImageStatusCreated) || sageMakerImageImportable(sagemakertypes.ImageStatusCreating) {
		t.Fatal("image importability should allow Created only")
	}
	if !sageMakerImageVersionImportable(sagemakertypes.ImageVersionStatusCreated) || sageMakerImageVersionImportable(sagemakertypes.ImageVersionStatusDeleting) {
		t.Fatal("image version importability should allow Created only")
	}
	if !sageMakerFeatureGroupImportable(sagemakertypes.FeatureGroupStatusCreated) || sageMakerFeatureGroupImportable(sagemakertypes.FeatureGroupStatusCreateFailed) {
		t.Fatal("feature group importability should allow Created only")
	}
	if !sageMakerModelPackageGroupImportable(sagemakertypes.ModelPackageGroupStatusCompleted) || sageMakerModelPackageGroupImportable(sagemakertypes.ModelPackageGroupStatusFailed) {
		t.Fatal("model package group importability should allow Completed only")
	}
	if !sageMakerPipelineImportable(sagemakertypes.PipelineStatusActive) || sageMakerPipelineImportable(sagemakertypes.PipelineStatusDeleting) {
		t.Fatal("pipeline importability should allow Active only")
	}
	if !sageMakerProjectImportable(sagemakertypes.ProjectStatusCreateCompleted) || !sageMakerProjectImportable(sagemakertypes.ProjectStatusUpdateCompleted) || sageMakerProjectImportable(sagemakertypes.ProjectStatusDeleteCompleted) {
		t.Fatal("project importability should allow create/update completed only")
	}
	if !sageMakerMonitoringScheduleImportable(sagemakertypes.ScheduleStatusScheduled) || !sageMakerMonitoringScheduleImportable(sagemakertypes.ScheduleStatusStopped) || sageMakerMonitoringScheduleImportable(sagemakertypes.ScheduleStatusFailed) {
		t.Fatal("monitoring schedule importability should allow scheduled/stopped only")
	}
	if !sageMakerNotebookInstanceImportable(sagemakertypes.NotebookInstanceStatusStopped) || sageMakerNotebookInstanceImportable(sagemakertypes.NotebookInstanceStatusUpdating) {
		t.Fatal("notebook instance importability should allow in-service/stopped only")
	}
	if !sageMakerModelCardImportable(sagemakertypes.ModelCardStatusDraft) || !sageMakerModelCardImportable(sagemakertypes.ModelCardStatusArchived) {
		t.Fatal("model card importability should allow all stable model card statuses")
	}
	if !sageMakerMLflowAppImportable(sagemakertypes.MlflowAppStatusUpdated) || sageMakerMLflowAppImportable(sagemakertypes.MlflowAppStatusUpdateFailed) {
		t.Fatal("MLflow app importability should allow created/updated only")
	}
	if !sageMakerMLflowTrackingServerImportable(sagemakertypes.TrackingServerStatusMaintenanceComplete) || sageMakerMLflowTrackingServerImportable(sagemakertypes.TrackingServerStatusMaintenanceInProgress) {
		t.Fatal("MLflow tracking server importability should allow stable states only")
	}
	if !sageMakerFlowDefinitionImportable(sagemakertypes.FlowDefinitionStatusActive) || sageMakerFlowDefinitionImportable(sagemakertypes.FlowDefinitionStatusDeleting) {
		t.Fatal("flow definition importability should allow Active only")
	}
	if !sageMakerWorkforceImportable(sagemakertypes.WorkforceStatusActive) || sageMakerWorkforceImportable(sagemakertypes.WorkforceStatusFailed) {
		t.Fatal("workforce importability should allow Active only")
	}
}

func TestSageMakerResourceNotFound(t *testing.T) {
	if !sageMakerResourceNotFound(&sagemakertypes.ResourceNotFound{}) {
		t.Fatal("ResourceNotFound should be detected")
	}
	if !sageMakerResourceNotFound(&smithy.GenericAPIError{Code: "ValidationException", Message: "Cannot find resource policy"}) {
		t.Fatal("SageMaker cannot-find validation errors should be detected")
	}
	if sageMakerResourceNotFound(&smithy.GenericAPIError{Code: "ValidationException", Message: "request rejected"}) {
		t.Fatal("non-not-found validation errors should not be detected")
	}
}

func TestSageMakerInitialCleanupHonorsTypedFilters(t *testing.T) {
	resources := sageMakerTestResources(t)
	g := SageMakerGenerator{}
	g.Resources = resources
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "sagemaker_endpoint",
		FieldPath:        "id",
		AcceptableValues: []string{"prod-endpoint"},
	}}

	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceInfo.Type; got != sageMakerEndpointResourceType {
		t.Fatalf("InitialCleanup() kept resource type = %q, want %s", got, sageMakerEndpointResourceType)
	}
}

func TestSageMakerInitialCleanupScopesTypedIDFilters(t *testing.T) {
	resources := sageMakerTestResources(t)
	g := SageMakerGenerator{}
	g.Resources = resources
	g.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "sagemaker_model",
			FieldPath:        "id",
			AcceptableValues: []string{"fraud-model"},
		},
		{
			ServiceName:      "sagemaker_endpoint",
			FieldPath:        "id",
			AcceptableValues: []string{"prod-endpoint"},
		},
		{
			ServiceName:      "bedrock_guardrail",
			FieldPath:        "id",
			AcceptableValues: []string{"gr-123,DRAFT"},
		},
	}

	g.InitialCleanup()

	gotTypes := make([]string, 0, len(g.Resources))
	for _, resource := range g.Resources {
		gotTypes = append(gotTypes, resource.InstanceInfo.Type)
	}
	sort.Strings(gotTypes)
	wantTypes := []string{sageMakerEndpointResourceType, sageMakerModelResourceType}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("InitialCleanup() kept resource types = %v, want %v", gotTypes, wantTypes)
	}
	for i, want := range wantTypes {
		if gotTypes[i] != want {
			t.Fatalf("InitialCleanup() kept resource types = %v, want %v", gotTypes, wantTypes)
		}
	}
}

func TestSageMakerInitialCleanupPreservesGlobalFilters(t *testing.T) {
	resources := sageMakerTestResources(t)
	g := SageMakerGenerator{}
	g.Resources = resources
	g.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "sagemaker_model",
			FieldPath:        "id",
			AcceptableValues: []string{"fraud-model"},
		},
		{
			FieldPath:        "tags.env",
			AcceptableValues: []string{"prod"},
		},
	}

	g.InitialCleanup()

	if len(g.Resources) != len(resources) {
		t.Fatalf("InitialCleanup() resources len = %d, want %d", len(g.Resources), len(resources))
	}
}

func TestSageMakerUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}

	entries, ok := unsupported["resources"].([]interface{})
	if !ok {
		t.Fatal("unsupported resources file is missing resources list")
	}

	resources := make([]string, 0, len(entries))
	found := map[string]bool{
		"aws_sagemaker_human_task_ui": false,
	}
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		resources = append(resources, resource)
		if _, ok := found[resource]; !ok {
			continue
		}
		found[resource] = true
		if serviceFamily, _ := entry["service_family"].(string); serviceFamily != "sagemaker" {
			t.Fatalf("%s service family = %q, want sagemaker", resource, serviceFamily)
		}
		if status, _ := entry["status"].(string); status != "unsupported" {
			t.Fatalf("%s status = %q, want unsupported", resource, status)
		}
		references, _ := entry["references"].([]interface{})
		reason, _ := entry["reason"].(string)
		evidence, _ := entry["evidence"].(string)
		if reason == "" || evidence == "" || len(references) == 0 {
			t.Fatalf("%s unsupported entry is missing reason, evidence, or references", resource)
		}
	}
	for resource, ok := range found {
		if !ok {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
	if !sort.StringsAreSorted(resources) {
		t.Fatalf("unsupported resources are not sorted by resource: %v", resources)
	}
}

func assertSageMakerResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource should be created")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
	if resource.ResourceName == "" {
		t.Fatal("resource name should not be empty")
	}
}

func sageMakerTestResources(t *testing.T) []terraformutils.Resource {
	t.Helper()
	constructors := []struct {
		name string
		make func() (terraformutils.Resource, bool)
	}{
		{name: "algorithm", make: func() (terraformutils.Resource, bool) {
			return newSageMakerAlgorithmResource(sagemakertypes.AlgorithmSummary{AlgorithmName: aws.String("fraud-algorithm"), AlgorithmStatus: sagemakertypes.AlgorithmStatusCompleted})
		}},
		{name: "model", make: func() (terraformutils.Resource, bool) {
			return newSageMakerModelResource(sagemakertypes.ModelSummary{ModelName: aws.String("fraud-model")})
		}},
		{name: "endpoint configuration", make: func() (terraformutils.Resource, bool) {
			return newSageMakerEndpointConfigurationResource(sagemakertypes.EndpointConfigSummary{EndpointConfigName: aws.String("prod-config")})
		}},
		{name: "endpoint", make: func() (terraformutils.Resource, bool) {
			return newSageMakerEndpointResource(sagemakertypes.EndpointSummary{EndpointName: aws.String("prod-endpoint"), EndpointStatus: sagemakertypes.EndpointStatusInService})
		}},
		{name: "domain", make: func() (terraformutils.Resource, bool) {
			return newSageMakerDomainResource(sagemakertypes.DomainDetails{DomainId: aws.String("d-abc123"), DomainName: aws.String("studio-domain"), Status: sagemakertypes.DomainStatusInService})
		}},
		{name: "user profile", make: func() (terraformutils.Resource, bool) {
			return newSageMakerUserProfileResource(&sagemaker.DescribeUserProfileOutput{DomainId: aws.String("d-abc123"), Status: sagemakertypes.UserProfileStatusInService, UserProfileArn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:user-profile/d-abc123/alice"), UserProfileName: aws.String("alice")})
		}},
		{name: "space", make: func() (terraformutils.Resource, bool) {
			return newSageMakerSpaceResource(&sagemaker.DescribeSpaceOutput{DomainId: aws.String("d-abc123"), SpaceArn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:space/d-abc123/team"), SpaceName: aws.String("team"), Status: sagemakertypes.SpaceStatusInService})
		}},
		{name: "app", make: func() (terraformutils.Resource, bool) {
			return newSageMakerAppResource(&sagemaker.DescribeAppOutput{AppArn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:app/d-abc123/alice/JupyterServer/default"), AppName: aws.String("default"), AppType: sagemakertypes.AppTypeJupyterServer, DomainId: aws.String("d-abc123"), Status: sagemakertypes.AppStatusInService, UserProfileName: aws.String("alice")})
		}},
		{name: "app image config", make: func() (terraformutils.Resource, bool) {
			return newSageMakerAppImageConfigResource(sagemakertypes.AppImageConfigDetails{AppImageConfigName: aws.String("custom-kernel")})
		}},
		{name: "image", make: func() (terraformutils.Resource, bool) {
			return newSageMakerImageResource(sagemakertypes.Image{ImageName: aws.String("studio-image"), ImageStatus: sagemakertypes.ImageStatusCreated})
		}},
		{name: "image version", make: func() (terraformutils.Resource, bool) {
			return newSageMakerImageVersionResource("studio-image", sagemakertypes.ImageVersion{ImageVersionStatus: sagemakertypes.ImageVersionStatusCreated, Version: aws.Int32(3)})
		}},
		{name: "studio lifecycle config", make: func() (terraformutils.Resource, bool) {
			return newSageMakerStudioLifecycleConfigResource(sagemakertypes.StudioLifecycleConfigDetails{StudioLifecycleConfigAppType: sagemakertypes.StudioLifecycleConfigAppTypeJupyterServer, StudioLifecycleConfigName: aws.String("bootstrap")})
		}},
		{name: "code repository", make: func() (terraformutils.Resource, bool) {
			return newSageMakerCodeRepositoryResource(sagemakertypes.CodeRepositorySummary{CodeRepositoryName: aws.String("notebooks")})
		}},
		{name: "notebook instance", make: func() (terraformutils.Resource, bool) {
			return newSageMakerNotebookInstanceResource(sagemakertypes.NotebookInstanceSummary{NotebookInstanceName: aws.String("analysis-notebook"), NotebookInstanceStatus: sagemakertypes.NotebookInstanceStatusInService})
		}},
		{name: "notebook instance lifecycle config", make: func() (terraformutils.Resource, bool) {
			return newSageMakerNotebookInstanceLifecycleConfigResource(sagemakertypes.NotebookInstanceLifecycleConfigSummary{NotebookInstanceLifecycleConfigName: aws.String("bootstrap-notebook")})
		}},
		{name: "feature group", make: func() (terraformutils.Resource, bool) {
			return newSageMakerFeatureGroupResource(sagemakertypes.FeatureGroupSummary{FeatureGroupName: aws.String("features"), FeatureGroupStatus: sagemakertypes.FeatureGroupStatusCreated})
		}},
		{name: "model package group", make: func() (terraformutils.Resource, bool) {
			return newSageMakerModelPackageGroupResource(sagemakertypes.ModelPackageGroupSummary{ModelPackageGroupName: aws.String("fraud-packages"), ModelPackageGroupStatus: sagemakertypes.ModelPackageGroupStatusCompleted})
		}},
		{name: "model package group policy", make: func() (terraformutils.Resource, bool) {
			return newSageMakerModelPackageGroupPolicyResource("fraud-packages", aws.String("{\"Version\":\"2012-10-17\",\"Statement\":[]}"))
		}},
		{name: "pipeline", make: func() (terraformutils.Resource, bool) {
			return newSageMakerPipelineResource(&sagemaker.DescribePipelineOutput{PipelineName: aws.String("train-pipeline"), PipelineStatus: sagemakertypes.PipelineStatusActive})
		}},
		{name: "project", make: func() (terraformutils.Resource, bool) {
			return newSageMakerProjectResource(sagemakertypes.ProjectSummary{ProjectName: aws.String("ml-project"), ProjectStatus: sagemakertypes.ProjectStatusCreateCompleted})
		}},
		{name: "servicecatalog portfolio status", make: func() (terraformutils.Resource, bool) {
			return newSageMakerServicecatalogPortfolioStatusResource("us-east-1", &sagemaker.GetSagemakerServicecatalogPortfolioStatusOutput{Status: sagemakertypes.SagemakerServicecatalogStatusEnabled})
		}},
		{name: "data quality job definition", make: func() (terraformutils.Resource, bool) {
			return newSageMakerDataQualityJobDefinitionResource(sagemakertypes.MonitoringJobDefinitionSummary{MonitoringJobDefinitionName: aws.String("data-quality")})
		}},
		{name: "monitoring schedule", make: func() (terraformutils.Resource, bool) {
			return newSageMakerMonitoringScheduleResource(sagemakertypes.MonitoringScheduleSummary{MonitoringScheduleName: aws.String("quality-schedule"), MonitoringScheduleStatus: sagemakertypes.ScheduleStatusScheduled})
		}},
		{name: "model card", make: func() (terraformutils.Resource, bool) {
			return newSageMakerModelCardResource(sagemakertypes.ModelCardSummary{ModelCardName: aws.String("risk-card"), ModelCardStatus: sagemakertypes.ModelCardStatusApproved})
		}},
		{name: "mlflow app", make: func() (terraformutils.Resource, bool) {
			return newSageMakerMLflowAppResource(sagemakertypes.MlflowAppSummary{Arn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:mlflow-app/team/default"), Name: aws.String("default"), Status: sagemakertypes.MlflowAppStatusCreated})
		}},
		{name: "mlflow tracking server", make: func() (terraformutils.Resource, bool) {
			return newSageMakerMLflowTrackingServerResource(sagemakertypes.TrackingServerSummary{TrackingServerName: aws.String("team-tracking"), TrackingServerStatus: sagemakertypes.TrackingServerStatusStarted})
		}},
		{name: "workforce", make: func() (terraformutils.Resource, bool) {
			return newSageMakerWorkforceResource(sagemakertypes.Workforce{Status: sagemakertypes.WorkforceStatusActive, WorkforceName: aws.String("private")})
		}},
		{name: "workteam", make: func() (terraformutils.Resource, bool) {
			return newSageMakerWorkteamResource(sagemakertypes.Workteam{Description: aws.String("review team"), WorkforceArn: aws.String("arn:aws:sagemaker:us-east-1:123456789012:workforce/private"), WorkteamName: aws.String("reviewers"), MemberDefinitions: []sagemakertypes.MemberDefinition{{CognitoMemberDefinition: &sagemakertypes.CognitoMemberDefinition{ClientId: aws.String("client"), UserGroup: aws.String("labelers"), UserPool: aws.String("pool")}}}})
		}},
		{name: "flow definition", make: func() (terraformutils.Resource, bool) {
			return newSageMakerFlowDefinitionResource(sagemakertypes.FlowDefinitionSummary{FlowDefinitionName: aws.String("human-loop"), FlowDefinitionStatus: sagemakertypes.FlowDefinitionStatusActive})
		}},
		{name: "device fleet", make: func() (terraformutils.Resource, bool) {
			return newSageMakerDeviceFleetResource(sagemakertypes.DeviceFleetSummary{DeviceFleetName: aws.String("edge-devices")})
		}},
	}
	resources := make([]terraformutils.Resource, 0, len(constructors))
	for _, constructor := range constructors {
		resource, ok := constructor.make()
		if !ok {
			t.Fatalf("%s resource should be created", constructor.name)
		}
		resources = append(resources, resource)
	}
	return resources
}

func sageMakerTestStringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type fakeSageMakerListAlgorithmsClient struct {
	pages []*sagemaker.ListAlgorithmsOutput
	calls int
}

func (f *fakeSageMakerListAlgorithmsClient) ListAlgorithms(context.Context, *sagemaker.ListAlgorithmsInput, ...func(*sagemaker.Options)) (*sagemaker.ListAlgorithmsOutput, error) {
	if f.calls >= len(f.pages) {
		return &sagemaker.ListAlgorithmsOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	return page, nil
}
