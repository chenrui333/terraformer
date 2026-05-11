// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lakeformation"
	lakeformationtypes "github.com/aws/aws-sdk-go-v2/service/lakeformation/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	lakeFormationDataLakeSettingsResourceType            = "aws_lakeformation_data_lake_settings"
	lakeFormationLFTagResourceType                       = "aws_lakeformation_lf_tag"
	lakeFormationLFTagExpressionResourceType             = "aws_lakeformation_lf_tag_expression"
	lakeFormationDataCellsFilterResourceType             = "aws_lakeformation_data_cells_filter"
	lakeFormationIdentityCenterConfigurationResourceType = "aws_lakeformation_identity_center_configuration"
	lakeFormationResourceIDSeparator                     = ","
	lakeFormationLFTagIDSeparator                        = ":"
)

var lakeFormationAllowEmptyValues = []string{
	"tags.",
}

type LakeFormationGenerator struct {
	AWSService
}

type lakeFormationOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *LakeFormationGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}

	svc := lakeformation.NewFromConfig(config)
	account, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}
	catalogID := StringValue(account)

	g.loadOptionalResources([]lakeFormationOptionalResourceLoader{
		{name: "data lake settings", load: func() error { return g.loadDataLakeSettings(svc, catalogID) }},
		{name: "LF tags", load: func() error { return g.loadLFTags(svc, catalogID) }},
		{name: "LF tag expressions", load: func() error { return g.loadLFTagExpressions(svc, catalogID) }},
		{name: "data cells filters", load: func() error { return g.loadDataCellsFilters(svc) }},
		{name: "Identity Center configuration", load: func() error { return g.loadIdentityCenterConfiguration(svc, catalogID) }},
	})

	return nil
}

func (g *LakeFormationGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo == nil || g.Resources[i].InstanceInfo.Type != lakeFormationDataCellsFilterResourceType {
			continue
		}
		preserveLakeFormationDataCellsFilterWildcardBlocks(&g.Resources[i])
	}
	return nil
}

func (g *LakeFormationGenerator) loadOptionalResources(loaders []lakeFormationOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping Lake Formation %s: %v", loader.name, err)
		}
	}
}

