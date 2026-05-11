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
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	athenaWorkGroupResourceType           = "aws_athena_workgroup"
	athenaDataCatalogResourceType         = "aws_athena_data_catalog"
	athenaNamedQueryResourceType          = "aws_athena_named_query"
	athenaPreparedStatementResourceType   = "aws_athena_prepared_statement"
	athenaCapacityReservationResourceType = "aws_athena_capacity_reservation"
	athenaDefaultDataCatalogName          = "AwsDataCatalog"
	athenaDefaultWorkGroupName            = "primary"
	athenaPreparedStatementIDSeparator    = "/"
)

var athenaAllowEmptyValues = []string{"tags.", "parameters.", "^description$", "^configuration\\.\\d+\\.enforce_workgroup_configuration$"}

type AthenaGenerator struct {
	AWSService
}

type athenaOptionalResourceLoader struct {
	name string
	load func() error
}

func (g *AthenaGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}

	svc := athena.NewFromConfig(config)
	var workGroupNames []string
	g.loadOptionalResources([]athenaOptionalResourceLoader{
		{name: "workgroups", load: func() error {
			var err error
			workGroupNames, err = g.loadWorkGroups(svc)
			return err
		}},
		{name: "data catalogs", load: func() error { return g.loadDataCatalogs(svc) }},
		{name: "named queries", load: func() error { return g.loadNamedQueries(svc, workGroupNames) }},
		{name: "capacity reservations", load: func() error { return g.loadCapacityReservations(svc) }},
	})

	return nil
}

func (g *AthenaGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo == nil {
			continue
		}
		switch resource.InstanceInfo.Type {
		case athenaNamedQueryResourceType:
			wrapAthenaQueryHeredoc(g, &g.Resources[i], "query")
		case athenaPreparedStatementResourceType:
			wrapAthenaQueryHeredoc(g, &g.Resources[i], "query_statement")
		}
	}
	return nil
}

func (g *AthenaGenerator) loadOptionalResources(loaders []athenaOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping Athena %s: %v", loader.name, err)
		}
	}
}

func (g *AthenaGenerator) loadWorkGroups(svc *athena.Client) ([]string, error) {
	workGroupNames := []string{}
	paginator := athena.NewListWorkGroupsPaginator(svc, &athena.ListWorkGroupsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return workGroupNames, err
		}
		for _, workGroup := range page.WorkGroups {
			workGroupName := StringValue(workGroup.Name)
			if workGroupName == "" || !athenaWorkGroupImportable(workGroup) {
				continue
			}
			workGroupNames = append(workGroupNames, workGroupName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				workGroupName,
				workGroupName,
				athenaWorkGroupResourceType,
				"aws",
				athenaAllowEmptyValues,
			))
			if err := g.loadPreparedStatements(svc, workGroupName); err != nil {
				log.Printf("Skipping Athena prepared statements for workgroup %s: %v", workGroupName, err)
			}
		}
	}
	return workGroupNames, nil
}

