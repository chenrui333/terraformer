// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigtypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appConfigApplicationResourceType                = "appconfig_application"
	appConfigConfigurationProfileResourceType       = "appconfig_configuration_profile"
	appConfigDeploymentResourceType                 = "appconfig_deployment"
	appConfigDeploymentStrategyResourceType         = "appconfig_deployment_strategy"
	appConfigEnvironmentResourceType                = "appconfig_environment"
	appConfigExtensionResourceType                  = "appconfig_extension"
	appConfigExtensionAssociationResourceType       = "appconfig_extension_association"
	appConfigHostedConfigurationVersionResourceType = "appconfig_hosted_configuration_version"

	appConfigConfigurationProfileIDSeparator       = ":"
	appConfigDeploymentIDSeparator                 = "/"
	appConfigEnvironmentIDSeparator                = ":"
	appConfigHostedConfigurationVersionIDSeparator = "/"
	appConfigHostedLocationURI                     = "hosted"
)

var (
	appConfigAllowEmptyValues = []string{"tags."}

	appConfigTopLevelResourceTypes = []string{
		appConfigApplicationResourceType,
		appConfigDeploymentStrategyResourceType,
		appConfigExtensionResourceType,
		appConfigExtensionAssociationResourceType,
	}
	appConfigApplicationChildResourceTypes = []string{
		appConfigConfigurationProfileResourceType,
		appConfigDeploymentResourceType,
		appConfigEnvironmentResourceType,
		appConfigHostedConfigurationVersionResourceType,
	}
	appConfigResourceTypes = append(appConfigTopLevelResourceTypes, appConfigApplicationChildResourceTypes...)
)

type AppConfigGenerator struct {
	AWSService
}

