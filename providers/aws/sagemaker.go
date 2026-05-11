// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	sageMakerAppResourceType                      = "aws_sagemaker_app"
	sageMakerAppImageConfigResourceType           = "aws_sagemaker_app_image_config"
	sageMakerCodeRepositoryResourceType           = "aws_sagemaker_code_repository"
	sageMakerDataQualityJobDefinitionResourceType = "aws_sagemaker_data_quality_job_definition"
	sageMakerDeviceFleetResourceType              = "aws_sagemaker_device_fleet"
	sageMakerDomainResourceType                   = "aws_sagemaker_domain"
	sageMakerEndpointConfigurationResourceType    = "aws_sagemaker_endpoint_configuration"
	sageMakerEndpointResourceType                 = "aws_sagemaker_endpoint"
	sageMakerFeatureGroupResourceType             = "aws_sagemaker_feature_group"
	sageMakerFlowDefinitionResourceType           = "aws_sagemaker_flow_definition"
	sageMakerImageResourceType                    = "aws_sagemaker_image"
	sageMakerImageVersionResourceType             = "aws_sagemaker_image_version"
	sageMakerModelPackageGroupResourceType        = "aws_sagemaker_model_package_group"
	sageMakerModelPackageGroupPolicyResourceType  = "aws_sagemaker_model_package_group_policy"
	sageMakerModelResourceType                    = "aws_sagemaker_model"
	sageMakerMonitoringScheduleResourceType       = "aws_sagemaker_monitoring_schedule"
	sageMakerPipelineResourceType                 = "aws_sagemaker_pipeline"
	sageMakerProjectResourceType                  = "aws_sagemaker_project"
	sageMakerServicecatalogPortfolioStatusType    = "aws_sagemaker_servicecatalog_portfolio_status"
	sageMakerSpaceResourceType                    = "aws_sagemaker_space"
	sageMakerStudioLifecycleConfigResourceType    = "aws_sagemaker_studio_lifecycle_config"
	sageMakerUserProfileResourceType              = "aws_sagemaker_user_profile"
	sageMakerWorkforceResourceType                = "aws_sagemaker_workforce"
	sageMakerWorkteamResourceType                 = "aws_sagemaker_workteam"
	sageMakerImportIDSeparator                    = ","
	sageMakerResourceNameFallback                 = "sagemaker-resource"
)

var (
	sageMakerAllowEmptyValues = []string{"tags."}
	sageMakerResourceTypes    = []string{
		sageMakerServiceName(sageMakerAppResourceType),
		sageMakerServiceName(sageMakerAppImageConfigResourceType),
		sageMakerServiceName(sageMakerCodeRepositoryResourceType),
		sageMakerServiceName(sageMakerDataQualityJobDefinitionResourceType),
		sageMakerServiceName(sageMakerDeviceFleetResourceType),
		sageMakerServiceName(sageMakerDomainResourceType),
		sageMakerServiceName(sageMakerEndpointConfigurationResourceType),
		sageMakerServiceName(sageMakerEndpointResourceType),
		sageMakerServiceName(sageMakerFeatureGroupResourceType),
		sageMakerServiceName(sageMakerFlowDefinitionResourceType),
		sageMakerServiceName(sageMakerImageResourceType),
		sageMakerServiceName(sageMakerImageVersionResourceType),
		sageMakerServiceName(sageMakerModelPackageGroupResourceType),
		sageMakerServiceName(sageMakerModelPackageGroupPolicyResourceType),
		sageMakerServiceName(sageMakerModelResourceType),
		sageMakerServiceName(sageMakerMonitoringScheduleResourceType),
		sageMakerServiceName(sageMakerPipelineResourceType),
		sageMakerServiceName(sageMakerProjectResourceType),
		sageMakerServiceName(sageMakerServicecatalogPortfolioStatusType),
		sageMakerServiceName(sageMakerSpaceResourceType),
		sageMakerServiceName(sageMakerStudioLifecycleConfigResourceType),
		sageMakerServiceName(sageMakerUserProfileResourceType),
		sageMakerServiceName(sageMakerWorkforceResourceType),
		sageMakerServiceName(sageMakerWorkteamResourceType),
	}
)

