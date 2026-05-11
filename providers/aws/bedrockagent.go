// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	bedrockagenttypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	bedrockAgentAgentResourceType                         = "aws_bedrockagent_agent"
	bedrockAgentAgentAliasResourceType                    = "aws_bedrockagent_agent_alias"
	bedrockAgentAgentKnowledgeBaseAssociationResourceType = "aws_bedrockagent_agent_knowledge_base_association"
	bedrockAgentDataSourceResourceType                    = "aws_bedrockagent_data_source"
	bedrockAgentKnowledgeBaseResourceType                 = "aws_bedrockagent_knowledge_base"
	bedrockAgentDraftVersion                              = "DRAFT"
	bedrockAgentImportIDSeparator                         = ","
	bedrockAgentResourceNameFallback                      = "bedrockagent-resource"
)

var (
	bedrockAgentAllowEmptyValues = []string{"tags."}
	bedrockAgentResourceTypes    = []string{
		bedrockAgentServiceName(bedrockAgentAgentResourceType),
		bedrockAgentServiceName(bedrockAgentAgentAliasResourceType),
		bedrockAgentServiceName(bedrockAgentAgentKnowledgeBaseAssociationResourceType),
		bedrockAgentServiceName(bedrockAgentDataSourceResourceType),
		bedrockAgentServiceName(bedrockAgentKnowledgeBaseResourceType),
	}
)

type BedrockAgentGenerator struct {
	AWSService
}

func (g *BedrockAgentGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := bedrockAgentServiceName(resource.InstanceInfo.Type)
		if g.hasTypedBedrockAgentFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
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

func (g *BedrockAgentGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := bedrockagent.NewFromConfig(config)

	loadAgents := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentResourceType))
	loadAgentAliases := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentAliasResourceType))
	loadAgentKnowledgeBaseAssociations := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentKnowledgeBaseAssociationResourceType))
	if loadAgents || loadAgentAliases || loadAgentKnowledgeBaseAssociations {
		agents, err := listBedrockAgentAgents(svc)
		if err != nil {
			return err
		}
		if loadAgents {
			g.loadAgents(agents)
		}
		if loadAgentAliases {
			if err := g.loadAgentAliases(svc, agents); err != nil {
				return err
			}
		}
		if loadAgentKnowledgeBaseAssociations {
			if err := g.loadAgentKnowledgeBaseAssociations(svc, agents); err != nil {
				return err
			}
		}
	}

	loadKnowledgeBases := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentKnowledgeBaseResourceType))
	loadDataSources := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentDataSourceResourceType))
	if loadKnowledgeBases || loadDataSources {
		knowledgeBases, err := listBedrockAgentKnowledgeBases(svc)
		if err != nil {
			return err
		}
		if loadKnowledgeBases {
			if err := g.loadKnowledgeBases(svc, knowledgeBases); err != nil {
				return err
			}
		}
		if loadDataSources {
			if err := g.loadDataSources(svc, knowledgeBases); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *BedrockAgentGenerator) shouldLoadBedrockAgentResource(serviceName string) bool {
	if !g.hasTypedBedrockAgentFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *BedrockAgentGenerator) hasTypedBedrockAgentFilter() bool {
	for _, serviceName := range bedrockAgentResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *BedrockAgentGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *BedrockAgentGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func bedrockAgentServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listBedrockAgentAgents(svc *bedrockagent.Client) ([]bedrockagenttypes.AgentSummary, error) {
	p := bedrockagent.NewListAgentsPaginator(svc, &bedrockagent.ListAgentsInput{})
	agents := []bedrockagenttypes.AgentSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		agents = append(agents, page.AgentSummaries...)
	}
	return agents, nil
}

func listBedrockAgentKnowledgeBases(svc *bedrockagent.Client) ([]bedrockagenttypes.KnowledgeBaseSummary, error) {
	p := bedrockagent.NewListKnowledgeBasesPaginator(svc, &bedrockagent.ListKnowledgeBasesInput{})
	knowledgeBases := []bedrockagenttypes.KnowledgeBaseSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		knowledgeBases = append(knowledgeBases, page.KnowledgeBaseSummaries...)
	}
	return knowledgeBases, nil
}