func (g *AthenaGenerator) loadDataCatalogs(svc *athena.Client) error {
	paginator := athena.NewListDataCatalogsPaginator(svc, &athena.ListDataCatalogsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, summary := range page.DataCatalogsSummary {
			catalogName := StringValue(summary.CatalogName)
			if catalogName == "" || !athenaDataCatalogSummaryImportable(summary) {
				continue
			}
			output, err := svc.GetDataCatalog(context.TODO(), &athena.GetDataCatalogInput{
				Name: aws.String(catalogName),
			})
			if athenaResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newAthenaDataCatalogResource(output.DataCatalog); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AthenaGenerator) loadNamedQueries(svc *athena.Client, workGroupNames []string) error {
	if len(workGroupNames) == 0 {
		workGroupNames = []string{athenaDefaultWorkGroupName}
	}
	for _, workGroupName := range workGroupNames {
		if err := g.loadNamedQueriesForWorkGroup(svc, workGroupName); err != nil {
			log.Printf("Skipping Athena named queries for workgroup %s: %v", workGroupName, err)
		}
	}
	return nil
}

func (g *AthenaGenerator) loadNamedQueriesForWorkGroup(svc *athena.Client, workGroupName string) error {
	if workGroupName == "" {
		return nil
	}
	paginator := athena.NewListNamedQueriesPaginator(svc, athenaNamedQueriesInput(workGroupName))
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, chunk := range athenaChunkStrings(page.NamedQueryIds, 50) {
			output, err := svc.BatchGetNamedQuery(context.TODO(), &athena.BatchGetNamedQueryInput{
				NamedQueryIds: chunk,
			})
			if err != nil {
				return err
			}
			for _, namedQuery := range output.NamedQueries {
				if resource, ok := newAthenaNamedQueryResource(namedQuery); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func athenaNamedQueriesInput(workGroupName string) *athena.ListNamedQueriesInput {
	return &athena.ListNamedQueriesInput{WorkGroup: aws.String(workGroupName)}
}

func (g *AthenaGenerator) loadPreparedStatements(svc *athena.Client, workGroupName string) error {
	paginator := athena.NewListPreparedStatementsPaginator(svc, &athena.ListPreparedStatementsInput{
		WorkGroup: aws.String(workGroupName),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, preparedStatement := range page.PreparedStatements {
			statementName := StringValue(preparedStatement.StatementName)
			if statementName == "" {
				continue
			}
			id := athenaPreparedStatementImportID(workGroupName, statementName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				athenaResourceName(workGroupName, statementName),
				athenaPreparedStatementResourceType,
				"aws",
				athenaAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *AthenaGenerator) loadCapacityReservations(svc *athena.Client) error {
	paginator := athena.NewListCapacityReservationsPaginator(svc, &athena.ListCapacityReservationsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, reservation := range page.CapacityReservations {
			if resource, ok := newAthenaCapacityReservationResource(reservation); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newAthenaDataCatalogResource(catalog *athenatypes.DataCatalog) (terraformutils.Resource, bool) {
	if catalog == nil {
		return terraformutils.Resource{}, false
	}
	catalogName := StringValue(catalog.Name)
	if catalogName == "" || catalogName == athenaDefaultDataCatalogName {
		return terraformutils.Resource{}, false
	}
	if catalog.Type == "" || !athenaDataCatalogParametersSafe(catalog.Parameters) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":        catalogName,
		"description": StringValue(catalog.Description),
		"type":        string(catalog.Type),
	}
	addStringMapAttributes(attributes, "parameters", catalog.Parameters)
	return terraformutils.NewResource(
		catalogName,
		catalogName,
		athenaDataCatalogResourceType,
		"aws",
		attributes,
		athenaAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAthenaNamedQueryResource(namedQuery athenatypes.NamedQuery) (terraformutils.Resource, bool) {
	namedQueryID := StringValue(namedQuery.NamedQueryId)
	queryName := StringValue(namedQuery.Name)
	if namedQueryID == "" || queryName == "" || StringValue(namedQuery.Database) == "" || StringValue(namedQuery.QueryString) == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		namedQueryID,
		athenaResourceName(StringValue(namedQuery.WorkGroup), StringValue(namedQuery.Database), queryName, namedQueryID),
		athenaNamedQueryResourceType,
		"aws",
		athenaAllowEmptyValues,
	), true
}

func newAthenaCapacityReservationResource(reservation athenatypes.CapacityReservation) (terraformutils.Resource, bool) {
	reservationName := StringValue(reservation.Name)
	if reservationName == "" || !athenaCapacityReservationImportable(reservation) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":        reservationName,
		"target_dpus": strconv.FormatInt(int64(aws.ToInt32(reservation.TargetDpus)), 10),
	}
	return terraformutils.NewResource(
		reservationName,
		reservationName,
		athenaCapacityReservationResourceType,
		"aws",
		attributes,
		athenaAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func athenaPreparedStatementImportID(workGroupName string, statementName string) string {
	return strings.Join([]string{workGroupName, statementName}, athenaPreparedStatementIDSeparator)
}

func athenaResourceName(parts ...string) string {
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	if len(segments) == 0 {
		return "athena_resource"
	}
	return strings.Join(segments, "/")
}

func athenaWorkGroupImportable(workGroup athenatypes.WorkGroupSummary) bool {
	return workGroup.State != ""
}

func athenaDataCatalogSummaryImportable(summary athenatypes.DataCatalogSummary) bool {
	if StringValue(summary.CatalogName) == athenaDefaultDataCatalogName {
		return false
	}
	switch summary.Status {
	case "", athenatypes.DataCatalogStatusCreateComplete:
		return true
	default:
		return false
	}
}

func athenaDataCatalogParametersSafe(parameters map[string]string) bool {
	if len(parameters) == 0 {
		return false
	}
	for key, value := range parameters {
		if containsSensitiveToken(key) || containsSensitiveToken(value) {
			return false
		}
	}
	return true
}

func athenaCapacityReservationImportable(reservation athenatypes.CapacityReservation) bool {
	return reservation.Status == athenatypes.CapacityReservationStatusActive && aws.ToInt32(reservation.TargetDpus) >= 24
}

func athenaResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	code := strings.ToLower(apiErr.ErrorCode())
	message := strings.ToLower(apiErr.ErrorMessage())
	return strings.Contains(code, "notfound") || strings.Contains(message, "not found")
}

func addStringMapAttributes(attributes map[string]string, prefix string, values map[string]string) {
	attributes[prefix+".%"] = strconv.Itoa(len(values))
	for key, value := range values {
		attributes[fmt.Sprintf("%s.%s", prefix, key)] = value
	}
}

func athenaChunkStrings(values []string, size int) [][]string {
	if size <= 0 || len(values) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func containsSensitiveToken(value string) bool {
	normalized := strings.ToLower(value)
	for _, token := range []string{"password", "secret", "token", "private_key", "credential", "access_key"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func wrapAthenaQueryHeredoc(g *AthenaGenerator, resource *terraformutils.Resource, field string) {
	query, ok := resource.Item[field].(string)
	if !ok || query == "" || strings.HasPrefix(query, "<<QUERY\n") {
		return
	}
	resource.Item[field] = fmt.Sprintf("<<QUERY\n%s\nQUERY", g.escapeAwsInterpolation(query))
}
