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
	bedrockAgentAgentResourceType      = "aws_bedrockagent_agent"
	bedrockAgentAgentAliasResourceType = "aws_bedrockagent_agent_alias"
	bedrockAgentImportIDSeparator      = ","
	bedrockAgentResourceNameFallback   = "bedrockagent-resource"
)

var (
	bedrockAgentAllowEmptyValues = []string{"tags."}
	bedrockAgentResourceTypes    = []string{
		bedrockAgentServiceName(bedrockAgentAgentResourceType),
		bedrockAgentServiceName(bedrockAgentAgentAliasResourceType),
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
	if loadAgents || loadAgentAliases {
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

func newBedrockAgentAgentResource(agent bedrockagenttypes.AgentSummary) (terraformutils.Resource, bool) {
	agentID := StringValue(agent.AgentId)
	agentName := StringValue(agent.AgentName)
	if agentID == "" || agentName == "" || !bedrockAgentAgentImportable(agent.AgentStatus) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		agentID,
		bedrockAgentResourceName("agent", agentName, agentID),
		bedrockAgentAgentResourceType,
		"aws",
		map[string]string{
			"agent_id":   agentID,
			"agent_name": agentName,
		},
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

func bedrockAgentAgentAliasImportID(agentAliasID, agentID string) string {
	return agentAliasID + bedrockAgentImportIDSeparator + agentID
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
	return status == bedrockagenttypes.AgentAliasStatusPrepared
}

func bedrockAgentResourceNotFound(err error) bool {
	var notFound *bedrockagenttypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
