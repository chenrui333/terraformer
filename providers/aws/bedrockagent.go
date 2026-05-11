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
	bedrockAgentAgentActionGroupResourceType              = "aws_bedrockagent_agent_action_group"
	bedrockAgentAgentResourceType                         = "aws_bedrockagent_agent"
	bedrockAgentAgentAliasResourceType                    = "aws_bedrockagent_agent_alias"
	bedrockAgentAgentCollaboratorResourceType             = "aws_bedrockagent_agent_collaborator"
	bedrockAgentAgentKnowledgeBaseAssociationResourceType = "aws_bedrockagent_agent_knowledge_base_association"
	bedrockAgentDataSourceResourceType                    = "aws_bedrockagent_data_source"
	bedrockAgentFlowResourceType                          = "aws_bedrockagent_flow"
	bedrockAgentKnowledgeBaseResourceType                 = "aws_bedrockagent_knowledge_base"
	bedrockAgentPromptResourceType                        = "aws_bedrockagent_prompt"
	bedrockAgentDraftVersion                              = "DRAFT"
	bedrockAgentImportIDSeparator                         = ","
	bedrockAgentResourceNameFallback                      = "bedrockagent-resource"
)

var (
	bedrockAgentAllowEmptyValues = []string{"tags."}
	bedrockAgentResourceTypes    = []string{
		bedrockAgentServiceName(bedrockAgentAgentActionGroupResourceType),
		bedrockAgentServiceName(bedrockAgentAgentResourceType),
		bedrockAgentServiceName(bedrockAgentAgentAliasResourceType),
		bedrockAgentServiceName(bedrockAgentAgentCollaboratorResourceType),
		bedrockAgentServiceName(bedrockAgentAgentKnowledgeBaseAssociationResourceType),
		bedrockAgentServiceName(bedrockAgentDataSourceResourceType),
		bedrockAgentServiceName(bedrockAgentFlowResourceType),
		bedrockAgentServiceName(bedrockAgentKnowledgeBaseResourceType),
		bedrockAgentServiceName(bedrockAgentPromptResourceType),
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
	loadAgentActionGroups := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentActionGroupResourceType))
	loadAgentAliases := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentAliasResourceType))
	loadAgentCollaborators := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentCollaboratorResourceType))
	loadAgentKnowledgeBaseAssociations := g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentAgentKnowledgeBaseAssociationResourceType))
	if loadAgents || loadAgentActionGroups || loadAgentAliases || loadAgentCollaborators || loadAgentKnowledgeBaseAssociations {
		agents, err := listBedrockAgentAgents(svc)
		if err != nil {
			return err
		}
		if loadAgents {
			g.loadAgents(agents)
		}
		if loadAgentActionGroups {
			if err := g.loadAgentActionGroups(svc, agents); err != nil {
				return err
			}
		}
		if loadAgentAliases {
			if err := g.loadAgentAliases(svc, agents); err != nil {
				return err
			}
		}
		if loadAgentCollaborators {
			if err := g.loadAgentCollaborators(svc, agents); err != nil {
				return err
			}
		}
		if loadAgentKnowledgeBaseAssociations {
			if err := g.loadAgentKnowledgeBaseAssociations(svc, agents); err != nil {
				return err
			}
		}
	}

	if g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentFlowResourceType)) {
		flows, err := listBedrockAgentFlows(svc)
		if err != nil {
			return err
		}
		if err := g.loadFlows(svc, flows); err != nil {
			return err
		}
	}

	if g.shouldLoadBedrockAgentResource(bedrockAgentServiceName(bedrockAgentPromptResourceType)) {
		prompts, err := listBedrockAgentPrompts(svc)
		if err != nil {
			return err
		}
		if err := g.loadPrompts(svc, prompts); err != nil {
			return err
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

func listBedrockAgentFlows(svc *bedrockagent.Client) ([]bedrockagenttypes.FlowSummary, error) {
	p := bedrockagent.NewListFlowsPaginator(svc, &bedrockagent.ListFlowsInput{})
	flows := []bedrockagenttypes.FlowSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		flows = append(flows, page.FlowSummaries...)
	}
	return flows, nil
}

func listBedrockAgentPrompts(svc *bedrockagent.Client) ([]bedrockagenttypes.PromptSummary, error) {
	p := bedrockagent.NewListPromptsPaginator(svc, &bedrockagent.ListPromptsInput{})
	prompts := []bedrockagenttypes.PromptSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, page.PromptSummaries...)
	}
	return prompts, nil
}

