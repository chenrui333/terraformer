// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrockagenttypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestBedrockAgentAgentAliasImportID(t *testing.T) {
	got := bedrockAgentAgentAliasImportID("66IVY0GUTF", "GGRRAED6JP")
	want := "66IVY0GUTF,GGRRAED6JP"
	if got != want {
		t.Fatalf("bedrockAgentAgentAliasImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentResourceNameFallback(t *testing.T) {
	if got := bedrockAgentResourceName("", ""); got != bedrockAgentResourceNameFallback {
		t.Fatalf("bedrockAgentResourceName() = %q, want %q", got, bedrockAgentResourceNameFallback)
	}
}

func TestBedrockAgentResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(bedrockAgentResourceName("ab", "c"))
	second := terraformutils.TfSanitize(bedrockAgentResourceName("a", "bc"))
	if first == second {
		t.Fatalf("bedrockAgentResourceName() collision after sanitize: %q", first)
	}
	aliasFirst := terraformutils.TfSanitize(bedrockAgentResourceName("agent-alias", "agent-a", "prod", "alias-1"))
	aliasSecond := terraformutils.TfSanitize(bedrockAgentResourceName("agent-alias", "agent-b", "prod", "alias-1"))
	if aliasFirst == aliasSecond {
		t.Fatalf("agent alias resource names should include parent agent identity: %q", aliasFirst)
	}
}

func TestBedrockAgentShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := BedrockAgentGenerator{}
	if !g.shouldLoadBedrockAgentResource("bedrockagent_agent") ||
		!g.shouldLoadBedrockAgentResource("bedrockagent_agent_alias") {
		t.Fatal("without typed filters, all Bedrock Agent resource families should be loaded")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "bedrockagent_agent",
		FieldPath:        "id",
		AcceptableValues: []string{"GGRRAED6JP"},
	}}
	if !g.shouldLoadBedrockAgentResource("bedrockagent_agent") {
		t.Fatal("typed agent filter should load agents")
	}
	if g.shouldLoadBedrockAgentResource("bedrockagent_agent_alias") {
		t.Fatal("typed agent filter should not load agent aliases")
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "bedrockagent_agent_alias",
		FieldPath:        "id",
		AcceptableValues: []string{"66IVY0GUTF,GGRRAED6JP"},
	}}
	if !g.shouldLoadBedrockAgentResource("bedrockagent_agent_alias") {
		t.Fatal("typed agent alias filter should load agent aliases")
	}
	if g.shouldLoadBedrockAgentResource("bedrockagent_agent") {
		t.Fatal("typed agent alias filter should not emit agent resources")
	}
}

func TestBedrockAgentShouldLoadResourceAllowsUntypedFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter terraformutils.ResourceFilter
	}{
		{
			name: "id",
			filter: terraformutils.ResourceFilter{
				FieldPath:        "id",
				AcceptableValues: []string{"66IVY0GUTF,GGRRAED6JP"},
			},
		},
		{
			name: "post-refresh attribute",
			filter: terraformutils.ResourceFilter{
				FieldPath:        "tags.env",
				AcceptableValues: []string{"prod"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := BedrockAgentGenerator{
				AWSService: AWSService{
					Service: terraformutils.Service{
						Filter: []terraformutils.ResourceFilter{
							{
								ServiceName:      "bedrockagent_agent",
								FieldPath:        "id",
								AcceptableValues: []string{"GGRRAED6JP"},
							},
							tt.filter,
						},
					},
				},
			}
			if !g.shouldLoadBedrockAgentResource("bedrockagent_agent_alias") {
				t.Fatal("untyped filter should keep agent alias discovery available")
			}
		})
	}
}

func TestNewBedrockAgentAgentResource(t *testing.T) {
	resource, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("GGRRAED6JP"),
		AgentName:   aws.String("support-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusPrepared,
	})
	assertBedrockAgentResource(t, resource, ok, "GGRRAED6JP", bedrockAgentAgentResourceType)
	if got := resource.InstanceState.Attributes["agent_id"]; got != "GGRRAED6JP" {
		t.Fatalf("agent_id attribute = %q, want GGRRAED6JP", got)
	}
	if got := resource.InstanceState.Attributes["agent_name"]; got != "support-agent" {
		t.Fatalf("agent_name attribute = %q, want support-agent", got)
	}

	if _, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentName:   aws.String("support-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusPrepared,
	}); ok {
		t.Fatal("agent without ID should be skipped")
	}
	if _, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("GGRRAED6JP"),
		AgentStatus: bedrockagenttypes.AgentStatusPrepared,
	}); ok {
		t.Fatal("agent without name should be skipped")
	}
	if _, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("GGRRAED6JP"),
		AgentName:   aws.String("support-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusCreating,
	}); ok {
		t.Fatal("creating agent should be skipped")
	}
}