func (g *AppConfigGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := appconfig.NewFromConfig(config)

	if g.shouldLoadApplications() {
		if err := g.loadApplications(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTopLevelResourceType(appConfigDeploymentStrategyResourceType) {
		if err := g.loadDeploymentStrategies(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTopLevelResourceType(appConfigExtensionResourceType) {
		if err := g.loadExtensions(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTopLevelResourceType(appConfigExtensionAssociationResourceType) {
		if err := g.loadExtensionAssociations(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadApplications(svc *appconfig.Client) error {
	p := appconfig.NewListApplicationsPaginator(svc, &appconfig.ListApplicationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, application := range page.Items {
			applicationID := StringValue(application.Id)
			applicationName := StringValue(application.Name)
			if applicationID == "" || applicationName == "" {
				continue
			}
			applicationResource := newAppConfigApplicationResource(applicationID, applicationName)
			if g.shouldAppendTopLevelResource(appConfigApplicationResourceType, applicationResource) {
				g.Resources = append(g.Resources, applicationResource)
			}
			if !g.shouldLoadApplicationChildren(applicationResource) {
				continue
			}
			if g.shouldLoadConfigurationProfiles(applicationID) {
				if err := g.loadConfigurationProfiles(svc, applicationID); err != nil {
					if appConfigResourceMissing(err) {
						continue
					}
					return err
				}
			}
			if g.shouldLoadEnvironments(applicationID) {
				if err := g.loadEnvironments(svc, applicationID); err != nil {
					if appConfigResourceMissing(err) {
						continue
					}
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadConfigurationProfiles(svc *appconfig.Client, applicationID string) error {
	p := appconfig.NewListConfigurationProfilesPaginator(svc, &appconfig.ListConfigurationProfilesInput{
		ApplicationId: aws.String(applicationID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, profile := range page.Items {
			profileID := StringValue(profile.Id)
			profileName := StringValue(profile.Name)
			if profileID == "" || profileName == "" {
				continue
			}
			profileResource := newAppConfigConfigurationProfileResource(applicationID, profileID, profileName)
			if g.shouldAppendApplicationChildResource(appConfigConfigurationProfileResourceType, profileResource) {
				g.Resources = append(g.Resources, profileResource)
			}
			if g.shouldLoadHostedConfigurationVersions(applicationID, profileID, StringValue(profile.LocationUri)) {
				if err := g.loadHostedConfigurationVersions(svc, applicationID, profileID); err != nil {
					if appConfigResourceMissing(err) {
						continue
					}
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadHostedConfigurationVersions(svc *appconfig.Client, applicationID, configurationProfileID string) error {
	p := appconfig.NewListHostedConfigurationVersionsPaginator(svc, &appconfig.ListHostedConfigurationVersionsInput{
		ApplicationId:          aws.String(applicationID),
		ConfigurationProfileId: aws.String(configurationProfileID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, version := range page.Items {
			versionNumber := version.VersionNumber
			if versionNumber == 0 {
				continue
			}
			resource := newAppConfigHostedConfigurationVersionResource(applicationID, configurationProfileID, versionNumber)
			if g.shouldAppendApplicationChildResource(appConfigHostedConfigurationVersionResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadEnvironments(svc *appconfig.Client, applicationID string) error {
	p := appconfig.NewListEnvironmentsPaginator(svc, &appconfig.ListEnvironmentsInput{
		ApplicationId: aws.String(applicationID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, environment := range page.Items {
			environmentID := StringValue(environment.Id)
			environmentName := StringValue(environment.Name)
			if environmentID == "" || environmentName == "" {
				continue
			}
			environmentResource := newAppConfigEnvironmentResource(applicationID, environmentID, environmentName)
			if g.shouldAppendApplicationChildResource(appConfigEnvironmentResourceType, environmentResource) {
				g.Resources = append(g.Resources, environmentResource)
			}
			if g.shouldLoadDeployments(applicationID, environmentID) {
				if err := g.loadDeployments(svc, applicationID, environmentID); err != nil {
					if appConfigResourceMissing(err) {
						continue
					}
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadDeployments(svc *appconfig.Client, applicationID, environmentID string) error {
	p := appconfig.NewListDeploymentsPaginator(svc, &appconfig.ListDeploymentsInput{
		ApplicationId: aws.String(applicationID),
		EnvironmentId: aws.String(environmentID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, deployment := range page.Items {
			deploymentNumber := deployment.DeploymentNumber
			if deploymentNumber == 0 {
				continue
			}
			resource := newAppConfigDeploymentResource(applicationID, environmentID, deploymentNumber)
			if g.shouldAppendApplicationChildResource(appConfigDeploymentResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadDeploymentStrategies(svc *appconfig.Client) error {
	p := appconfig.NewListDeploymentStrategiesPaginator(svc, &appconfig.ListDeploymentStrategiesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, strategy := range page.Items {
			strategyID := StringValue(strategy.Id)
			strategyName := StringValue(strategy.Name)
			if strategyID == "" || strategyName == "" || !appConfigDeploymentStrategyImportable(strategyID) {
				continue
			}
			resource := newAppConfigDeploymentStrategyResource(strategyID, strategyName)
			if g.shouldAppendTopLevelResource(appConfigDeploymentStrategyResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadExtensions(svc *appconfig.Client) error {
	p := appconfig.NewListExtensionsPaginator(svc, &appconfig.ListExtensionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, extension := range page.Items {
			extensionID := StringValue(extension.Id)
			extensionName := StringValue(extension.Name)
			if extensionID == "" || extensionName == "" || !appConfigExtensionImportable(extensionID, extensionName) {
				continue
			}
			resource := newAppConfigExtensionResource(extensionID, extensionName)
			if g.shouldAppendTopLevelResource(appConfigExtensionResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppConfigGenerator) loadExtensionAssociations(svc *appconfig.Client) error {
	p := appconfig.NewListExtensionAssociationsPaginator(svc, &appconfig.ListExtensionAssociationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.Items {
			associationID := StringValue(association.Id)
			if associationID == "" {
				continue
			}
			resource := newAppConfigExtensionAssociationResource(associationID)
			if g.shouldAppendTopLevelResource(appConfigExtensionAssociationResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newAppConfigApplicationResource(applicationID, name string) terraformutils.Resource {
	return terraformutils.NewResource(
		applicationID,
		appConfigResourceName(name, applicationID),
		"aws_appconfig_application",
		"aws",
		map[string]string{
			"name": name,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigConfigurationProfileResource(applicationID, configurationProfileID, name string) terraformutils.Resource {
	id := appConfigConfigurationProfileResourceID(configurationProfileID, applicationID)
	return terraformutils.NewResource(
		id,
		appConfigResourceName(applicationID, name, configurationProfileID),
		"aws_appconfig_configuration_profile",
		"aws",
		map[string]string{
			"application_id":           applicationID,
			"configuration_profile_id": configurationProfileID,
			"name":                     name,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigDeploymentResource(applicationID, environmentID string, deploymentNumber int32) terraformutils.Resource {
	id := appConfigDeploymentResourceID(applicationID, environmentID, deploymentNumber)
	return terraformutils.NewResource(
		id,
		appConfigResourceName(applicationID, environmentID, strconv.FormatInt(int64(deploymentNumber), 10)),
		"aws_appconfig_deployment",
		"aws",
		map[string]string{
			"application_id":    applicationID,
			"deployment_number": strconv.FormatInt(int64(deploymentNumber), 10),
			"environment_id":    environmentID,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigDeploymentStrategyResource(strategyID, name string) terraformutils.Resource {
	return terraformutils.NewResource(
		strategyID,
		appConfigResourceName(name, strategyID),
		"aws_appconfig_deployment_strategy",
		"aws",
		map[string]string{
			"name": name,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigEnvironmentResource(applicationID, environmentID, name string) terraformutils.Resource {
	id := appConfigEnvironmentResourceID(environmentID, applicationID)
	return terraformutils.NewResource(
		id,
		appConfigResourceName(applicationID, name, environmentID),
		"aws_appconfig_environment",
		"aws",
		map[string]string{
			"application_id": applicationID,
			"environment_id": environmentID,
			"name":           name,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigExtensionResource(extensionID, name string) terraformutils.Resource {
	return terraformutils.NewResource(
		extensionID,
		appConfigResourceName(name, extensionID),
		"aws_appconfig_extension",
		"aws",
		map[string]string{
			"name": name,
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppConfigExtensionAssociationResource(associationID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		associationID,
		associationID,
		"aws_appconfig_extension_association",
		"aws",
		appConfigAllowEmptyValues,
	)
}

func newAppConfigHostedConfigurationVersionResource(applicationID, configurationProfileID string, versionNumber int32) terraformutils.Resource {
	id := appConfigHostedConfigurationVersionResourceID(applicationID, configurationProfileID, versionNumber)
	return terraformutils.NewResource(
		id,
		appConfigResourceName(applicationID, configurationProfileID, strconv.FormatInt(int64(versionNumber), 10)),
		"aws_appconfig_hosted_configuration_version",
		"aws",
		map[string]string{
			"application_id":           applicationID,
			"configuration_profile_id": configurationProfileID,
			"version_number":           strconv.FormatInt(int64(versionNumber), 10),
		},
		appConfigAllowEmptyValues,
		map[string]interface{}{},
	)
}

func appConfigConfigurationProfileResourceID(configurationProfileID, applicationID string) string {
	return strings.Join([]string{configurationProfileID, applicationID}, appConfigConfigurationProfileIDSeparator)
}

func appConfigDeploymentResourceID(applicationID, environmentID string, deploymentNumber int32) string {
	return strings.Join([]string{applicationID, environmentID, strconv.FormatInt(int64(deploymentNumber), 10)}, appConfigDeploymentIDSeparator)
}

func appConfigEnvironmentResourceID(environmentID, applicationID string) string {
	return strings.Join([]string{environmentID, applicationID}, appConfigEnvironmentIDSeparator)
}

func appConfigHostedConfigurationVersionResourceID(applicationID, configurationProfileID string, versionNumber int32) string {
	return strings.Join([]string{applicationID, configurationProfileID, strconv.FormatInt(int64(versionNumber), 10)}, appConfigHostedConfigurationVersionIDSeparator)
}

func appConfigResourceName(parts ...string) string {
	nonEmptyParts := []string{}
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, ":")
}

func appConfigResourceMissing(err error) bool {
	var notFound *appconfigtypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}

func appConfigDeploymentStrategyImportable(strategyID string) bool {
	return !strings.HasPrefix(strategyID, "AppConfig.")
}

func appConfigExtensionImportable(extensionID, extensionName string) bool {
	return !strings.HasPrefix(extensionID, "AWS.") && !strings.HasPrefix(extensionName, "AWS.")
}

func (g *AppConfigGenerator) shouldLoadApplications() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppConfigFilter() {
		return g.hasTypedFilterFor(appConfigApplicationResourceType) || g.hasTypedAppConfigApplicationChildFilter()
	}
	return true
}

func (g *AppConfigGenerator) shouldLoadApplicationChildren(applicationResource terraformutils.Resource) bool {
	if g.hasTypedAppConfigFilter() && !g.hasTypedFilterFor(appConfigApplicationResourceType) && !g.hasTypedAppConfigApplicationChildFilter() && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.hasTypedAppConfigApplicationChildFilter() && !g.hasUntypedIDFilter() {
		if !g.applicationMatchesPreDiscoveryFilters(applicationResource.InstanceState.ID) {
			return false
		}
	}

	applicationID := applicationResource.InstanceState.ID
	for _, childServiceName := range appConfigApplicationChildResourceTypes {
		if g.shouldLoadApplicationChildResourceType(childServiceName, applicationID) {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) shouldLoadConfigurationProfiles(applicationID string) bool {
	return g.shouldLoadApplicationChildResourceType(appConfigConfigurationProfileResourceType, applicationID) ||
		g.shouldLoadApplicationChildResourceType(appConfigHostedConfigurationVersionResourceType, applicationID)
}

func (g *AppConfigGenerator) shouldLoadEnvironments(applicationID string) bool {
	return g.shouldLoadApplicationChildResourceType(appConfigEnvironmentResourceType, applicationID) ||
		g.shouldLoadApplicationChildResourceType(appConfigDeploymentResourceType, applicationID)
}

func (g *AppConfigGenerator) shouldLoadHostedConfigurationVersions(applicationID, configurationProfileID, locationURI string) bool {
	if locationURI != appConfigHostedLocationURI {
		return false
	}
	if !g.shouldLoadApplicationChildResourceType(appConfigHostedConfigurationVersionResourceType, applicationID) {
		return false
	}
	return g.initialIDFiltersCanMatchHostedConfigurationProfile(applicationID, configurationProfileID)
}

func (g *AppConfigGenerator) shouldLoadDeployments(applicationID, environmentID string) bool {
	if !g.shouldLoadApplicationChildResourceType(appConfigDeploymentResourceType, applicationID) {
		return false
	}
	return g.initialIDFiltersCanMatchDeploymentEnvironment(applicationID, environmentID)
}

func (g *AppConfigGenerator) shouldLoadTopLevelResourceType(serviceName string) bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppConfigFilter() {
		return g.hasTypedFilterFor(serviceName)
	}
	return true
}

func (g *AppConfigGenerator) shouldLoadApplicationChildResourceType(serviceName, applicationID string) bool {
	hasTypedChildFilter := g.hasTypedFilterFor(serviceName)
	if g.hasTypedAppConfigApplicationChildFilter() && !hasTypedChildFilter {
		return false
	}
	if g.hasTypedAppConfigFilter() && !hasTypedChildFilter && !g.hasTypedFilterFor(appConfigApplicationResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.initialIDFiltersCanMatchApplicationChild(serviceName, applicationID) {
		return false
	}
	if !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return g.applicationMatchesPreDiscoveryFilters(applicationID)
	}
	if hasTypedChildFilter && !g.hasIDFilterFor(serviceName) && !g.hasUntypedIDFilter() && g.hasTypedFilterFor(appConfigApplicationResourceType) {
		return g.applicationMatchesPreDiscoveryFilters(applicationID)
	}
	return true
}

func (g *AppConfigGenerator) applicationMatchesPreDiscoveryFilters(applicationID string) bool {
	applicationResource := newAppConfigApplicationResource(applicationID, applicationID)
	if !g.resourceMatchesInitialIDFilters(appConfigApplicationResourceType, applicationResource) {
		return false
	}
	return !g.hasTypedNonIDFilterFor(appConfigApplicationResourceType)
}

func (g *AppConfigGenerator) shouldAppendTopLevelResource(serviceName string, resource terraformutils.Resource) bool {
	if !g.resourceMatchesInitialIDFilters(serviceName, resource) {
		return false
	}
	if g.hasTypedAppConfigFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *AppConfigGenerator) shouldAppendApplicationChildResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedAppConfigApplicationChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	if g.hasTypedAppConfigFilter() && !g.hasTypedAppConfigApplicationChildFilter() && !g.hasUntypedIDFilter() {
		if !g.hasTypedFilterFor(appConfigApplicationResourceType) || g.hasTypedNonIDFilterFor(appConfigApplicationResourceType) {
			return false
		}
	}
	return g.resourceMatchesInitialIDFilters(serviceName, resource)
}

func (g *AppConfigGenerator) resourceMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !filter.Filter(resource) {
			return false
		}
	}
	return true
}

func (g *AppConfigGenerator) initialIDFiltersCanMatchApplicationChild(serviceName, applicationID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appConfigAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return appConfigChildIDMayBelongToApplication(serviceName, applicationID, value)
		}) {
			return false
		}
	}
	return true
}

func (g *AppConfigGenerator) initialIDFiltersCanMatchHostedConfigurationProfile(applicationID, configurationProfileID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(appConfigHostedConfigurationVersionResourceType) {
			continue
		}
		if !appConfigAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return strings.HasPrefix(value, applicationID+appConfigHostedConfigurationVersionIDSeparator+configurationProfileID+appConfigHostedConfigurationVersionIDSeparator)
		}) {
			return false
		}
	}
	return true
}

func (g *AppConfigGenerator) initialIDFiltersCanMatchDeploymentEnvironment(applicationID, environmentID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(appConfigDeploymentResourceType) {
			continue
		}
		if !appConfigAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return strings.HasPrefix(value, applicationID+appConfigDeploymentIDSeparator+environmentID+appConfigDeploymentIDSeparator)
		}) {
			return false
		}
	}
	return true
}

func appConfigChildIDMayBelongToApplication(serviceName, applicationID, value string) bool {
	switch serviceName {
	case appConfigConfigurationProfileResourceType:
		return strings.HasSuffix(value, appConfigConfigurationProfileIDSeparator+applicationID)
	case appConfigEnvironmentResourceType:
		return strings.HasSuffix(value, appConfigEnvironmentIDSeparator+applicationID)
	case appConfigDeploymentResourceType:
		return strings.HasPrefix(value, applicationID+appConfigDeploymentIDSeparator)
	case appConfigHostedConfigurationVersionResourceType:
		return strings.HasPrefix(value, applicationID+appConfigHostedConfigurationVersionIDSeparator)
	default:
		return value == applicationID
	}
}

func appConfigAnyAcceptableIDMatches(values []string, match func(string) bool) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasTypedAppConfigApplicationChildFilter() bool {
	for _, serviceName := range appConfigApplicationChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasTypedAppConfigFilter() bool {
	for _, serviceName := range appConfigResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasTypedNonIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppConfigGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}