func (g *BedrockAgentGenerator) loadAgents(agents []bedrockagenttypes.AgentSummary) {
	for _, agent := range agents {
		if resource, ok := newBedrockAgentAgentResource(agent); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *BedrockAgentGenerator) loadAgentActionGroups(svc *bedrockagent.Client, agents []bedrockagenttypes.AgentSummary) error {
	for _, agent := range agents {
		agentID := StringValue(agent.AgentId)
		if agentID == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
			continue
		}
		actionGroups, err := listBedrockAgentActionGroups(svc, agentID, bedrockAgentDraftVersion)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, actionGroup := range actionGroups {
			actionGroupID := StringValue(actionGroup.ActionGroupId)
			if actionGroupID == "" || !bedrockAgentActionGroupImportable(actionGroup.ActionGroupState) {
				continue
			}
			agentActionGroup, err := getBedrockAgentActionGroup(svc, actionGroupID, agentID, bedrockAgentDraftVersion)
			if err != nil {
				if bedrockAgentResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newBedrockAgentAgentActionGroupResource(agentActionGroup, agent.AgentStatus); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listBedrockAgentActionGroups(svc *bedrockagent.Client, agentID, agentVersion string) ([]bedrockagenttypes.ActionGroupSummary, error) {
	p := bedrockagent.NewListAgentActionGroupsPaginator(svc, &bedrockagent.ListAgentActionGroupsInput{
		AgentId:      &agentID,
		AgentVersion: &agentVersion,
	})
	actionGroups := []bedrockagenttypes.ActionGroupSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		actionGroups = append(actionGroups, page.ActionGroupSummaries...)
	}
	return actionGroups, nil
}

func getBedrockAgentActionGroup(svc *bedrockagent.Client, actionGroupID, agentID, agentVersion string) (*bedrockagenttypes.AgentActionGroup, error) {
	output, err := svc.GetAgentActionGroup(context.TODO(), &bedrockagent.GetAgentActionGroupInput{
		ActionGroupId: &actionGroupID,
		AgentId:       &agentID,
		AgentVersion:  &agentVersion,
	})
	if err != nil {
		return nil, err
	}
	return output.AgentActionGroup, nil
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

func (g *BedrockAgentGenerator) loadAgentCollaborators(svc *bedrockagent.Client, agents []bedrockagenttypes.AgentSummary) error {
	for _, agent := range agents {
		agentID := StringValue(agent.AgentId)
		if agentID == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
			continue
		}
		collaborators, err := listBedrockAgentCollaborators(svc, agentID, bedrockAgentDraftVersion)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		for _, collaborator := range collaborators {
			collaboratorID := StringValue(collaborator.CollaboratorId)
			if collaboratorID == "" {
				continue
			}
			agentCollaborator, err := getBedrockAgentCollaborator(svc, agentID, bedrockAgentDraftVersion, collaboratorID)
			if err != nil {
				if bedrockAgentResourceNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newBedrockAgentAgentCollaboratorResource(agentCollaborator, agent.AgentStatus); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listBedrockAgentCollaborators(svc *bedrockagent.Client, agentID, agentVersion string) ([]bedrockagenttypes.AgentCollaboratorSummary, error) {
	p := bedrockagent.NewListAgentCollaboratorsPaginator(svc, &bedrockagent.ListAgentCollaboratorsInput{
		AgentId:      &agentID,
		AgentVersion: &agentVersion,
	})
	collaborators := []bedrockagenttypes.AgentCollaboratorSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		collaborators = append(collaborators, page.AgentCollaboratorSummaries...)
	}
	return collaborators, nil
}

func getBedrockAgentCollaborator(svc *bedrockagent.Client, agentID, agentVersion, collaboratorID string) (*bedrockagenttypes.AgentCollaborator, error) {
	output, err := svc.GetAgentCollaborator(context.TODO(), &bedrockagent.GetAgentCollaboratorInput{
		AgentId:        &agentID,
		AgentVersion:   &agentVersion,
		CollaboratorId: &collaboratorID,
	})
	if err != nil {
		return nil, err
	}
	return output.AgentCollaborator, nil
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

func (g *BedrockAgentGenerator) loadFlows(svc *bedrockagent.Client, flows []bedrockagenttypes.FlowSummary) error {
	for _, flow := range flows {
		flowID := StringValue(flow.Id)
		if flowID == "" || !bedrockAgentFlowImportable(flow.Status) {
			continue
		}
		flowOutput, err := getBedrockAgentFlow(svc, flowID)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		if resource, ok := newBedrockAgentFlowResource(flowOutput); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func getBedrockAgentFlow(svc *bedrockagent.Client, flowID string) (*bedrockagent.GetFlowOutput, error) {
	output, err := svc.GetFlow(context.TODO(), &bedrockagent.GetFlowInput{
		FlowIdentifier: &flowID,
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (g *BedrockAgentGenerator) loadPrompts(svc *bedrockagent.Client, prompts []bedrockagenttypes.PromptSummary) error {
	for _, prompt := range prompts {
		promptID := StringValue(prompt.Id)
		if promptID == "" {
			continue
		}
		promptOutput, err := getBedrockAgentPrompt(svc, promptID)
		if err != nil {
			if bedrockAgentResourceNotFound(err) {
				continue
			}
			return err
		}
		if resource, ok := newBedrockAgentPromptResource(promptOutput); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func getBedrockAgentPrompt(svc *bedrockagent.Client, promptID string) (*bedrockagent.GetPromptOutput, error) {
	output, err := svc.GetPrompt(context.TODO(), &bedrockagent.GetPromptInput{
		PromptIdentifier: &promptID,
	})
	if err != nil {
		return nil, err
	}
	return output, nil
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

func newBedrockAgentAgentActionGroupResource(actionGroup *bedrockagenttypes.AgentActionGroup, agentStatus bedrockagenttypes.AgentStatus) (terraformutils.Resource, bool) {
	if actionGroup == nil {
		return terraformutils.Resource{}, false
	}
	actionGroupID := StringValue(actionGroup.ActionGroupId)
	actionGroupName := StringValue(actionGroup.ActionGroupName)
	agentID := StringValue(actionGroup.AgentId)
	agentVersion := StringValue(actionGroup.AgentVersion)
	if actionGroupID == "" ||
		actionGroupName == "" ||
		agentID == "" ||
		agentVersion == "" ||
		!bedrockAgentActionGroupImportable(actionGroup.ActionGroupState) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"action_group_id":    actionGroupID,
		"action_group_name":  actionGroupName,
		"action_group_state": string(actionGroup.ActionGroupState),
		"agent_id":           agentID,
		"agent_version":      agentVersion,
	}
	bedrockAgentAddStringAttribute(attributes, "description", actionGroup.Description)
	if actionGroup.ParentActionSignature != "" {
		attributes["parent_action_group_signature"] = string(actionGroup.ParentActionSignature)
	}
	bedrockAgentAddPrepareAgentAttribute(attributes, agentStatus)
	return terraformutils.NewResource(
		bedrockAgentAgentActionGroupImportID(actionGroupID, agentID, agentVersion),
		bedrockAgentResourceName("agent-action-group", agentID, agentVersion, actionGroupName, actionGroupID),
		bedrockAgentAgentActionGroupResourceType,
		"aws",
		attributes,
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newBedrockAgentAgentCollaboratorResource(collaborator *bedrockagenttypes.AgentCollaborator, agentStatus bedrockagenttypes.AgentStatus) (terraformutils.Resource, bool) {
	if collaborator == nil || collaborator.AgentDescriptor == nil {
		return terraformutils.Resource{}, false
	}
	agentID := StringValue(collaborator.AgentId)
	agentVersion := StringValue(collaborator.AgentVersion)
	collaboratorID := StringValue(collaborator.CollaboratorId)
	collaboratorName := StringValue(collaborator.CollaboratorName)
	collaborationInstruction := StringValue(collaborator.CollaborationInstruction)
	aliasARN := StringValue(collaborator.AgentDescriptor.AliasArn)
	if agentID == "" ||
		agentVersion == "" ||
		collaboratorID == "" ||
		collaboratorName == "" ||
		collaborationInstruction == "" ||
		aliasARN == "" ||
		!bedrockAgentRelayConversationHistoryImportable(collaborator.RelayConversationHistory) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"agent_descriptor.0.alias_arn": aliasARN,
		"agent_id":                     agentID,
		"agent_version":                agentVersion,
		"collaboration_instruction":    collaborationInstruction,
		"collaborator_id":              collaboratorID,
		"collaborator_name":            collaboratorName,
	}
	if collaborator.RelayConversationHistory != "" {
		attributes["relay_conversation_history"] = string(collaborator.RelayConversationHistory)
	}
	bedrockAgentAddPrepareAgentAttribute(attributes, agentStatus)
	return terraformutils.NewResource(
		bedrockAgentAgentCollaboratorImportID(agentID, agentVersion, collaboratorID),
		bedrockAgentResourceName("agent-collaborator", agentID, agentVersion, collaboratorName, collaboratorID),
		bedrockAgentAgentCollaboratorResourceType,
		"aws",
		attributes,
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

func newBedrockAgentFlowResource(flow *bedrockagent.GetFlowOutput) (terraformutils.Resource, bool) {
	if flow == nil {
		return terraformutils.Resource{}, false
	}
	flowID := StringValue(flow.Id)
	flowName := StringValue(flow.Name)
	executionRoleARN := StringValue(flow.ExecutionRoleArn)
	if flowID == "" ||
		flowName == "" ||
		executionRoleARN == "" ||
		!bedrockAgentFlowImportable(flow.Status) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"execution_role_arn": executionRoleARN,
		"id":                 flowID,
		"name":               flowName,
		"status":             string(flow.Status),
	}
	bedrockAgentAddStringAttribute(attributes, "customer_encryption_key_arn", flow.CustomerEncryptionKeyArn)
	bedrockAgentAddStringAttribute(attributes, "description", flow.Description)
	bedrockAgentAddStringAttribute(attributes, "version", flow.Version)
	return terraformutils.NewResource(
		bedrockAgentFlowImportID(flowID),
		bedrockAgentResourceName("flow", flowName, flowID),
		bedrockAgentFlowResourceType,
		"aws",
		attributes,
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

func newBedrockAgentPromptResource(prompt *bedrockagent.GetPromptOutput) (terraformutils.Resource, bool) {
	if prompt == nil {
		return terraformutils.Resource{}, false
	}
	promptID := StringValue(prompt.Id)
	promptName := StringValue(prompt.Name)
	if promptID == "" || promptName == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"id":   promptID,
		"name": promptName,
	}
	bedrockAgentAddStringAttribute(attributes, "customer_encryption_key_arn", prompt.CustomerEncryptionKeyArn)
	bedrockAgentAddStringAttribute(attributes, "default_variant", prompt.DefaultVariant)
	bedrockAgentAddStringAttribute(attributes, "description", prompt.Description)
	bedrockAgentAddStringAttribute(attributes, "version", prompt.Version)
	return terraformutils.NewResource(
		bedrockAgentPromptImportID(promptID),
		bedrockAgentResourceName("prompt", promptName, promptID),
		bedrockAgentPromptResourceType,
		"aws",
		attributes,
		bedrockAgentAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func bedrockAgentAgentAliasImportID(agentAliasID, agentID string) string {
	return agentAliasID + bedrockAgentImportIDSeparator + agentID
}

func bedrockAgentAgentActionGroupImportID(actionGroupID, agentID, agentVersion string) string {
	return actionGroupID + bedrockAgentImportIDSeparator + agentID + bedrockAgentImportIDSeparator + agentVersion
}

func bedrockAgentAgentCollaboratorImportID(agentID, agentVersion, collaboratorID string) string {
	return agentID + bedrockAgentImportIDSeparator + agentVersion + bedrockAgentImportIDSeparator + collaboratorID
}

func bedrockAgentAgentKnowledgeBaseAssociationImportID(agentID, agentVersion, knowledgeBaseID string) string {
	return agentID + bedrockAgentImportIDSeparator + agentVersion + bedrockAgentImportIDSeparator + knowledgeBaseID
}

func bedrockAgentDataSourceImportID(dataSourceID, knowledgeBaseID string) string {
	return dataSourceID + bedrockAgentImportIDSeparator + knowledgeBaseID
}

func bedrockAgentFlowImportID(flowID string) string {
	return flowID
}

func bedrockAgentPromptImportID(promptID string) string {
	return promptID
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

func bedrockAgentActionGroupImportable(state bedrockagenttypes.ActionGroupState) bool {
	return state == bedrockagenttypes.ActionGroupStateEnabled || state == bedrockagenttypes.ActionGroupStateDisabled
}

func bedrockAgentRelayConversationHistoryImportable(history bedrockagenttypes.RelayConversationHistory) bool {
	return history == "" ||
		history == bedrockagenttypes.RelayConversationHistoryToCollaborator ||
		history == bedrockagenttypes.RelayConversationHistoryDisabled
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

func bedrockAgentFlowImportable(status bedrockagenttypes.FlowStatus) bool {
	return status == bedrockagenttypes.FlowStatusPrepared || status == bedrockagenttypes.FlowStatusNotPrepared
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

func bedrockAgentAddStringAttribute(attributes map[string]string, name string, value *string) {
	if StringValue(value) != "" {
		attributes[name] = StringValue(value)
	}
}

func bedrockAgentAddPrepareAgentAttribute(attributes map[string]string, agentStatus bedrockagenttypes.AgentStatus) {
	if agentStatus == bedrockagenttypes.AgentStatusNotPrepared {
		attributes["prepare_agent"] = "false"
	}
}

func bedrockAgentResourceNotFound(err error) bool {
	var notFound *bedrockagenttypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
