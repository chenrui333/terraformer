// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appsync"
	appsynctypes "github.com/aws/aws-sdk-go-v2/service/appsync/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appSyncGraphQLAPIResourceType = "appsync_graphql_api"
	appSyncAPICacheResourceType   = "appsync_api_cache"
	//nolint:gosec // Terraform resource type name, not a credential.
	appSyncAPIKeyResourceType                    = "appsync_api_key"
	appSyncDataSourceResourceType                = "appsync_datasource"
	appSyncDomainNameResourceType                = "appsync_domain_name"
	appSyncDomainNameAPIAssociationResourceType  = "appsync_domain_name_api_association"
	appSyncFunctionResourceType                  = "appsync_function"
	appSyncResolverResourceType                  = "appsync_resolver"
	appSyncSourceAPIAssociationResourceType      = "appsync_source_api_association"
	appSyncTypeResourceType                      = "appsync_type"
	appSyncAPIKeyResourceIDSeparator             = ":"
	appSyncDataSourceResourceIDSeparator         = "-"
	appSyncFunctionResourceIDSeparator           = "-"
	appSyncResolverResourceIDSeparator           = "-"
	appSyncSourceAPIAssociationResourceSeparator = ","
	appSyncTypeResourceIDSeparator               = ":"
)

var (
	appSyncAllowEmptyValues      = []string{"tags."}
	appSyncTopLevelResourceTypes = []string{
		appSyncGraphQLAPIResourceType,
		appSyncDomainNameResourceType,
	}
	appSyncAPIChildResourceTypes = []string{
		appSyncAPICacheResourceType,
		appSyncAPIKeyResourceType,
		appSyncDataSourceResourceType,
		appSyncFunctionResourceType,
		appSyncResolverResourceType,
		appSyncSourceAPIAssociationResourceType,
		appSyncTypeResourceType,
	}
	appSyncChildResourceTypes = append([]string{
		appSyncDomainNameAPIAssociationResourceType,
	}, appSyncAPIChildResourceTypes...)
	appSyncResourceTypes         = append(appSyncTopLevelResourceTypes, appSyncChildResourceTypes...)
	appSyncTypeDefinitionFormats = []appsynctypes.TypeDefinitionFormat{
		appsynctypes.TypeDefinitionFormatSdl,
		appsynctypes.TypeDefinitionFormatJson,
	}
)

type AppSyncGenerator struct {
	AWSService
}

type appSyncOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *AppSyncGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := appsync.NewFromConfig(config)

	if g.shouldLoadGraphQLAPIs() {
		if err := g.loadGraphQLAPIs(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadDomainNames() {
		if g.shouldRequireDomainNameLoad() {
			if err := g.loadDomainNames(svc); err != nil {
				return err
			}
		} else {
			g.loadOptionalResources([]appSyncOptionalResourceLoader{
				{name: "domain names", load: func() error { return g.loadDomainNames(svc) }},
			})
		}
	}
	return nil
}

func (g *AppSyncGenerator) loadGraphQLAPIs(svc *appsync.Client) error {
	p := appsync.NewListGraphqlApisPaginator(svc, &appsync.ListGraphqlApisInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, api := range page.GraphqlApis {
			apiID := StringValue(api.ApiId)
			apiName := StringValue(api.Name)
			if apiID == "" || apiName == "" {
				continue
			}
			apiResource := newAppSyncGraphQLAPIResource(apiID, apiName)
			if g.shouldAppendGraphQLAPIResource(apiResource) {
				g.Resources = append(g.Resources, apiResource)
			}
			if !g.shouldLoadGraphQLAPIChildren(apiResource) {
				continue
			}
			g.loadOptionalResources(g.graphQLAPIChildLoaders(svc, apiID))
		}
	}
	return nil
}

func (g *AppSyncGenerator) graphQLAPIChildLoaders(svc *appsync.Client, apiID string) []appSyncOptionalResourceLoader {
	loaders := []appSyncOptionalResourceLoader{}
	if g.shouldLoadAPIChildResourceType(appSyncAPICacheResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("API cache for %s", apiID), load: func() error {
			return g.addAPICache(svc, apiID)
		}})
	}
	if g.shouldLoadAPIChildResourceType(appSyncAPIKeyResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("API keys for %s", apiID), load: func() error {
			return g.addAPIKeys(svc, apiID)
		}})
	}
	if g.shouldLoadAPIChildResourceType(appSyncDataSourceResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("data sources for %s", apiID), load: func() error {
			return g.addDataSources(svc, apiID)
		}})
	}
	if g.shouldLoadAPIChildResourceType(appSyncFunctionResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("functions for %s", apiID), load: func() error {
			return g.addFunctions(svc, apiID)
		}})
	}
	if g.shouldLoadAPIChildResourceType(appSyncTypeResourceType, apiID) || g.shouldLoadAPIChildResourceType(appSyncResolverResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("types and resolvers for %s", apiID), load: func() error {
			return g.addTypesAndResolvers(svc, apiID)
		}})
	}
	if g.shouldLoadAPIChildResourceType(appSyncSourceAPIAssociationResourceType, apiID) {
		loaders = append(loaders, appSyncOptionalResourceLoader{name: fmt.Sprintf("source API associations for %s", apiID), load: func() error {
			return g.addSourceAPIAssociations(svc, apiID)
		}})
	}
	return loaders
}