func TestNewBedrockAgentAgentAliasResource(t *testing.T) {
	resource, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	})
	assertBedrockAgentResource(t, resource, ok, "66IVY0GUTF,GGRRAED6JP", bedrockAgentAgentAliasResourceType)
	if got := resource.InstanceState.Attributes["agent_alias_id"]; got != "66IVY0GUTF" {
		t.Fatalf("agent_alias_id attribute = %q, want 66IVY0GUTF", got)
	}
	if got := resource.InstanceState.Attributes["agent_alias_name"]; got != "prod" {
		t.Fatalf("agent_alias_name attribute = %q, want prod", got)
	}
	if got := resource.InstanceState.Attributes["agent_id"]; got != "GGRRAED6JP" {
		t.Fatalf("agent_id attribute = %q, want GGRRAED6JP", got)
	}

	if _, ok := newBedrockAgentAgentAliasResource("", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	}); ok {
		t.Fatal("alias without parent agent ID should be skipped")
	}
	if _, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	}); ok {
		t.Fatal("alias without ID should be skipped")
	}
	if _, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	}); ok {
		t.Fatal("alias without name should be skipped")
	}
	if _, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusFailed,
	}); ok {
		t.Fatal("failed alias should be skipped")
	}
}

func TestBedrockAgentImportableStatuses(t *testing.T) {
	for _, status := range []bedrockagenttypes.AgentStatus{
		bedrockagenttypes.AgentStatusPrepared,
		bedrockagenttypes.AgentStatusNotPrepared,
	} {
		if !bedrockAgentAgentImportable(status) {
			t.Fatalf("%s agent should be importable", status)
		}
	}
	for _, status := range []bedrockagenttypes.AgentStatus{
		bedrockagenttypes.AgentStatusCreating,
		bedrockagenttypes.AgentStatusPreparing,
		bedrockagenttypes.AgentStatusUpdating,
		bedrockagenttypes.AgentStatusVersioning,
		bedrockagenttypes.AgentStatusDeleting,
		bedrockagenttypes.AgentStatusFailed,
	} {
		if bedrockAgentAgentImportable(status) {
			t.Fatalf("%s agent should not be importable", status)
		}
	}

	if !bedrockAgentAgentAliasImportable(bedrockagenttypes.AgentAliasStatusPrepared) {
		t.Fatal("PREPARED agent alias should be importable")
	}
	for _, status := range []bedrockagenttypes.AgentAliasStatus{
		bedrockagenttypes.AgentAliasStatusCreating,
		bedrockagenttypes.AgentAliasStatusUpdating,
		bedrockagenttypes.AgentAliasStatusDeleting,
		bedrockagenttypes.AgentAliasStatusFailed,
		bedrockagenttypes.AgentAliasStatusDissociated,
	} {
		if bedrockAgentAgentAliasImportable(status) {
			t.Fatalf("%s agent alias should not be importable", status)
		}
	}
}

func TestBedrockAgentResourceNotFound(t *testing.T) {
	if !bedrockAgentResourceNotFound(&bedrockagenttypes.ResourceNotFoundException{}) {
		t.Fatal("ResourceNotFoundException should be detected")
	}
	if !bedrockAgentResourceNotFound(errors.Join(errors.New("lookup failed"), &bedrockagenttypes.ResourceNotFoundException{})) {
		t.Fatal("wrapped ResourceNotFoundException should be detected")
	}
	if bedrockAgentResourceNotFound(errors.New("other error")) {
		t.Fatal("non-not-found error should not be detected")
	}
}

func TestBedrockAgentInitialCleanupHonorsTypedFilters(t *testing.T) {
	agent, alias := bedrockAgentTestResources(t)
	g := BedrockAgentGenerator{}
	g.Resources = []terraformutils.Resource{agent, alias}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "bedrockagent_agent_alias",
		FieldPath:        "id",
		AcceptableValues: []string{"66IVY0GUTF,GGRRAED6JP"},
	}}

	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceInfo.Type; got != bedrockAgentAgentAliasResourceType {
		t.Fatalf("InitialCleanup() kept resource type = %q, want %s", got, bedrockAgentAgentAliasResourceType)
	}
}

func TestBedrockAgentInitialCleanupPreservesGlobalFilters(t *testing.T) {
	agent, alias := bedrockAgentTestResources(t)
	g := BedrockAgentGenerator{}
	g.Resources = []terraformutils.Resource{agent, alias}
	g.Filter = []terraformutils.ResourceFilter{
		{
			ServiceName:      "bedrockagent_agent",
			FieldPath:        "id",
			AcceptableValues: []string{"GGRRAED6JP"},
		},
		{
			FieldPath:        "tags.env",
			AcceptableValues: []string{"prod"},
		},
	}

	g.InitialCleanup()

	if len(g.Resources) != 2 {
		t.Fatalf("InitialCleanup() resources len = %d, want 2", len(g.Resources))
	}
}

func assertBedrockAgentResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantType string) {
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

func bedrockAgentTestResources(t *testing.T) (terraformutils.Resource, terraformutils.Resource) {
	t.Helper()
	agent, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("GGRRAED6JP"),
		AgentName:   aws.String("support-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusPrepared,
	})
	if !ok {
		t.Fatal("newBedrockAgentAgentResource() should create agent")
	}
	alias, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	})
	if !ok {
		t.Fatal("newBedrockAgentAgentAliasResource() should create alias")
	}
	return agent, alias
}