func (g *BedrockAgentGenerator) loadAgents(agents []bedrockagenttypes.AgentSummary) {
	for _, agent := range agents {
		if resource, ok := newBedrockAgentAgentResource(agent); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *BedrockAgentGenerator) loadAgentAliases(svc *bedrockagent.Client, agents []bedrockagenttypes.AgentSummary) error {
	for _, agent := range agents {
		agentID := StringValue(agent.AgentId)
		if agentID == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
			continue
		}
		p := bedrockagent.NewListAgentAliasesPaginator(svc, &bedrockagent.ListAgentAliasesInput{
			AgentId: &agentID,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if bedrockAgentResourceNotFound(err) {
					break
				}
				return err
			}
			for _, alias := range page.AgentAliasSummaries {
				if resource, ok := newBedrockAgentAgentAliasResource(agentID, alias); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *BedrockAgentGenerator) loadAgentKnowledgeBaseAssociations(svc *bedrockagent.Client, agents []bedrockagenttypes.AgentSummary) error {
	for _, agent := range agents {
		agentID := StringValue(agent.AgentId)
		if agentID == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
			continue
		}
		associations, err := listBedrockAgentKnowledgeBaseAssociations(svc, agentID, bedrockAgentDraftVersion)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, association := range associations {
			knowledgeBaseID := StringValue(association.KnowledgeBaseId)
			if knowledgeBaseID == "" {
				continue
			}
			agentKnowledgeBase, err := getBedrockAgentKnowledgeBaseAssociation(svc, agentID, bedrockAgentDraftVersion, knowledgeBaseID)
			if err != nil {
				if bedrockAgentResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(agentKnowledgeBase); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listBedrockAgentKnowledgeBaseAssociations(svc *bedrockagent.Client, agentID, agentVersion string) ([]bedrockagenttypes.AgentKnowledgeBaseSummary, error) {
	p := bedrockagent.NewListAgentKnowledgeBasesPaginator(svc, &bedrockagent.ListAgentKnowledgeBasesInput{
		AgentId:      &agentID,
		AgentVersion: &agentVersion,
	})
	associations := []bedrockagenttypes.AgentKnowledgeBaseSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		associations = append(associations, page.AgentKnowledgeBaseSummaries...)
	}
	return associations, nil
}

func getBedrockAgentKnowledgeBaseAssociation(svc *bedrockagent.Client, agentID, agentVersion, knowledgeBaseID string) (*bedrockagenttypes.AgentKnowledgeBase, error) {
	output, err := svc.GetAgentKnowledgeBase(context.TODO(), &bedrockagent.GetAgentKnowledgeBaseInput{
		AgentId:         &agentID,
		AgentVersion:    &agentVersion,
		KnowledgeBaseId: &knowledgeBaseID,
	})
	if err != nil {
		return nil, err
	}
	return output.AgentKnowledgeBase, nil
}

func (g *BedrockAgentGenerator) loadKnowledgeBases(svc *bedrockagent.Client, knowledgeBases []bedrockagenttypes.KnowledgeBaseSummary) error {
	for _, summary := range knowledgeBases {
		knowledgeBaseID := StringValue(summary.KnowledgeBaseId)
		if knowledgeBaseID == "" || !bedrockAgentKnowledgeBaseImportable(summary.Status) {
			continue
		}
		knowledgeBase, err := getBedrockAgentKnowledgeBase(svc, knowledgeBaseID)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		if resource, ok := newBedrockAgentKnowledgeBaseResource(knowledgeBase); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func getBedrockAgentKnowledgeBase(svc *bedrockagent.Client, knowledgeBaseID string) (*bedrockagenttypes.KnowledgeBase, error) {
	output, err := svc.GetKnowledgeBase(context.TODO(), &bedrockagent.GetKnowledgeBaseInput{
		KnowledgeBaseId: &knowledgeBaseID,
	})
	if err != nil {
		return nil, err
	}
	return output.KnowledgeBase, nil
}

func (g *BedrockAgentGenerator) loadDataSources(svc *bedrockagent.Client, knowledgeBases []bedrockagenttypes.KnowledgeBaseSummary) error {
	for _, knowledgeBase := range knowledgeBases {
		knowledgeBaseID := StringValue(knowledgeBase.KnowledgeBaseId)
		if knowledgeBaseID == "" || !bedrockAgentKnowledgeBaseImportable(knowledgeBase.Status) {
			continue
		}
		dataSources, err := listBedrockAgentDataSources(svc, knowledgeBaseID)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, summary := range dataSources {
			dataSourceID := StringValue(summary.DataSourceId)
			if dataSourceID == "" || !bedrockAgentDataSourceImportable(summary.Status) {
				continue
			}
			dataSource, err := getBedrockAgentDataSource(svc, knowledgeBaseID, dataSourceID)
			if err != nil {
				if bedrockAgentResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newBedrockAgentDataSourceResource(dataSource); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listBedrockAgentDataSources(svc *bedrockagent.Client, knowledgeBaseID string) ([]bedrockagenttypes.DataSourceSummary, error) {
	p := bedrockagent.NewListDataSourcesPaginator(svc, &bedrockagent.ListDataSourcesInput{
		KnowledgeBaseId: &knowledgeBaseID,
	})
	dataSources := []bedrockagenttypes.DataSourceSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		dataSources = append(dataSources, page.DataSourceSummaries...)
	}
	return dataSources, nil
}

func getBedrockAgentDataSource(svc *bedrockagent.Client, knowledgeBaseID, dataSourceID string) (*bedrockagenttypes.DataSource, error) {
	output, err := svc.GetDataSource(context.TODO(), &bedrockagent.GetDataSourceInput{
		DataSourceId:    &dataSourceID,
		KnowledgeBaseId: &knowledgeBaseID,
	})
	if err != nil {
		return nil, err
	}
	return output.DataSource, nil
}

func newBedrockAgentAgentResource(agent bedrockagenttypes.AgentSummary) (terraformutils.Resource, bool) {
	agentID := StringValue(agent.AgentId)
	agentName := StringValue(agent.AgentName)
	if agentID == "" || agentName == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"agent_id":   agentID,
		"agent_name": agentName,
	}
	if agent.AgentStatus == bedrockagenttypes.AgentStatusNotPrepared {
		attributes["prepare_agent"] = "false"
	}
	return terraformutils.NewResource(
		agentID,
		bedrockAgentResourceName("agent", agentName, agentID),
		bedrockAgentAgentResourceType,
		"aws",
		attributes,
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockAgentAgentAliasResource(agentID string, alias bedrockagenttypes.AgentAliasSummary) (terraformutils.Resource, bool) {
	agentAliasID := StringValue(alias.AgentAliasId)
	agentAliasName := StringValue(alias.AgentAliasName)
	if agentID == "" || agentAliasID == "" || agentAliasName == "" || !bedrockAgentAgentAliasImportable(alias.AgentAliasStatus) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		bedrockAgentAgentAliasImportID(agentAliasID, agentID),
		bedrockAgentResourceName("agent-alias", agentID, agentAliasName, agentAliasID),
		bedrockAgentAgentAliasResourceType,
		"aws",
		map[string]string{
			"agent_alias_id":   agentAliasID,
			"agent_alias_name": agentAliasName,
			"agent_id":         agentID,
		},
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockAgentAgentKnowledgeBaseAssociationResource(association *bedrockagenttypes.AgentKnowledgeBase) (terraformutils.Resource, bool) {
	if association == nil {
		return terraformutils.Resource{}, false
	}
	agentID := StringValue(association.AgentId)
	agentVersion := StringValue(association.AgentVersion)
	knowledgeBaseID := StringValue(association.KnowledgeBaseId)
	description := StringValue(association.Description)
	knowledgeBaseState := string(association.KnowledgeBaseState)
	if agentID == "" ||
		agentVersion == "" ||
		knowledgeBaseID == "" ||
		description == "" ||
		!bedrockAgentAgentKnowledgeBaseAssociationImportable(agentVersion, association.KnowledgeBaseState) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		bedrockAgentAgentKnowledgeBaseAssociationImportID(agentID, agentVersion, knowledgeBaseID),
		bedrockAgentResourceName("agent-knowledge-base-association", agentID, agentVersion, knowledgeBaseID),
		bedrockAgentAgentKnowledgeBaseAssociationResourceType,
		"aws",
		map[string]string{
			"agent_id":             agentID,
			"agent_version":        agentVersion,
			"description":          description,
			"knowledge_base_id":    knowledgeBaseID,
			"knowledge_base_state": knowledgeBaseState,
		},
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockAgentKnowledgeBaseResource(knowledgeBase *bedrockagenttypes.KnowledgeBase) (terraformutils.Resource, bool) {
	if knowledgeBase == nil {
		return terraformutils.Resource{}, false
	}
	knowledgeBaseID := StringValue(knowledgeBase.KnowledgeBaseId)
	knowledgeBaseName := StringValue(knowledgeBase.Name)
	roleARN := StringValue(knowledgeBase.RoleArn)
	if knowledgeBaseID == "" ||
		knowledgeBaseName == "" ||
		roleARN == "" ||
		!bedrockAgentKnowledgeBaseConfigurationImportable(knowledgeBase.KnowledgeBaseConfiguration) ||
		!bedrockAgentKnowledgeBaseStorageConfigurationImportable(knowledgeBase.KnowledgeBaseConfiguration, knowledgeBase.StorageConfiguration) ||
		!bedrockAgentKnowledgeBaseImportable(knowledgeBase.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		knowledgeBaseID,
		bedrockAgentResourceName("knowledge-base", knowledgeBaseName, knowledgeBaseID),
		bedrockAgentKnowledgeBaseResourceType,
		"aws",
		map[string]string{
			"id":       knowledgeBaseID,
			"name":     knowledgeBaseName,
			"role_arn": roleARN,
		},
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockAgentDataSourceResource(dataSource *bedrockagenttypes.DataSource) (terraformutils.Resource, bool) {
	if dataSource == nil {
		return terraformutils.Resource{}, false
	}
	dataSourceID := StringValue(dataSource.DataSourceId)
	knowledgeBaseID := StringValue(dataSource.KnowledgeBaseId)
	dataSourceName := StringValue(dataSource.Name)
	if dataSourceID == "" ||
		knowledgeBaseID == "" ||
		dataSourceName == "" ||
		!bedrockAgentDataSourceConfigurationImportable(dataSource.DataSourceConfiguration) ||
		!bedrockAgentDataSourceImportable(dataSource.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		bedrockAgentDataSourceImportID(dataSourceID, knowledgeBaseID),
		bedrockAgentResourceName("data-source", knowledgeBaseID, dataSourceName, dataSourceID),
		bedrockAgentDataSourceResourceType,
		"aws",
		map[string]string{
			"data_source_id":    dataSourceID,
			"knowledge_base_id": knowledgeBaseID,
			"name":              dataSourceName,
		},
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func bedrockAgentAgentAliasImportID(agentAliasID, agentID string) string {
	return agentAliasID + bedrockAgentImportIDSeparator + agentID
}

func bedrockAgentAgentKnowledgeBaseAssociationImportID(agentID, agentVersion, knowledgeBaseID string) string {
	return agentID + bedrockAgentImportIDSeparator + agentVersion + bedrockAgentImportIDSeparator + knowledgeBaseID
}

func bedrockAgentDataSourceImportID(dataSourceID, knowledgeBaseID string) string {
	return dataSourceID + bedrockAgentImportIDSeparator + knowledgeBaseID
}

func bedrockAgentResourceName(parts ...string) string {
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return bedrockAgentResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func bedrockAgentAgentImportable(status bedrockagenttypes.AgentStatus) bool {
	return status == bedrockagenttypes.AgentStatusPrepared || status == bedrockagenttypes.AgentStatusNotPrepared
}

func bedrockAgentAgentAliasImportable(status bedrockagenttypes.AgentAliasStatus) bool {
	return status == bedrockagenttypes.AgentAliasStatusPrepared || status == bedrockagenttypes.AgentAliasStatusDissociated
}

func bedrockAgentAgentKnowledgeBaseAssociationImportable(agentVersion string, state bedrockagenttypes.KnowledgeBaseState) bool {
	return agentVersion == bedrockAgentDraftVersion &&
		(state == bedrockagenttypes.KnowledgeBaseStateEnabled || state == bedrockagenttypes.KnowledgeBaseStateDisabled)
}

func bedrockAgentKnowledgeBaseImportable(status bedrockagenttypes.KnowledgeBaseStatus) bool {
	return status == bedrockagenttypes.KnowledgeBaseStatusActive
}

func bedrockAgentDataSourceImportable(status bedrockagenttypes.DataSourceStatus) bool {
	return status == bedrockagenttypes.DataSourceStatusAvailable
}

func bedrockAgentKnowledgeBaseConfigurationImportable(config *bedrockagenttypes.KnowledgeBaseConfiguration) bool {
	if config == nil {
		return false
	}
	switch config.Type {
	case bedrockagenttypes.KnowledgeBaseTypeVector:
		return config.VectorKnowledgeBaseConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseTypeKendra:
		return config.KendraKnowledgeBaseConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseTypeSql:
		return config.SqlKnowledgeBaseConfiguration != nil
	default:
		return false
	}
}

func bedrockAgentKnowledgeBaseStorageConfigurationImportable(config *bedrockagenttypes.KnowledgeBaseConfiguration, storage *bedrockagenttypes.StorageConfiguration) bool {
	if config == nil {
		return false
	}
	if config.Type != bedrockagenttypes.KnowledgeBaseTypeVector {
		return true
	}
	if storage == nil {
		return false
	}
	switch storage.Type {
	case bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchServerless:
		return storage.OpensearchServerlessConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypePinecone:
		return storage.PineconeConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeRedisEnterpriseCloud:
		return storage.RedisEnterpriseCloudConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeRds:
		return storage.RdsConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeMongoDbAtlas:
		return storage.MongoDbAtlasConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeNeptuneAnalytics:
		return storage.NeptuneAnalyticsConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchManagedCluster:
		return storage.OpensearchManagedClusterConfiguration != nil
	case bedrockagenttypes.KnowledgeBaseStorageTypeS3Vectors:
		return storage.S3VectorsConfiguration != nil
	default:
		return false
	}
}

func bedrockAgentDataSourceConfigurationImportable(config *bedrockagenttypes.DataSourceConfiguration) bool {
	if config == nil {
		return false
	}
	switch config.Type {
	case bedrockagenttypes.DataSourceTypeS3:
		return config.S3Configuration != nil
	case bedrockagenttypes.DataSourceTypeWeb:
		return config.WebConfiguration != nil
	case bedrockagenttypes.DataSourceTypeConfluence:
		return config.ConfluenceConfiguration != nil
	case bedrockagenttypes.DataSourceTypeSalesforce:
		return config.SalesforceConfiguration != nil
	case bedrockagenttypes.DataSourceTypeSharepoint:
		return config.SharePointConfiguration != nil
	case bedrockagenttypes.DataSourceTypeCustom, bedrockagenttypes.DataSourceTypeRedshiftMetadata:
		return true
	default:
		return false
	}
}

func bedrockAgentResourceNotFound(err error) bool {
	var notFound *bedrockagenttypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