func (g *AppSyncGenerator) loadOptionalResources(loaders []appSyncOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if appSyncResourceMissing(err) {
				continue
			}
			log.Printf("Skipping AppSync %s: %v", loader.name, err)
		}
	}
}

func (g *AppSyncGenerator) addAPICache(svc *appsync.Client, apiID string) error {
	output, err := svc.GetApiCache(context.TODO(), &appsync.GetApiCacheInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	if output == nil || output.ApiCache == nil {
		return nil
	}
	resource := newAppSyncAPICacheResource(apiID)
	if g.shouldAppendAppSyncChildResource(appSyncAPICacheResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *AppSyncGenerator) addAPIKeys(svc *appsync.Client, apiID string) error {
	p := appsync.NewListApiKeysPaginator(svc, &appsync.ListApiKeysInput{ApiId: aws.String(apiID)})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, key := range page.ApiKeys {
			keyID := StringValue(key.Id)
			if keyID == "" {
				continue
			}
			resource := newAppSyncAPIKeyResource(apiID, keyID)
			if g.shouldAppendAppSyncChildResource(appSyncAPIKeyResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addDataSources(svc *appsync.Client, apiID string) error {
	p := appsync.NewListDataSourcesPaginator(svc, &appsync.ListDataSourcesInput{ApiId: aws.String(apiID)})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, dataSource := range page.DataSources {
			name := StringValue(dataSource.Name)
			if name == "" {
				continue
			}
			resource := newAppSyncDataSourceResource(apiID, name)
			if g.shouldAppendAppSyncChildResource(appSyncDataSourceResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addFunctions(svc *appsync.Client, apiID string) error {
	p := appsync.NewListFunctionsPaginator(svc, &appsync.ListFunctionsInput{ApiId: aws.String(apiID)})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, function := range page.Functions {
			functionID := StringValue(function.FunctionId)
			if functionID == "" {
				continue
			}
			resource := newAppSyncFunctionResource(apiID, functionID)
			if g.shouldAppendAppSyncChildResource(appSyncFunctionResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addTypesAndResolvers(svc *appsync.Client, apiID string) error {
	seenTypes := map[string]struct{}{}
	seenResolverTypes := map[string]struct{}{}
	for _, format := range appSyncTypeDefinitionFormats {
		p := appsync.NewListTypesPaginator(svc, &appsync.ListTypesInput{
			ApiId:  aws.String(apiID),
			Format: format,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				return err
			}
			for _, graphqlType := range page.Types {
				typeName := StringValue(graphqlType.Name)
				if typeName == "" {
					continue
				}
				if g.shouldLoadAPIChildResourceType(appSyncTypeResourceType, apiID) {
					if _, seen := seenTypes[typeName]; seen {
						continue
					}
					resource := newAppSyncTypeResource(apiID, string(format), typeName)
					if g.shouldAppendAppSyncChildResource(appSyncTypeResourceType, resource) {
						g.Resources = append(g.Resources, resource)
						seenTypes[typeName] = struct{}{}
					}
				}
				if !g.shouldLoadAPIChildResourceType(appSyncResolverResourceType, apiID) {
					continue
				}
				if _, seen := seenResolverTypes[typeName]; seen {
					continue
				}
				seenResolverTypes[typeName] = struct{}{}
				if err := g.addResolvers(svc, apiID, typeName); err != nil {
					if appSyncResourceMissing(err) {
						continue
					}
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addResolvers(svc *appsync.Client, apiID, typeName string) error {
	p := appsync.NewListResolversPaginator(svc, &appsync.ListResolversInput{
		ApiId:    aws.String(apiID),
		TypeName: aws.String(typeName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, resolver := range page.Resolvers {
			fieldName := StringValue(resolver.FieldName)
			if fieldName == "" {
				continue
			}
			resource := newAppSyncResolverResource(apiID, typeName, fieldName)
			if g.shouldAppendAppSyncChildResource(appSyncResolverResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addSourceAPIAssociations(svc *appsync.Client, mergedAPIID string) error {
	p := appsync.NewListSourceApiAssociationsPaginator(svc, &appsync.ListSourceApiAssociationsInput{ApiId: aws.String(mergedAPIID)})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.SourceApiAssociationSummaries {
			associationID := StringValue(association.AssociationId)
			if associationID == "" {
				continue
			}
			resource := newAppSyncSourceAPIAssociationResource(mergedAPIID, associationID)
			if g.shouldAppendAppSyncChildResource(appSyncSourceAPIAssociationResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) loadDomainNames(svc *appsync.Client) error {
	p := appsync.NewListDomainNamesPaginator(svc, &appsync.ListDomainNamesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, domain := range page.DomainNameConfigs {
			domainName := StringValue(domain.DomainName)
			if domainName == "" {
				continue
			}
			domainResource := newAppSyncDomainNameResource(domainName)
			if g.shouldAppendDomainNameResource(domainResource) {
				g.Resources = append(g.Resources, domainResource)
			}
			if !g.shouldLoadDomainNameAPIAssociation(domainName) {
				continue
			}
			if err := g.addDomainNameAPIAssociation(svc, domainName); err != nil {
				if appSyncResourceMissing(err) {
					continue
				}
				log.Printf("Skipping AppSync domain name API association for %s: %v", domainName, err)
			}
		}
	}
	return nil
}

func (g *AppSyncGenerator) addDomainNameAPIAssociation(svc *appsync.Client, domainName string) error {
	output, err := svc.GetApiAssociation(context.TODO(), &appsync.GetApiAssociationInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return err
	}
	if output == nil || output.ApiAssociation == nil {
		return nil
	}
	apiID := StringValue(output.ApiAssociation.ApiId)
	if apiID == "" {
		return nil
	}
	resource := newAppSyncDomainNameAPIAssociationResource(domainName, apiID)
	if g.shouldAppendAppSyncChildResource(appSyncDomainNameAPIAssociationResourceType, resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newAppSyncGraphQLAPIResource(apiID, apiName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(apiID, apiName, "aws_appsync_graphql_api", "aws", appSyncAllowEmptyValues)
}

func newAppSyncAPICacheResource(apiID string) terraformutils.Resource {
	return terraformutils.NewResource(
		apiID,
		appSyncResourceName(apiID, "api-cache"),
		"aws_appsync_api_cache",
		"aws",
		map[string]string{"api_id": apiID},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncAPIKeyResource(apiID, keyID string) terraformutils.Resource {
	return terraformutils.NewResource(
		appSyncAPIKeyResourceID(apiID, keyID),
		appSyncResourceName(apiID, "api-key", keyID),
		"aws_appsync_api_key",
		"aws",
		map[string]string{"api_id": apiID},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncDataSourceResource(apiID, name string) terraformutils.Resource {
	return terraformutils.NewResource(
		appSyncDataSourceResourceID(apiID, name),
		appSyncResourceName(apiID, "datasource", name),
		"aws_appsync_datasource",
		"aws",
		map[string]string{
			"api_id": apiID,
			"name":   name,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncDomainNameResource(domainName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(domainName, domainName, "aws_appsync_domain_name", "aws", appSyncAllowEmptyValues)
}

func newAppSyncDomainNameAPIAssociationResource(domainName, apiID string) terraformutils.Resource {
	return terraformutils.NewResource(
		domainName,
		appSyncResourceName(domainName, "api-association"),
		"aws_appsync_domain_name_api_association",
		"aws",
		map[string]string{
			"api_id":      apiID,
			"domain_name": domainName,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncFunctionResource(apiID, functionID string) terraformutils.Resource {
	return terraformutils.NewResource(
		appSyncFunctionResourceID(apiID, functionID),
		appSyncResourceName(apiID, "function", functionID),
		"aws_appsync_function",
		"aws",
		map[string]string{
			"api_id":      apiID,
			"function_id": functionID,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncResolverResource(apiID, typeName, fieldName string) terraformutils.Resource {
	return terraformutils.NewResource(
		appSyncResolverResourceID(apiID, typeName, fieldName),
		appSyncResourceName(apiID, "resolver", typeName, fieldName),
		"aws_appsync_resolver",
		"aws",
		map[string]string{
			"api_id": apiID,
			"field":  fieldName,
			"type":   typeName,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppSyncSourceAPIAssociationResource(mergedAPIID, associationID string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		appSyncSourceAPIAssociationResourceID(mergedAPIID, associationID),
		appSyncResourceName(mergedAPIID, "source-api-association", associationID),
		"aws_appsync_source_api_association",
		"aws",
		map[string]string{
			"association_id": associationID,
			"merged_api_id":  mergedAPIID,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
	resource.IgnoreKeys = append(resource.IgnoreKeys, "^merged_api_arn$", "^source_api_arn$")
	return resource
}

func newAppSyncTypeResource(apiID, format, typeName string) terraformutils.Resource {
	return terraformutils.NewResource(
		appSyncTypeResourceID(apiID, format, typeName),
		appSyncResourceName(apiID, "type", format, typeName),
		"aws_appsync_type",
		"aws",
		map[string]string{
			"api_id": apiID,
			"format": format,
			"name":   typeName,
		},
		appSyncAllowEmptyValues,
		map[string]interface{}{},
	)
}

func appSyncAPIKeyResourceID(apiID, keyID string) string {
	return strings.Join([]string{apiID, keyID}, appSyncAPIKeyResourceIDSeparator)
}

func appSyncDataSourceResourceID(apiID, name string) string {
	return strings.Join([]string{apiID, name}, appSyncDataSourceResourceIDSeparator)
}

func appSyncFunctionResourceID(apiID, functionID string) string {
	return strings.Join([]string{apiID, functionID}, appSyncFunctionResourceIDSeparator)
}

func appSyncResolverResourceID(apiID, typeName, fieldName string) string {
	return strings.Join([]string{apiID, typeName, fieldName}, appSyncResolverResourceIDSeparator)
}

func appSyncSourceAPIAssociationResourceID(mergedAPIID, associationID string) string {
	return strings.Join([]string{mergedAPIID, associationID}, appSyncSourceAPIAssociationResourceSeparator)
}

func appSyncTypeResourceID(apiID, format, typeName string) string {
	return strings.Join([]string{apiID, format, typeName}, appSyncTypeResourceIDSeparator)
}

func appSyncResourceName(parts ...string) string {
	nonEmptyParts := []string{}
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, ":")
}

func appSyncResourceMissing(err error) bool {
	var notFound *appsynctypes.NotFoundException
	return errors.As(err, &notFound)
}

func (g *AppSyncGenerator) shouldAppendGraphQLAPIResource(resource terraformutils.Resource) bool {
	if !g.resourceMatchesInitialIDFilters(appSyncGraphQLAPIResourceType, resource) {
		return false
	}
	if g.hasTypedAppSyncFilter() && !g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *AppSyncGenerator) shouldAppendDomainNameResource(resource terraformutils.Resource) bool {
	if !g.resourceMatchesInitialIDFilters(appSyncDomainNameResourceType, resource) {
		return false
	}
	if g.hasTypedAppSyncFilter() && !g.hasTypedFilterFor(appSyncDomainNameResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *AppSyncGenerator) shouldAppendAppSyncChildResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedAppSyncChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	if g.hasTypedAppSyncFilter() && !g.hasTypedAppSyncChildFilter() && !g.hasUntypedIDFilter() {
		switch serviceName {
		case appSyncDomainNameAPIAssociationResourceType:
			if !g.hasTypedFilterFor(appSyncDomainNameResourceType) {
				return false
			}
		default:
			if !g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) {
				return false
			}
		}
	}
	return g.resourceMatchesInitialIDFilters(serviceName, resource)
}

func (g *AppSyncGenerator) shouldLoadGraphQLAPIChildren(apiResource terraformutils.Resource) bool {
	if g.hasTypedAppSyncFilter() && !g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) && !g.hasTypedAppSyncAPIChildFilter() && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.hasTypedAppSyncAPIChildFilter() && !g.hasUntypedIDFilter() {
		if !g.resourceMatchesInitialIDFilters(appSyncGraphQLAPIResourceType, apiResource) {
			return false
		}
		if g.hasTypedNonIDFilterFor(appSyncGraphQLAPIResourceType) {
			return false
		}
	}

	apiID := apiResource.InstanceState.ID
	for _, childServiceName := range appSyncAPIChildResourceTypes {
		if g.shouldLoadAPIChildResourceType(childServiceName, apiID) {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) shouldLoadGraphQLAPIs() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppSyncFilter() {
		return g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) || g.hasTypedAppSyncAPIChildFilter()
	}
	return true
}

func (g *AppSyncGenerator) shouldLoadAPIChildResourceType(serviceName, apiID string) bool {
	hasTypedChildFilter := g.hasTypedFilterFor(serviceName)
	if g.hasTypedAppSyncAPIChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	if g.hasTypedAppSyncFilter() && !hasTypedChildFilter && !g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.initialIDFiltersCanMatchAPIChild(serviceName, apiID) {
		return false
	}
	if !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return g.graphQLAPIMatchesPreDiscoveryFilters(apiID)
	}
	if hasTypedChildFilter && !g.hasIDFilterFor(serviceName) && !g.hasUntypedIDFilter() && g.hasTypedFilterFor(appSyncGraphQLAPIResourceType) {
		return g.graphQLAPIMatchesPreDiscoveryFilters(apiID)
	}
	return true
}

func (g *AppSyncGenerator) graphQLAPIMatchesPreDiscoveryFilters(apiID string) bool {
	apiResource := newAppSyncGraphQLAPIResource(apiID, apiID)
	if !g.resourceMatchesInitialIDFilters(appSyncGraphQLAPIResourceType, apiResource) {
		return false
	}
	return !g.hasTypedNonIDFilterFor(appSyncGraphQLAPIResourceType)
}

func (g *AppSyncGenerator) shouldLoadDomainNames() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppSyncFilter() {
		return g.hasTypedFilterFor(appSyncDomainNameResourceType) || g.hasTypedFilterFor(appSyncDomainNameAPIAssociationResourceType)
	}
	return true
}

func (g *AppSyncGenerator) shouldRequireDomainNameLoad() bool {
	return g.hasTypedFilterFor(appSyncDomainNameResourceType) || g.hasTypedFilterFor(appSyncDomainNameAPIAssociationResourceType)
}

func (g *AppSyncGenerator) shouldLoadDomainNameAPIAssociation(domainName string) bool {
	hasTypedDomainFilter := g.hasTypedFilterFor(appSyncDomainNameResourceType)
	hasTypedAssociationIDFilter := g.hasTypedFilterFor(appSyncDomainNameAPIAssociationResourceType) && g.hasIDFilterFor(appSyncDomainNameAPIAssociationResourceType)
	if hasTypedDomainFilter && !hasTypedAssociationIDFilter {
		domainResource := newAppSyncDomainNameResource(domainName)
		if !g.resourceMatchesInitialIDFilters(appSyncDomainNameResourceType, domainResource) || g.hasTypedNonIDFilterFor(appSyncDomainNameResourceType) {
			return false
		}
	}
	if g.hasTypedFilterFor(appSyncDomainNameAPIAssociationResourceType) {
		return g.initialIDFiltersCanMatchDomainName(appSyncDomainNameAPIAssociationResourceType, domainName)
	}
	if hasTypedDomainFilter {
		return true
	}
	if g.hasTypedAppSyncFilter() && !g.hasUntypedIDFilter() {
		return false
	}
	return g.initialIDFiltersCanMatchDomainName(appSyncDomainNameAPIAssociationResourceType, domainName)
}

func (g *AppSyncGenerator) resourceMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
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

func (g *AppSyncGenerator) initialIDFiltersCanMatchAPIChild(serviceName, apiID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appSyncAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return appSyncChildIDMayBelongToAPI(serviceName, apiID, value)
		}) {
			return false
		}
	}
	return true
}

func (g *AppSyncGenerator) initialIDFiltersCanMatchDomainName(serviceName, domainName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appSyncAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return value == domainName
		}) {
			return false
		}
	}
	return true
}

func appSyncAnyAcceptableIDMatches(values []string, match func(string) bool) bool {
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

func appSyncChildIDMayBelongToAPI(serviceName, apiID, value string) bool {
	switch serviceName {
	case appSyncAPICacheResourceType:
		return value == apiID
	case appSyncAPIKeyResourceType, appSyncTypeResourceType:
		return strings.HasPrefix(value, apiID+appSyncAPIKeyResourceIDSeparator)
	case appSyncDataSourceResourceType, appSyncFunctionResourceType, appSyncResolverResourceType:
		return strings.HasPrefix(value, apiID+appSyncDataSourceResourceIDSeparator)
	case appSyncSourceAPIAssociationResourceType:
		return strings.HasPrefix(value, apiID+appSyncSourceAPIAssociationResourceSeparator)
	default:
		return true
	}
}

func (g *AppSyncGenerator) hasTypedAppSyncChildFilter() bool {
	for _, serviceName := range appSyncChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasTypedAppSyncAPIChildFilter() bool {
	for _, serviceName := range appSyncAPIChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasTypedAppSyncFilter() bool {
	for _, serviceName := range appSyncResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasTypedNonIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppSyncGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}