func (g *LakeFormationGenerator) loadDataLakeSettings(svc *lakeformation.Client, catalogID string) error {
	output, err := svc.GetDataLakeSettings(context.TODO(), &lakeformation.GetDataLakeSettingsInput{
		CatalogId: aws.String(catalogID),
	})
	if lakeFormationResourceNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if resource, ok := newLakeFormationDataLakeSettingsResource(catalogID, output.DataLakeSettings); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *LakeFormationGenerator) loadLFTags(svc *lakeformation.Client, catalogID string) error {
	paginator := lakeformation.NewListLFTagsPaginator(svc, &lakeformation.ListLFTagsInput{
		CatalogId: aws.String(catalogID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tag := range page.LFTags {
			if resource, ok := newLakeFormationLFTagResource(catalogID, tag); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LakeFormationGenerator) loadLFTagExpressions(svc *lakeformation.Client, catalogID string) error {
	paginator := lakeformation.NewListLFTagExpressionsPaginator(svc, &lakeformation.ListLFTagExpressionsInput{
		CatalogId: aws.String(catalogID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, expression := range page.LFTagExpressions {
			if resource, ok := newLakeFormationLFTagExpressionResource(catalogID, expression); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LakeFormationGenerator) loadDataCellsFilters(svc *lakeformation.Client) error {
	paginator := lakeformation.NewListDataCellsFilterPaginator(svc, &lakeformation.ListDataCellsFilterInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, filter := range page.DataCellsFilters {
			if resource, ok := newLakeFormationDataCellsFilterResource(filter); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *LakeFormationGenerator) loadIdentityCenterConfiguration(svc *lakeformation.Client, catalogID string) error {
	output, err := svc.DescribeLakeFormationIdentityCenterConfiguration(context.TODO(), &lakeformation.DescribeLakeFormationIdentityCenterConfigurationInput{
		CatalogId: aws.String(catalogID),
	})
	if lakeFormationResourceNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if resource, ok := newLakeFormationIdentityCenterConfigurationResource(catalogID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newLakeFormationDataLakeSettingsResource(catalogID string, settings *lakeformationtypes.DataLakeSettings) (terraformutils.Resource, bool) {
	if catalogID == "" || !lakeFormationDataLakeSettingsImportable(settings) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		catalogID,
		"data_lake_settings",
		lakeFormationDataLakeSettingsResourceType,
		"aws",
		map[string]string{"catalog_id": catalogID},
		lakeFormationAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newLakeFormationLFTagResource(defaultCatalogID string, tag lakeformationtypes.LFTagPair) (terraformutils.Resource, bool) {
	tagKey := StringValue(tag.TagKey)
	catalogID := StringValue(tag.CatalogId)
	if catalogID == "" {
		catalogID = defaultCatalogID
	}
	if catalogID == "" || tagKey == "" || len(tag.TagValues) == 0 {
		return terraformutils.Resource{}, false
	}
	id := lakeFormationLFTagImportID(catalogID, tagKey)
	return terraformutils.NewSimpleResource(
		id,
		lakeFormationResourceName(catalogID, tagKey),
		lakeFormationLFTagResourceType,
		"aws",
		lakeFormationAllowEmptyValues,
	), true
}

func newLakeFormationLFTagExpressionResource(defaultCatalogID string, expression lakeformationtypes.LFTagExpression) (terraformutils.Resource, bool) {
	name := StringValue(expression.Name)
	catalogID := StringValue(expression.CatalogId)
	if catalogID == "" {
		catalogID = defaultCatalogID
	}
	if catalogID == "" || name == "" || len(expression.Expression) == 0 {
		return terraformutils.Resource{}, false
	}
	id := lakeFormationLFTagExpressionImportID(name, catalogID)
	resource := terraformutils.NewResource(
		id,
		lakeFormationResourceName(catalogID, name),
		lakeFormationLFTagExpressionResourceType,
		"aws",
		map[string]string{
			"name":       name,
			"catalog_id": catalogID,
		},
		lakeFormationAllowEmptyValues,
		map[string]interface{}{},
	)
	setAwsFrameworkResourcePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newLakeFormationDataCellsFilterResource(filter lakeformationtypes.DataCellsFilter) (terraformutils.Resource, bool) {
	databaseName := StringValue(filter.DatabaseName)
	filterName := StringValue(filter.Name)
	tableCatalogID := StringValue(filter.TableCatalogId)
	tableName := StringValue(filter.TableName)
	if databaseName == "" || filterName == "" || tableCatalogID == "" || tableName == "" {
		return terraformutils.Resource{}, false
	}
	id := lakeFormationDataCellsFilterImportID(databaseName, filterName, tableCatalogID, tableName)
	return terraformutils.NewSimpleResource(
		id,
		lakeFormationResourceName(tableCatalogID, databaseName, tableName, filterName),
		lakeFormationDataCellsFilterResourceType,
		"aws",
		lakeFormationAllowEmptyValues,
	), true
}

func newLakeFormationIdentityCenterConfigurationResource(catalogID string, output *lakeformation.DescribeLakeFormationIdentityCenterConfigurationOutput) (terraformutils.Resource, bool) {
	if output == nil {
		return terraformutils.Resource{}, false
	}
	if catalogID == "" {
		catalogID = StringValue(output.CatalogId)
	}
	instanceARN := StringValue(output.InstanceArn)
	if catalogID == "" || instanceARN == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		catalogID,
		"identity_center_configuration",
		lakeFormationIdentityCenterConfigurationResourceType,
		"aws",
		map[string]string{
			"catalog_id":      catalogID,
			"instance_arn":    instanceARN,
			"application_arn": StringValue(output.ApplicationArn),
			"resource_share":  StringValue(output.ResourceShare),
		},
		lakeFormationAllowEmptyValues,
		map[string]interface{}{},
	)
	setAwsFrameworkResourcePreserveIDAfterRefresh(&resource)
	return resource, true
}

func preserveLakeFormationDataCellsFilterWildcardBlocks(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil || resource.Item == nil {
		return
	}
	tableData, ok := resource.Item["table_data"].([]interface{})
	if !ok {
		return
	}
	for i := range tableData {
		tableDataItem, ok := tableData[i].(map[string]interface{})
		if !ok {
			continue
		}
		tablePrefix := "table_data." + strconv.Itoa(i)
		if lakeFormationStateBlockCount(resource, tablePrefix+".column_wildcard") > 0 {
			if _, exists := tableDataItem["column_wildcard"]; !exists {
				tableDataItem["column_wildcard"] = []interface{}{map[string]interface{}{}}
			}
		}
		preserveLakeFormationDataCellsFilterRowWildcards(resource, tableDataItem, tablePrefix)
	}
}

func preserveLakeFormationDataCellsFilterRowWildcards(resource *terraformutils.Resource, tableDataItem map[string]interface{}, tablePrefix string) {
	rowFilterCount := lakeFormationStateBlockCount(resource, tablePrefix+".row_filter")
	if rowFilterCount == 0 {
		return
	}
	rowFilters, _ := tableDataItem["row_filter"].([]interface{})
	for len(rowFilters) < rowFilterCount {
		rowFilters = append(rowFilters, map[string]interface{}{})
	}
	changed := false
	for i := 0; i < rowFilterCount; i++ {
		rowFilter, ok := rowFilters[i].(map[string]interface{})
		if !ok {
			rowFilter = map[string]interface{}{}
			rowFilters[i] = rowFilter
		}
		rowPrefix := tablePrefix + ".row_filter." + strconv.Itoa(i)
		if lakeFormationStateBlockCount(resource, rowPrefix+".all_rows_wildcard") == 0 {
			continue
		}
		if _, exists := rowFilter["filter_expression"]; exists {
			delete(rowFilter, "filter_expression")
			changed = true
		}
		if _, exists := rowFilter["all_rows_wildcard"]; !exists {
			rowFilter["all_rows_wildcard"] = []interface{}{map[string]interface{}{}}
			changed = true
		}
	}
	if changed {
		tableDataItem["row_filter"] = rowFilters
	}
}

func lakeFormationStateBlockCount(resource *terraformutils.Resource, key string) int {
	if resource == nil || resource.InstanceState == nil {
		return 0
	}
	count, err := strconv.Atoi(resource.InstanceState.Attributes[key+".#"])
	if err != nil {
		return 0
	}
	return count
}

func lakeFormationDataLakeSettingsImportable(settings *lakeformationtypes.DataLakeSettings) bool {
	if settings == nil {
		return false
	}
	if len(settings.DataLakeAdmins) > 0 ||
		len(settings.ReadOnlyAdmins) > 0 ||
		lakeFormationDefaultPermissionsConfigured(settings.CreateDatabaseDefaultPermissions) ||
		lakeFormationDefaultPermissionsConfigured(settings.CreateTableDefaultPermissions) ||
		len(settings.ExternalDataFilteringAllowList) > 0 ||
		len(settings.AuthorizedSessionTagValueList) > 0 ||
		len(settings.TrustedResourceOwners) > 0 ||
		aws.ToBool(settings.AllowExternalDataFiltering) ||
		aws.ToBool(settings.AllowFullTableExternalDataAccess) {
		return true
	}
	for key, value := range settings.Parameters {
		if key == "CROSS_ACCOUNT_VERSION" && value == "1" {
			continue
		}
		if key == "SET_CONTEXT" && strings.EqualFold(value, "TRUE") {
			continue
		}
		if value != "" {
			return true
		}
	}
	return false
}

func lakeFormationDefaultPermissionsConfigured(permissions []lakeformationtypes.PrincipalPermissions) bool {
	for _, permission := range permissions {
		if permission.Principal == nil || StringValue(permission.Principal.DataLakePrincipalIdentifier) != "IAM_ALLOWED_PRINCIPALS" {
			return true
		}
		if len(permission.Permissions) != 1 || permission.Permissions[0] != lakeformationtypes.PermissionAll {
			return true
		}
	}
	return false
}

func lakeFormationLFTagImportID(catalogID string, tagKey string) string {
	return strings.Join([]string{catalogID, tagKey}, lakeFormationLFTagIDSeparator)
}

func lakeFormationLFTagExpressionImportID(name string, catalogID string) string {
	return strings.Join([]string{name, catalogID}, lakeFormationResourceIDSeparator)
}

func lakeFormationDataCellsFilterImportID(databaseName string, filterName string, tableCatalogID string, tableName string) string {
	return strings.Join([]string{databaseName, filterName, tableCatalogID, tableName}, lakeFormationResourceIDSeparator)
}

func lakeFormationResourceName(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		return "lakeformation_resource"
	}
	return strings.Join(segments, "/")
}

func lakeFormationResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var entityNotFound *lakeformationtypes.EntityNotFoundException
	if errors.As(err, &entityNotFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.ToLower(apiErr.ErrorCode())
	message := strings.ToLower(apiErr.ErrorMessage())
	return strings.Contains(code, "notfound") || strings.Contains(message, "not found")
}