type SageMakerGenerator struct {
	AWSService
}

func (g *SageMakerGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := sageMakerServiceName(resource.InstanceInfo.Type)
		if g.hasTypedSageMakerFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *SageMakerGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := sagemaker.NewFromConfig(config)

	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerModelResourceType)) {
		if err := g.loadModels(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerEndpointConfigurationResourceType)) {
		if err := g.loadEndpointConfigurations(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerEndpointResourceType)) {
		if err := g.loadEndpoints(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerDomainResourceType)) {
		if err := g.loadDomains(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerUserProfileResourceType)) {
		if err := g.loadUserProfiles(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerSpaceResourceType)) {
		if err := g.loadSpaces(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerAppResourceType)) {
		if err := g.loadApps(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerAppImageConfigResourceType)) {
		if err := g.loadAppImageConfigs(svc); err != nil {
			return err
		}
	}

	loadImages := g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerImageResourceType))
	loadImageVersions := g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerImageVersionResourceType))
	if loadImages || loadImageVersions {
		images, err := listSageMakerImages(svc)
		if err != nil {
			return err
		}
		if loadImages {
			g.loadImages(images)
		}
		if loadImageVersions {
			if err := g.loadImageVersions(svc, images); err != nil {
				return err
			}
		}
	}

	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerStudioLifecycleConfigResourceType)) {
		if err := g.loadStudioLifecycleConfigs(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerCodeRepositoryResourceType)) {
		if err := g.loadCodeRepositories(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerFeatureGroupResourceType)) {
		if err := g.loadFeatureGroups(svc); err != nil {
			return err
		}
	}

	loadModelPackageGroups := g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerModelPackageGroupResourceType))
	loadModelPackageGroupPolicies := g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerModelPackageGroupPolicyResourceType))
	if loadModelPackageGroups || loadModelPackageGroupPolicies {
		groups, err := listSageMakerModelPackageGroups(svc)
		if err != nil {
			return err
		}
		if loadModelPackageGroups {
			g.loadModelPackageGroups(groups)
		}
		if loadModelPackageGroupPolicies {
			if err := g.loadModelPackageGroupPolicies(svc, groups); err != nil {
				return err
			}
		}
	}

	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerPipelineResourceType)) {
		if err := g.loadPipelines(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerProjectResourceType)) {
		if err := g.loadProjects(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerServicecatalogPortfolioStatusType)) {
		if err := g.loadServicecatalogPortfolioStatus(svc, config.Region); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerDataQualityJobDefinitionResourceType)) {
		if err := g.loadDataQualityJobDefinitions(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerMonitoringScheduleResourceType)) {
		if err := g.loadMonitoringSchedules(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerWorkforceResourceType)) {
		if err := g.loadWorkforces(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerWorkteamResourceType)) {
		if err := g.loadWorkteams(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerFlowDefinitionResourceType)) {
		if err := g.loadFlowDefinitions(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadSageMakerResource(sageMakerServiceName(sageMakerDeviceFleetResourceType)) {
		if err := g.loadDeviceFleets(svc); err != nil {
			return err
		}
	}

	return nil
}

func (g *SageMakerGenerator) shouldLoadSageMakerResource(serviceName string) bool {
	if !g.hasTypedSageMakerFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *SageMakerGenerator) hasTypedSageMakerFilter() bool {
	for _, serviceName := range sageMakerResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *SageMakerGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *SageMakerGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func sageMakerServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func (g *SageMakerGenerator) loadModels(svc *sagemaker.Client) error {
	p := sagemaker.NewListModelsPaginator(svc, &sagemaker.ListModelsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, model := range page.Models {
			if resource, ok := newSageMakerModelResource(model); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadEndpointConfigurations(svc *sagemaker.Client) error {
	p := sagemaker.NewListEndpointConfigsPaginator(svc, &sagemaker.ListEndpointConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.EndpointConfigs {
			if resource, ok := newSageMakerEndpointConfigurationResource(config); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadEndpoints(svc *sagemaker.Client) error {
	p := sagemaker.NewListEndpointsPaginator(svc, &sagemaker.ListEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.Endpoints {
			if resource, ok := newSageMakerEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadDomains(svc *sagemaker.Client) error {
	p := sagemaker.NewListDomainsPaginator(svc, &sagemaker.ListDomainsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, domain := range page.Domains {
			if resource, ok := newSageMakerDomainResource(domain); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadUserProfiles(svc *sagemaker.Client) error {
	p := sagemaker.NewListUserProfilesPaginator(svc, &sagemaker.ListUserProfilesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, userProfile := range page.UserProfiles {
			domainID := StringValue(userProfile.DomainId)
			userProfileName := StringValue(userProfile.UserProfileName)
			if domainID == "" || userProfileName == "" || !sageMakerUserProfileImportable(userProfile.Status) {
				continue
			}
			userProfileOutput, err := getSageMakerUserProfile(svc, domainID, userProfileName)
			if err != nil {
				if sageMakerResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSageMakerUserProfileResource(userProfileOutput); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getSageMakerUserProfile(svc *sagemaker.Client, domainID, userProfileName string) (*sagemaker.DescribeUserProfileOutput, error) {
	return svc.DescribeUserProfile(context.TODO(), &sagemaker.DescribeUserProfileInput{
		DomainId:        &domainID,
		UserProfileName: &userProfileName,
	})
}

func (g *SageMakerGenerator) loadSpaces(svc *sagemaker.Client) error {
	p := sagemaker.NewListSpacesPaginator(svc, &sagemaker.ListSpacesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, space := range page.Spaces {
			if StringValue(space.DomainId) == "" || StringValue(space.SpaceName) == "" || !sageMakerSpaceImportable(space.Status) {
				continue
			}
			spaceOutput, err := getSageMakerSpace(svc, StringValue(space.DomainId), StringValue(space.SpaceName))
			if err != nil {
				if sageMakerResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSageMakerSpaceResource(spaceOutput); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getSageMakerSpace(svc *sagemaker.Client, domainID, spaceName string) (*sagemaker.DescribeSpaceOutput, error) {
	return svc.DescribeSpace(context.TODO(), &sagemaker.DescribeSpaceInput{
		DomainId:  &domainID,
		SpaceName: &spaceName,
	})
}

func (g *SageMakerGenerator) loadApps(svc *sagemaker.Client) error {
	p := sagemaker.NewListAppsPaginator(svc, &sagemaker.ListAppsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, app := range page.Apps {
			if StringValue(app.AppName) == "" ||
				StringValue(app.DomainId) == "" ||
				app.AppType == "" ||
				(StringValue(app.SpaceName) == "" && StringValue(app.UserProfileName) == "") ||
				!sageMakerAppImportable(app.Status) {
				continue
			}
			appOutput, err := getSageMakerApp(svc, app)
			if err != nil {
				if sageMakerResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSageMakerAppResource(appOutput); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getSageMakerApp(svc *sagemaker.Client, app sagemakertypes.AppDetails) (*sagemaker.DescribeAppOutput, error) {
	appName := StringValue(app.AppName)
	domainID := StringValue(app.DomainId)
	input := &sagemaker.DescribeAppInput{
		AppName:  &appName,
		AppType:  app.AppType,
		DomainId: &domainID,
	}
	if spaceName := StringValue(app.SpaceName); spaceName != "" {
		input.SpaceName = &spaceName
	} else if userProfileName := StringValue(app.UserProfileName); userProfileName != "" {
		input.UserProfileName = &userProfileName
	}
	return svc.DescribeApp(context.TODO(), input)
}

func (g *SageMakerGenerator) loadAppImageConfigs(svc *sagemaker.Client) error {
	p := sagemaker.NewListAppImageConfigsPaginator(svc, &sagemaker.ListAppImageConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.AppImageConfigs {
			if resource, ok := newSageMakerAppImageConfigResource(config); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listSageMakerImages(svc *sagemaker.Client) ([]sagemakertypes.Image, error) {
	p := sagemaker.NewListImagesPaginator(svc, &sagemaker.ListImagesInput{})
	images := []sagemakertypes.Image{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		images = append(images, page.Images...)
	}
	return images, nil
}

func (g *SageMakerGenerator) loadImages(images []sagemakertypes.Image) {
	for _, image := range images {
		if resource, ok := newSageMakerImageResource(image); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *SageMakerGenerator) loadImageVersions(svc *sagemaker.Client, images []sagemakertypes.Image) error {
	for _, image := range images {
		imageName := StringValue(image.ImageName)
		if imageName == "" || !sageMakerImageImportable(image.ImageStatus) {
			continue
		}
		p := sagemaker.NewListImageVersionsPaginator(svc, &sagemaker.ListImageVersionsInput{ImageName: &imageName})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, version := range page.ImageVersions {
				if resource, ok := newSageMakerImageVersionResource(imageName, version); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadStudioLifecycleConfigs(svc *sagemaker.Client) error {
	p := sagemaker.NewListStudioLifecycleConfigsPaginator(svc, &sagemaker.ListStudioLifecycleConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.StudioLifecycleConfigs {
			if resource, ok := newSageMakerStudioLifecycleConfigResource(config); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadCodeRepositories(svc *sagemaker.Client) error {
	p := sagemaker.NewListCodeRepositoriesPaginator(svc, &sagemaker.ListCodeRepositoriesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, repo := range page.CodeRepositorySummaryList {
			if resource, ok := newSageMakerCodeRepositoryResource(repo); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadFeatureGroups(svc *sagemaker.Client) error {
	p := sagemaker.NewListFeatureGroupsPaginator(svc, &sagemaker.ListFeatureGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.FeatureGroupSummaries {
			if resource, ok := newSageMakerFeatureGroupResource(group); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listSageMakerModelPackageGroups(svc *sagemaker.Client) ([]sagemakertypes.ModelPackageGroupSummary, error) {
	p := sagemaker.NewListModelPackageGroupsPaginator(svc, &sagemaker.ListModelPackageGroupsInput{})
	groups := []sagemakertypes.ModelPackageGroupSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		groups = append(groups, page.ModelPackageGroupSummaryList...)
	}
	return groups, nil
}

func (g *SageMakerGenerator) loadModelPackageGroups(groups []sagemakertypes.ModelPackageGroupSummary) {
	for _, group := range groups {
		if resource, ok := newSageMakerModelPackageGroupResource(group); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *SageMakerGenerator) loadModelPackageGroupPolicies(svc *sagemaker.Client, groups []sagemakertypes.ModelPackageGroupSummary) error {
	for _, group := range groups {
		groupName := StringValue(group.ModelPackageGroupName)
		if groupName == "" || !sageMakerModelPackageGroupImportable(group.ModelPackageGroupStatus) {
			continue
		}
		output, err := svc.GetModelPackageGroupPolicy(context.TODO(), &sagemaker.GetModelPackageGroupPolicyInput{
			ModelPackageGroupName: &groupName,
		})
		if err != nil {
			if sageMakerResourceNotFound(err) {
				continue
			}
			return err
		}
		if resource, ok := newSageMakerModelPackageGroupPolicyResource(groupName, output.ResourcePolicy); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadPipelines(svc *sagemaker.Client) error {
	p := sagemaker.NewListPipelinesPaginator(svc, &sagemaker.ListPipelinesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, pipeline := range page.PipelineSummaries {
			pipelineName := StringValue(pipeline.PipelineName)
			if pipelineName == "" {
				continue
			}
			pipelineOutput, err := svc.DescribePipeline(context.TODO(), &sagemaker.DescribePipelineInput{
				PipelineName: &pipelineName,
			})
			if err != nil {
				if sageMakerResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSageMakerPipelineResource(pipelineOutput); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadProjects(svc *sagemaker.Client) error {
	p := sagemaker.NewListProjectsPaginator(svc, &sagemaker.ListProjectsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, project := range page.ProjectSummaryList {
			if resource, ok := newSageMakerProjectResource(project); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadServicecatalogPortfolioStatus(svc *sagemaker.Client, region string) error {
	output, err := svc.GetSagemakerServicecatalogPortfolioStatus(context.TODO(), &sagemaker.GetSagemakerServicecatalogPortfolioStatusInput{})
	if err != nil {
		return err
	}
	if resource, ok := newSageMakerServicecatalogPortfolioStatusResource(region, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SageMakerGenerator) loadDataQualityJobDefinitions(svc *sagemaker.Client) error {
	p := sagemaker.NewListDataQualityJobDefinitionsPaginator(svc, &sagemaker.ListDataQualityJobDefinitionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, definition := range page.JobDefinitionSummaries {
			if resource, ok := newSageMakerDataQualityJobDefinitionResource(definition); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadMonitoringSchedules(svc *sagemaker.Client) error {
	p := sagemaker.NewListMonitoringSchedulesPaginator(svc, &sagemaker.ListMonitoringSchedulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, schedule := range page.MonitoringScheduleSummaries {
			if resource, ok := newSageMakerMonitoringScheduleResource(schedule); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadWorkforces(svc *sagemaker.Client) error {
	p := sagemaker.NewListWorkforcesPaginator(svc, &sagemaker.ListWorkforcesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, workforce := range page.Workforces {
			if resource, ok := newSageMakerWorkforceResource(workforce); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadWorkteams(svc *sagemaker.Client) error {
	p := sagemaker.NewListWorkteamsPaginator(svc, &sagemaker.ListWorkteamsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, workteam := range page.Workteams {
			if resource, ok := newSageMakerWorkteamResource(workteam); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadFlowDefinitions(svc *sagemaker.Client) error {
	p := sagemaker.NewListFlowDefinitionsPaginator(svc, &sagemaker.ListFlowDefinitionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, definition := range page.FlowDefinitionSummaries {
			if resource, ok := newSageMakerFlowDefinitionResource(definition); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SageMakerGenerator) loadDeviceFleets(svc *sagemaker.Client) error {
	p := sagemaker.NewListDeviceFleetsPaginator(svc, &sagemaker.ListDeviceFleetsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, fleet := range page.DeviceFleetSummaries {
			if resource, ok := newSageMakerDeviceFleetResource(fleet); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newSageMakerModelResource(model sagemakertypes.ModelSummary) (terraformutils.Resource, bool) {
	name := StringValue(model.ModelName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("model", name), sageMakerModelResourceType, map[string]string{"name": name})
}

func newSageMakerEndpointConfigurationResource(config sagemakertypes.EndpointConfigSummary) (terraformutils.Resource, bool) {
	name := StringValue(config.EndpointConfigName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("endpoint-configuration", name), sageMakerEndpointConfigurationResourceType, map[string]string{"name": name})
}

func newSageMakerEndpointResource(endpoint sagemakertypes.EndpointSummary) (terraformutils.Resource, bool) {
	name := StringValue(endpoint.EndpointName)
	if name == "" || !sageMakerEndpointImportable(endpoint.EndpointStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("endpoint", name), sageMakerEndpointResourceType, map[string]string{"name": name})
}

func newSageMakerDomainResource(domain sagemakertypes.DomainDetails) (terraformutils.Resource, bool) {
	domainID := StringValue(domain.DomainId)
	domainName := StringValue(domain.DomainName)
	if domainID == "" || domainName == "" || !sageMakerDomainImportable(domain.Status) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(domainID, sageMakerResourceName("domain", domainName, domainID), sageMakerDomainResourceType, map[string]string{
		"domain_name": domainName,
	})
}

func newSageMakerUserProfileResource(profile *sagemaker.DescribeUserProfileOutput) (terraformutils.Resource, bool) {
	if profile == nil {
		return terraformutils.Resource{}, false
	}
	userProfileArn := StringValue(profile.UserProfileArn)
	domainID := StringValue(profile.DomainId)
	name := StringValue(profile.UserProfileName)
	if userProfileArn == "" || domainID == "" || name == "" || !sageMakerUserProfileImportable(profile.Status) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(sageMakerUserProfileImportID(userProfileArn), sageMakerResourceName("user-profile", domainID, name), sageMakerUserProfileResourceType, map[string]string{
		"domain_id":         domainID,
		"user_profile_name": name,
	})
}

func newSageMakerSpaceResource(space *sagemaker.DescribeSpaceOutput) (terraformutils.Resource, bool) {
	if space == nil {
		return terraformutils.Resource{}, false
	}
	spaceArn := StringValue(space.SpaceArn)
	domainID := StringValue(space.DomainId)
	name := StringValue(space.SpaceName)
	if spaceArn == "" || domainID == "" || name == "" || !sageMakerSpaceImportable(space.Status) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"domain_id":  domainID,
		"space_name": name,
	}
	sageMakerAddStringAttribute(attributes, "space_display_name", space.SpaceDisplayName)
	return sageMakerResource(spaceArn, sageMakerResourceName("space", domainID, name), sageMakerSpaceResourceType, attributes)
}

func newSageMakerAppResource(app *sagemaker.DescribeAppOutput) (terraformutils.Resource, bool) {
	if app == nil {
		return terraformutils.Resource{}, false
	}
	appArn := StringValue(app.AppArn)
	appName := StringValue(app.AppName)
	domainID := StringValue(app.DomainId)
	if appArn == "" || appName == "" || domainID == "" || app.AppType == "" || !sageMakerAppImportable(app.Status) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"app_name":  appName,
		"app_type":  string(app.AppType),
		"domain_id": domainID,
	}
	ownerKind := ""
	ownerName := ""
	if spaceName := StringValue(app.SpaceName); spaceName != "" {
		attributes["space_name"] = spaceName
		ownerKind = "space"
		ownerName = spaceName
	} else if userProfileName := StringValue(app.UserProfileName); userProfileName != "" {
		attributes["user_profile_name"] = userProfileName
		ownerKind = "user-profile"
		ownerName = userProfileName
	} else {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(appArn, sageMakerResourceName("app", domainID, ownerKind, ownerName, string(app.AppType), appName), sageMakerAppResourceType, attributes)
}

func newSageMakerAppImageConfigResource(config sagemakertypes.AppImageConfigDetails) (terraformutils.Resource, bool) {
	name := StringValue(config.AppImageConfigName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("app-image-config", name), sageMakerAppImageConfigResourceType, map[string]string{
		"app_image_config_name": name,
	})
}

func newSageMakerImageResource(image sagemakertypes.Image) (terraformutils.Resource, bool) {
	name := StringValue(image.ImageName)
	if name == "" || !sageMakerImageImportable(image.ImageStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("image", name), sageMakerImageResourceType, map[string]string{
		"image_name": name,
	})
}

func newSageMakerImageVersionResource(imageName string, version sagemakertypes.ImageVersion) (terraformutils.Resource, bool) {
	versionNumber := sageMakerInt32Value(version.Version)
	if imageName == "" || versionNumber == 0 || !sageMakerImageVersionImportable(version.ImageVersionStatus) {
		return terraformutils.Resource{}, false
	}
	versionString := strconv.Itoa(int(versionNumber))
	return sageMakerResource(sageMakerImageVersionImportID(imageName, versionNumber), sageMakerResourceName("image-version", imageName, versionString), sageMakerImageVersionResourceType, map[string]string{
		"image_name": imageName,
		"version":    versionString,
	})
}

func newSageMakerStudioLifecycleConfigResource(config sagemakertypes.StudioLifecycleConfigDetails) (terraformutils.Resource, bool) {
	name := StringValue(config.StudioLifecycleConfigName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"studio_lifecycle_config_name": name}
	if config.StudioLifecycleConfigAppType != "" {
		attributes["studio_lifecycle_config_app_type"] = string(config.StudioLifecycleConfigAppType)
	}
	return sageMakerResource(name, sageMakerResourceName("studio-lifecycle-config", name), sageMakerStudioLifecycleConfigResourceType, attributes)
}

func newSageMakerCodeRepositoryResource(repo sagemakertypes.CodeRepositorySummary) (terraformutils.Resource, bool) {
	name := StringValue(repo.CodeRepositoryName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("code-repository", name), sageMakerCodeRepositoryResourceType, map[string]string{
		"code_repository_name": name,
	})
}

func newSageMakerFeatureGroupResource(group sagemakertypes.FeatureGroupSummary) (terraformutils.Resource, bool) {
	name := StringValue(group.FeatureGroupName)
	if name == "" || !sageMakerFeatureGroupImportable(group.FeatureGroupStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("feature-group", name), sageMakerFeatureGroupResourceType, map[string]string{
		"feature_group_name": name,
	})
}

func newSageMakerModelPackageGroupResource(group sagemakertypes.ModelPackageGroupSummary) (terraformutils.Resource, bool) {
	name := StringValue(group.ModelPackageGroupName)
	if name == "" || !sageMakerModelPackageGroupImportable(group.ModelPackageGroupStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("model-package-group", name), sageMakerModelPackageGroupResourceType, map[string]string{
		"model_package_group_name": name,
	})
}

func newSageMakerModelPackageGroupPolicyResource(groupName string, policy *string) (terraformutils.Resource, bool) {
	policyText := StringValue(policy)
	if groupName == "" || policyText == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(groupName, sageMakerResourceName("model-package-group-policy", groupName), sageMakerModelPackageGroupPolicyResourceType, map[string]string{
		"model_package_group_name": groupName,
		"resource_policy":          policyText,
	})
}

func newSageMakerPipelineResource(pipeline *sagemaker.DescribePipelineOutput) (terraformutils.Resource, bool) {
	if pipeline == nil {
		return terraformutils.Resource{}, false
	}
	name := StringValue(pipeline.PipelineName)
	if name == "" || !sageMakerPipelineImportable(pipeline.PipelineStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("pipeline", name), sageMakerPipelineResourceType, map[string]string{
		"pipeline_name": name,
	})
}

func newSageMakerProjectResource(project sagemakertypes.ProjectSummary) (terraformutils.Resource, bool) {
	name := StringValue(project.ProjectName)
	if name == "" || !sageMakerProjectImportable(project.ProjectStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("project", name), sageMakerProjectResourceType, map[string]string{
		"project_name": name,
	})
}

func newSageMakerServicecatalogPortfolioStatusResource(region string, status *sagemaker.GetSagemakerServicecatalogPortfolioStatusOutput) (terraformutils.Resource, bool) {
	if status == nil || region == "" || status.Status == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(sageMakerServicecatalogPortfolioStatusImportID(region), sageMakerResourceName("servicecatalog-portfolio-status", region), sageMakerServicecatalogPortfolioStatusType, map[string]string{
		"status": string(status.Status),
	})
}

func newSageMakerDataQualityJobDefinitionResource(definition sagemakertypes.MonitoringJobDefinitionSummary) (terraformutils.Resource, bool) {
	name := StringValue(definition.MonitoringJobDefinitionName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("data-quality-job-definition", name), sageMakerDataQualityJobDefinitionResourceType, map[string]string{
		"name": name,
	})
}

func newSageMakerMonitoringScheduleResource(schedule sagemakertypes.MonitoringScheduleSummary) (terraformutils.Resource, bool) {
	name := StringValue(schedule.MonitoringScheduleName)
	if name == "" || !sageMakerMonitoringScheduleImportable(schedule.MonitoringScheduleStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("monitoring-schedule", name), sageMakerMonitoringScheduleResourceType, map[string]string{
		"name": name,
	})
}

func newSageMakerWorkforceResource(workforce sagemakertypes.Workforce) (terraformutils.Resource, bool) {
	name := StringValue(workforce.WorkforceName)
	if name == "" || workforce.OidcConfig != nil || !sageMakerWorkforceImportable(workforce.Status) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("workforce", name), sageMakerWorkforceResourceType, map[string]string{
		"workforce_name": name,
	})
}

func newSageMakerWorkteamResource(workteam sagemakertypes.Workteam) (terraformutils.Resource, bool) {
	name := StringValue(workteam.WorkteamName)
	if name == "" || StringValue(workteam.Description) == "" || len(workteam.MemberDefinitions) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"description":   StringValue(workteam.Description),
		"workteam_name": name,
	}
	if workforceName := sageMakerWorkforceNameFromARN(StringValue(workteam.WorkforceArn)); workforceName != "" {
		attributes["workforce_name"] = workforceName
	}
	return sageMakerResource(name, sageMakerResourceName("workteam", name), sageMakerWorkteamResourceType, attributes)
}

func newSageMakerFlowDefinitionResource(definition sagemakertypes.FlowDefinitionSummary) (terraformutils.Resource, bool) {
	name := StringValue(definition.FlowDefinitionName)
	if name == "" || !sageMakerFlowDefinitionImportable(definition.FlowDefinitionStatus) {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("flow-definition", name), sageMakerFlowDefinitionResourceType, map[string]string{
		"flow_definition_name": name,
	})
}

func newSageMakerDeviceFleetResource(fleet sagemakertypes.DeviceFleetSummary) (terraformutils.Resource, bool) {
	name := StringValue(fleet.DeviceFleetName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return sageMakerResource(name, sageMakerResourceName("device-fleet", name), sageMakerDeviceFleetResourceType, map[string]string{
		"device_fleet_name": name,
	})
}

func sageMakerResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" || resourceType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		sageMakerAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func sageMakerUserProfileImportID(userProfileArn string) string {
	return userProfileArn
}

func sageMakerServicecatalogPortfolioStatusImportID(region string) string {
	return region
}

func sageMakerImageVersionImportID(imageName string, version int32) string {
	return strings.Join([]string{imageName, strconv.Itoa(int(version))}, sageMakerImportIDSeparator)
}

func sageMakerResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return sageMakerResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func sageMakerEndpointImportable(status sagemakertypes.EndpointStatus) bool {
	return status == sagemakertypes.EndpointStatusInService
}

func sageMakerDomainImportable(status sagemakertypes.DomainStatus) bool {
	return status == sagemakertypes.DomainStatusInService
}

func sageMakerUserProfileImportable(status sagemakertypes.UserProfileStatus) bool {
	return status == sagemakertypes.UserProfileStatusInService
}

func sageMakerSpaceImportable(status sagemakertypes.SpaceStatus) bool {
	return status == sagemakertypes.SpaceStatusInService
}

func sageMakerAppImportable(status sagemakertypes.AppStatus) bool {
	return status == sagemakertypes.AppStatusInService
}

func sageMakerImageImportable(status sagemakertypes.ImageStatus) bool {
	return status == sagemakertypes.ImageStatusCreated
}

func sageMakerImageVersionImportable(status sagemakertypes.ImageVersionStatus) bool {
	return status == sagemakertypes.ImageVersionStatusCreated
}

func sageMakerFeatureGroupImportable(status sagemakertypes.FeatureGroupStatus) bool {
	return status == sagemakertypes.FeatureGroupStatusCreated
}

func sageMakerModelPackageGroupImportable(status sagemakertypes.ModelPackageGroupStatus) bool {
	return status == sagemakertypes.ModelPackageGroupStatusCompleted
}

func sageMakerPipelineImportable(status sagemakertypes.PipelineStatus) bool {
	return status == sagemakertypes.PipelineStatusActive
}

func sageMakerProjectImportable(status sagemakertypes.ProjectStatus) bool {
	return status == sagemakertypes.ProjectStatusCreateCompleted ||
		status == sagemakertypes.ProjectStatusUpdateCompleted
}

func sageMakerMonitoringScheduleImportable(status sagemakertypes.ScheduleStatus) bool {
	return status == sagemakertypes.ScheduleStatusScheduled ||
		status == sagemakertypes.ScheduleStatusStopped
}

func sageMakerFlowDefinitionImportable(status sagemakertypes.FlowDefinitionStatus) bool {
	return status == sagemakertypes.FlowDefinitionStatusActive
}

func sageMakerWorkforceImportable(status sagemakertypes.WorkforceStatus) bool {
	return status == sagemakertypes.WorkforceStatusActive
}

func sageMakerAddStringAttribute(attributes map[string]string, name string, value *string) {
	if StringValue(value) != "" {
		attributes[name] = StringValue(value)
	}
}

func sageMakerInt32Value(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func sageMakerWorkforceNameFromARN(workforceARN string) string {
	const workforceResourcePrefix = "workforce/"
	_, resource, ok := strings.Cut(workforceARN, workforceResourcePrefix)
	if !ok {
		return ""
	}
	return resource
}

func sageMakerResourceNotFound(err error) bool {
	var notFound *sagemakertypes.ResourceNotFound
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	errorCode := strings.ToLower(apiErr.ErrorCode())
	errorMessage := strings.ToLower(apiErr.ErrorMessage())
	return strings.Contains(errorCode, "notfound") ||
		strings.Contains(errorCode, "not_found") ||
		strings.Contains(errorMessage, "cannot find") ||
		strings.Contains(errorMessage, "not found") ||
		strings.Contains(errorMessage, "does not exist")
}
