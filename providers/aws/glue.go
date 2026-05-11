// SPDX-License-Identifier: Apache-2.0

//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	glueConnectionResourceType                    = "aws_glue_connection"
	glueDataCatalogEncryptionSettingsResourceType = "aws_glue_data_catalog_encryption_settings"
	glueSchemaResourceType                        = "aws_glue_schema"
	gluePartitionIndexResourceType                = "aws_glue_partition_index"
	glueCatalogTableOptimizerResourceType         = "aws_glue_catalog_table_optimizer"
	glueCatalogTableOptimizerIDSeparator          = ","
)

var glueAllowEmptyValues = []string{
	"tags.",
	"data_catalog_encryption_settings.*.connection_password_encryption.*.return_connection_password_encrypted",
	"configuration.*.enabled",
}

type GlueGenerator struct {
	AWSService
}

type glueOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *GlueGenerator) loadOptionalResources(loaders []glueOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if glueResourceMissing(err) {
				continue
			}
			log.Printf("Skipping Glue %s: %v", loader.name, err)
		}
	}
}

func (g *GlueGenerator) loadGlueCrawlers(svc *glue.Client) error {
	p := glue.NewGetCrawlersPaginator(svc, &glue.GetCrawlersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, crawler := range page.Crawlers {
			crawlerName := StringValue(crawler.Name)
			if crawlerName == "" {
				continue
			}
			resource := terraformutils.NewSimpleResource(crawlerName, crawlerName,
				"aws_glue_crawler",
				"aws",
				glueAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueCatalogDatabase(svc *glue.Client, account *string) (databaseNames []*string, error error) {
	p := glue.NewGetDatabasesPaginator(svc, &glue.GetDatabasesInput{})
	for p.HasMorePages() {
		page, error := p.NextPage(context.TODO())
		if error != nil {
			return databaseNames, error
		}
		for _, catalogDatabase := range page.DatabaseList {
			databaseName := StringValue(catalogDatabase.Name)
			if databaseName == "" {
				continue
			}
			// format of ID is "CATALOG-ID:DATABASE-NAME".
			// CATALOG-ID is AWS Account ID
			// https://docs.aws.amazon.com/cli/latest/reference/glue/create-database.html#options
			id := *account + ":" + databaseName
			resource := terraformutils.NewSimpleResource(id, databaseName,
				"aws_glue_catalog_database",
				"aws",
				glueAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
			databaseNames = append(databaseNames, catalogDatabase.Name)
		}
	}
	return databaseNames, nil
}

func (g *GlueGenerator) loadGlueCatalogTable(svc *glue.Client, account *string, databaseName *string) error {
	// format of ID is "CATALOG-ID:DATABASE-NAME:TABLE-NAME".
	// CATALOG-ID is AWS Account ID
	// https://docs.aws.amazon.com/cli/latest/reference/glue/create-database.html#options
	p := glue.NewGetTablesPaginator(svc, &glue.GetTablesInput{DatabaseName: databaseName})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, catalogTable := range page.TableList {
			tableName := StringValue(catalogTable.Name)
			if tableName == "" {
				continue
			}
			databaseTable := *databaseName + ":" + tableName
			id := *account + ":" + databaseTable
			resource := terraformutils.NewSimpleResource(id, databaseTable,
				"aws_glue_catalog_table",
				"aws",
				glueAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
			if err := g.loadGluePartitionIndexes(svc, account, databaseName, catalogTable.Name); err != nil {
				log.Printf("Skipping Glue partition indexes for %s: %v", databaseTable, err)
			}
			if err := g.loadGlueCatalogTableOptimizers(svc, account, databaseName, catalogTable.Name); err != nil {
				log.Printf("Skipping Glue catalog table optimizers for %s: %v", databaseTable, err)
			}
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueJobs(svc *glue.Client) error {
	p := glue.NewGetJobsPaginator(svc, &glue.GetJobsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, job := range page.Jobs {
			jobName := StringValue(job.Name)
			if jobName == "" {
				continue
			}
			resource := terraformutils.NewSimpleResource(jobName, jobName,
				"aws_glue_job",
				"aws",
				glueAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueTriggers(svc *glue.Client) error {
	p := glue.NewGetTriggersPaginator(svc, &glue.GetTriggersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, trigger := range page.Triggers {
			triggerName := StringValue(trigger.Name)
			if triggerName == "" {
				continue
			}
			resource := terraformutils.NewSimpleResource(triggerName, triggerName,
				"aws_glue_trigger",
				"aws",
				glueAllowEmptyValues)
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueClassifiers(svc *glue.Client) error {
	p := glue.NewGetClassifiersPaginator(svc, &glue.GetClassifiersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, classifier := range page.Classifiers {
			classifierName := glueClassifierName(classifier)
			if classifierName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				classifierName,
				classifierName,
				"aws_glue_classifier",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueWorkflows(svc *glue.Client) error {
	p := glue.NewListWorkflowsPaginator(svc, &glue.ListWorkflowsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, workflowName := range page.Workflows {
			if workflowName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				workflowName,
				workflowName,
				"aws_glue_workflow",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueSecurityConfigurations(svc *glue.Client) error {
	p := glue.NewGetSecurityConfigurationsPaginator(svc, &glue.GetSecurityConfigurationsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configuration := range page.SecurityConfigurations {
			configurationName := StringValue(configuration.Name)
			if configurationName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				configurationName,
				configurationName,
				"aws_glue_security_configuration",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueDevEndpoints(svc *glue.Client) error {
	p := glue.NewGetDevEndpointsPaginator(svc, &glue.GetDevEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.DevEndpoints {
			endpointName := StringValue(endpoint.EndpointName)
			if endpointName == "" || !glueDevEndpointImportable(endpoint) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				endpointName,
				endpointName,
				"aws_glue_dev_endpoint",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueMLTransforms(svc *glue.Client) error {
	p := glue.NewGetMLTransformsPaginator(svc, &glue.GetMLTransformsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, transform := range page.Transforms {
			transformID := StringValue(transform.TransformId)
			transformName := StringValue(transform.Name)
			if transformID == "" || !glueMLTransformImportable(transform) {
				continue
			}
			if transformName == "" {
				transformName = transformID
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				transformID,
				transformName,
				"aws_glue_ml_transform",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueRegistries(svc *glue.Client) error {
	p := glue.NewListRegistriesPaginator(svc, &glue.ListRegistriesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, registry := range page.Registries {
			registryARN := StringValue(registry.RegistryArn)
			registryName := StringValue(registry.RegistryName)
			if registryARN == "" || registryName == "" || !glueRegistryImportable(registry) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				registryARN,
				registryName,
				"aws_glue_registry",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueDataQualityRulesets(svc *glue.Client) error {
	p := glue.NewListDataQualityRulesetsPaginator(svc, &glue.ListDataQualityRulesetsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ruleset := range page.Rulesets {
			rulesetName := StringValue(ruleset.Name)
			if rulesetName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				rulesetName,
				rulesetName,
				"aws_glue_data_quality_ruleset",
				"aws",
				glueAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueConnections(svc *glue.Client, account *string) error {
	p := glue.NewGetConnectionsPaginator(svc, &glue.GetConnectionsInput{CatalogId: account})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, connection := range page.ConnectionList {
			if resource, ok := newGlueConnectionResource(StringValue(account), connection); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueDataCatalogEncryptionSettings(svc *glue.Client, account *string) error {
	catalogID := StringValue(account)
	if catalogID == "" {
		return nil
	}
	output, err := svc.GetDataCatalogEncryptionSettings(context.TODO(), &glue.GetDataCatalogEncryptionSettingsInput{
		CatalogId: account,
	})
	if err != nil {
		return err
	}
	if resource, ok := newGlueDataCatalogEncryptionSettingsResource(catalogID, output.DataCatalogEncryptionSettings); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *GlueGenerator) loadGlueSchemas(svc *glue.Client) error {
	p := glue.NewListSchemasPaginator(svc, &glue.ListSchemasInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, schema := range page.Schemas {
			if resource, ok := newGlueSchemaResource(schema); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *GlueGenerator) loadGluePartitionIndexes(svc *glue.Client, account *string, databaseName *string, tableName *string) error {
	catalogID := StringValue(account)
	database := StringValue(databaseName)
	table := StringValue(tableName)
	if catalogID == "" || database == "" || table == "" {
		return nil
	}
	p := glue.NewGetPartitionIndexesPaginator(svc, &glue.GetPartitionIndexesInput{
		CatalogId:    account,
		DatabaseName: databaseName,
		TableName:    tableName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, index := range page.PartitionIndexDescriptorList {
			if resource, ok := newGluePartitionIndexResource(catalogID, database, table, index); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueCatalogTableOptimizers(svc *glue.Client, account *string, databaseName *string, tableName *string) error {
	catalogID := StringValue(account)
	database := StringValue(databaseName)
	table := StringValue(tableName)
	if catalogID == "" || database == "" || table == "" {
		return nil
	}
	for _, optimizerType := range (gluetypes.TableOptimizerType("")).Values() {
		output, err := svc.GetTableOptimizer(context.TODO(), &glue.GetTableOptimizerInput{
			CatalogId:    account,
			DatabaseName: databaseName,
			TableName:    tableName,
			Type:         optimizerType,
		})
		if glueResourceMissing(err) {
			continue
		}
		if err != nil {
			return err
		}
		if resource, ok := newGlueCatalogTableOptimizerResource(catalogID, database, table, optimizerType, output.TableOptimizer); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueUserDefinedFunctions(svc *glue.Client, account *string, databaseName *string) error {
	pattern := ".*"
	p := glue.NewGetUserDefinedFunctionsPaginator(svc, &glue.GetUserDefinedFunctionsInput{
		CatalogId:    account,
		DatabaseName: databaseName,
		Pattern:      &pattern,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, function := range page.UserDefinedFunctions {
			functionName := StringValue(function.FunctionName)
			if functionName == "" {
				continue
			}
			database := StringValue(databaseName)
			catalog := StringValue(account)
			if function.DatabaseName != nil {
				database = StringValue(function.DatabaseName)
			}
			if function.CatalogId != nil {
				catalog = StringValue(function.CatalogId)
			}
			if database == "" || catalog == "" {
				continue
			}
			id := glueUserDefinedFunctionImportID(catalog, database, functionName)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				id,
				glueResourceName(database, functionName),
				"aws_glue_user_defined_function",
				"aws",
				map[string]string{
					"catalog_id":    catalog,
					"database_name": database,
					"name":          functionName,
				},
				glueAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *GlueGenerator) loadGlueResourcePolicy(svc *glue.Client, region string) error {
	output, err := svc.GetResourcePolicy(context.TODO(), &glue.GetResourcePolicyInput{})
	if err != nil {
		return err
	}
	policy := StringValue(output.PolicyInJson)
	if policy == "" || region == "" {
		return nil
	}
	g.Resources = append(g.Resources, terraformutils.NewResource(
		region,
		"resource_policy",
		"aws_glue_resource_policy",
		"aws",
		map[string]string{"policy": policy},
		glueAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}

func newGlueConnectionResource(catalogID string, connection gluetypes.Connection) (terraformutils.Resource, bool) {
	connectionName := StringValue(connection.Name)
	if catalogID == "" || connectionName == "" || !glueConnectionImportable(connection) {
		return terraformutils.Resource{}, false
	}
	id := glueConnectionImportID(catalogID, connectionName)
	return terraformutils.NewSimpleResource(
		id,
		glueResourceName(catalogID, connectionName),
		glueConnectionResourceType,
		"aws",
		glueAllowEmptyValues,
	), true
}

func newGlueDataCatalogEncryptionSettingsResource(catalogID string, settings *gluetypes.DataCatalogEncryptionSettings) (terraformutils.Resource, bool) {
	if catalogID == "" || !glueDataCatalogEncryptionSettingsImportable(settings) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		catalogID,
		"data_catalog_encryption_settings",
		glueDataCatalogEncryptionSettingsResourceType,
		"aws",
		map[string]string{"catalog_id": catalogID},
		glueAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGlueSchemaResource(schema gluetypes.SchemaListItem) (terraformutils.Resource, bool) {
	schemaARN := StringValue(schema.SchemaArn)
	schemaName := StringValue(schema.SchemaName)
	if schemaARN == "" || schemaName == "" || schema.SchemaStatus != gluetypes.SchemaStatusAvailable {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		schemaARN,
		glueResourceName(StringValue(schema.RegistryName), schemaName),
		glueSchemaResourceType,
		"aws",
		glueAllowEmptyValues,
	), true
}

func newGluePartitionIndexResource(catalogID string, databaseName string, tableName string, index gluetypes.PartitionIndexDescriptor) (terraformutils.Resource, bool) {
	indexName := StringValue(index.IndexName)
	if catalogID == "" || databaseName == "" || tableName == "" || indexName == "" || !gluePartitionIndexImportable(index) {
		return terraformutils.Resource{}, false
	}
	id := gluePartitionIndexImportID(catalogID, databaseName, tableName, indexName)
	return terraformutils.NewSimpleResource(
		id,
		glueResourceName(catalogID, databaseName, tableName, indexName),
		gluePartitionIndexResourceType,
		"aws",
		glueAllowEmptyValues,
	), true
}

func newGlueCatalogTableOptimizerResource(catalogID string, databaseName string, tableName string, optimizerType gluetypes.TableOptimizerType, optimizer *gluetypes.TableOptimizer) (terraformutils.Resource, bool) {
	if catalogID == "" || databaseName == "" || tableName == "" || optimizerType == "" || optimizer == nil || optimizer.Configuration == nil {
		return terraformutils.Resource{}, false
	}
	id := glueCatalogTableOptimizerImportID(catalogID, databaseName, tableName, string(optimizerType))
	return terraformutils.NewSimpleResource(
		id,
		glueResourceName(catalogID, databaseName, tableName, string(optimizerType)),
		glueCatalogTableOptimizerResourceType,
		"aws",
		glueAllowEmptyValues,
	), true
}

func glueClassifierName(classifier gluetypes.Classifier) string {
	switch {
	case classifier.CsvClassifier != nil:
		return StringValue(classifier.CsvClassifier.Name)
	case classifier.GrokClassifier != nil:
		return StringValue(classifier.GrokClassifier.Name)
	case classifier.JsonClassifier != nil:
		return StringValue(classifier.JsonClassifier.Name)
	case classifier.XMLClassifier != nil:
		return StringValue(classifier.XMLClassifier.Name)
	default:
		return ""
	}
}

func glueDevEndpointImportable(endpoint gluetypes.DevEndpoint) bool {
	status := strings.ToUpper(StringValue(endpoint.Status))
	return status != "DELETING" && status != "FAILED"
}

func glueMLTransformImportable(transform gluetypes.MLTransform) bool {
	switch transform.Status {
	case gluetypes.TransformStatusTypeReady, gluetypes.TransformStatusTypeNotReady:
		return true
	default:
		return false
	}
}

func glueRegistryImportable(registry gluetypes.RegistryListItem) bool {
	return registry.Status == gluetypes.RegistryStatusAvailable
}

func glueConnectionImportable(connection gluetypes.Connection) bool {
	if connection.Status != "" && connection.Status != gluetypes.ConnectionStatusReady {
		return false
	}
	return !glueConnectionHasSensitiveConfiguration(connection)
}

func glueConnectionHasSensitiveConfiguration(connection gluetypes.Connection) bool {
	if connection.AuthenticationConfiguration != nil {
		return true
	}
	for _, properties := range []map[string]string{
		connection.AthenaProperties,
		connection.ConnectionProperties,
		connection.PythonProperties,
		connection.SparkProperties,
	} {
		for key, value := range properties {
			if containsSensitiveToken(key) || containsSensitiveToken(value) {
				return true
			}
		}
	}
	return false
}

func glueDataCatalogEncryptionSettingsImportable(settings *gluetypes.DataCatalogEncryptionSettings) bool {
	return settings != nil && (settings.ConnectionPasswordEncryption != nil || settings.EncryptionAtRest != nil)
}

func gluePartitionIndexImportable(index gluetypes.PartitionIndexDescriptor) bool {
	return index.IndexStatus == gluetypes.PartitionIndexStatusActive
}

func glueUserDefinedFunctionImportID(catalogID string, databaseName string, functionName string) string {
	return fmt.Sprintf("%s:%s:%s", catalogID, databaseName, functionName)
}

func glueConnectionImportID(catalogID string, connectionName string) string {
	return fmt.Sprintf("%s:%s", catalogID, connectionName)
}

func gluePartitionIndexImportID(catalogID string, databaseName string, tableName string, indexName string) string {
	return fmt.Sprintf("%s:%s:%s:%s", catalogID, databaseName, tableName, indexName)
}

func glueCatalogTableOptimizerImportID(catalogID string, databaseName string, tableName string, optimizerType string) string {
	return strings.Join([]string{catalogID, databaseName, tableName, optimizerType}, glueCatalogTableOptimizerIDSeparator)
}

func glueResourceName(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		return "glue_resource"
	}
	return strings.Join(segments, "/")
}

func glueResourceMissing(err error) bool {
	var entityNotFound *gluetypes.EntityNotFoundException
	if errors.As(err, &entityNotFound) {
		return true
	}
	var resourceNotFound *gluetypes.ResourceNotFoundException
	if errors.As(err, &resourceNotFound) {
		return true
	}
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && strings.Contains(strings.ToLower(apiErr.ErrorCode()), "notfound")
}

// Generate TerraformResources from AWS API,
// from each database create 1 TerraformResource.
// Need only database name as ID for terraform resource
// AWS api support paging
func (g *GlueGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := glue.NewFromConfig(config)

	account, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}

	if err := g.loadGlueCrawlers(svc); err != nil {
		return err
	}
	var DatabaseNames []*string
	if DatabaseNames, err = g.loadGlueCatalogDatabase(svc, account); err != nil {
		return err
	}
	for _, DatabaseName := range DatabaseNames {
		if err := g.loadGlueCatalogTable(svc, account, DatabaseName); err != nil {
			return err
		}
	}

	if err := g.loadGlueJobs(svc); err != nil {
		return err
	}

	if err := g.loadGlueTriggers(svc); err != nil {
		return err
	}

	g.loadOptionalResources([]glueOptionalResourceLoader{
		{name: "classifiers", load: func() error { return g.loadGlueClassifiers(svc) }},
		{name: "connections", load: func() error { return g.loadGlueConnections(svc, account) }},
		{name: "workflows", load: func() error { return g.loadGlueWorkflows(svc) }},
		{name: "security configurations", load: func() error { return g.loadGlueSecurityConfigurations(svc) }},
		{name: "dev endpoints", load: func() error { return g.loadGlueDevEndpoints(svc) }},
		{name: "ML transforms", load: func() error { return g.loadGlueMLTransforms(svc) }},
		{name: "registries", load: func() error { return g.loadGlueRegistries(svc) }},
		{name: "schemas", load: func() error { return g.loadGlueSchemas(svc) }},
		{name: "data quality rulesets", load: func() error { return g.loadGlueDataQualityRulesets(svc) }},
		{name: "data catalog encryption settings", load: func() error { return g.loadGlueDataCatalogEncryptionSettings(svc, account) }},
		{name: "resource policy", load: func() error { return g.loadGlueResourcePolicy(svc, config.Region) }},
	})

	for _, databaseName := range DatabaseNames {
		g.loadOptionalResources([]glueOptionalResourceLoader{
			{
				name: fmt.Sprintf("user-defined functions for %s", StringValue(databaseName)),
				load: func() error { return g.loadGlueUserDefinedFunctions(svc, account, databaseName) },
			},
		})
	}

	return nil
}

func (g *GlueGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != "aws_glue_resource_policy" {
			continue
		}
		policy, ok := resource.Item["policy"].(string)
		if !ok || policy == "" {
			continue
		}
		g.Resources[i].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
	}
	return nil
}
