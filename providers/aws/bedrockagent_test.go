// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	bedrockagenttypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestBedrockAgentAgentActionGroupImportID(t *testing.T) {
	got := bedrockAgentAgentActionGroupImportID("ACTGP12345", "GGRRAED6JP", bedrockAgentDraftVersion)
	want := "ACTGP12345,GGRRAED6JP,DRAFT"
	if got != want {
		t.Fatalf("bedrockAgentAgentActionGroupImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentAgentAliasImportID(t *testing.T) {
	got := bedrockAgentAgentAliasImportID("66IVY0GUTF", "GGRRAED6JP")
	want := "66IVY0GUTF,GGRRAED6JP"
	if got != want {
		t.Fatalf("bedrockAgentAgentAliasImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentAgentCollaboratorImportID(t *testing.T) {
	got := bedrockAgentAgentCollaboratorImportID("GGRRAED6JP", bedrockAgentDraftVersion, "COLLAB1234")
	want := "GGRRAED6JP,DRAFT,COLLAB1234"
	if got != want {
		t.Fatalf("bedrockAgentAgentCollaboratorImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentAgentKnowledgeBaseAssociationImportID(t *testing.T) {
	got := bedrockAgentAgentKnowledgeBaseAssociationImportID("GGRRAED6JP", bedrockAgentDraftVersion, "EMDPPAYPZI")
	want := "GGRRAED6JP,DRAFT,EMDPPAYPZI"
	if got != want {
		t.Fatalf("bedrockAgentAgentKnowledgeBaseAssociationImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentDataSourceImportID(t *testing.T) {
	got := bedrockAgentDataSourceImportID("GWCMFMQF6T", "EMDPPAYPZI")
	want := "GWCMFMQF6T,EMDPPAYPZI"
	if got != want {
		t.Fatalf("bedrockAgentDataSourceImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentFlowImportID(t *testing.T) {
	got := bedrockAgentFlowImportID("FLOW123456")
	want := "FLOW123456"
	if got != want {
		t.Fatalf("bedrockAgentFlowImportID() = %q, want %q", got, want)
	}
}

func TestBedrockAgentPromptImportID(t *testing.T) {
	got := bedrockAgentPromptImportID("PROMPT1234")
	want := "PROMPT1234"
	if got != want {
		t.Fatalf("bedrockAgentPromptImportID() = %q, want %q", got, want)
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
	actionGroupFirst := terraformutils.TfSanitize(bedrockAgentResourceName("agent-action-group", "agent-a", bedrockAgentDraftVersion, "lookup", "action-1"))
	actionGroupSecond := terraformutils.TfSanitize(bedrockAgentResourceName("agent-action-group", "agent-b", bedrockAgentDraftVersion, "lookup", "action-1"))
	if actionGroupFirst == actionGroupSecond {
		t.Fatalf("action group resource names should include parent agent identity: %q", actionGroupFirst)
	}
	collaboratorFirst := terraformutils.TfSanitize(bedrockAgentResourceName("agent-collaborator", "agent-a", bedrockAgentDraftVersion, "helper", "collab-1"))
	collaboratorSecond := terraformutils.TfSanitize(bedrockAgentResourceName("agent-collaborator", "agent-b", bedrockAgentDraftVersion, "helper", "collab-1"))
	if collaboratorFirst == collaboratorSecond {
		t.Fatalf("collaborator resource names should include parent agent identity: %q", collaboratorFirst)
	}
	dataSourceFirst := terraformutils.TfSanitize(bedrockAgentResourceName("data-source", "kb-a", "docs", "source-1"))
	dataSourceSecond := terraformutils.TfSanitize(bedrockAgentResourceName("data-source", "kb-b", "docs", "source-1"))
	if dataSourceFirst == dataSourceSecond {
		t.Fatalf("data source resource names should include parent knowledge base identity: %q", dataSourceFirst)
	}
	associationFirst := terraformutils.TfSanitize(bedrockAgentResourceName("agent-knowledge-base-association", "agent-a", bedrockAgentDraftVersion, "kb-1"))
	associationSecond := terraformutils.TfSanitize(bedrockAgentResourceName("agent-knowledge-base-association", "agent-b", bedrockAgentDraftVersion, "kb-1"))
	if associationFirst == associationSecond {
		t.Fatalf("association resource names should include parent agent identity: %q", associationFirst)
	}
	flowFirst := terraformutils.TfSanitize(bedrockAgentResourceName("flow", "support", "flow-a"))
	flowSecond := terraformutils.TfSanitize(bedrockAgentResourceName("flow", "support", "flow-b"))
	if flowFirst == flowSecond {
		t.Fatalf("flow resource names should include flow identity: %q", flowFirst)
	}
	promptFirst := terraformutils.TfSanitize(bedrockAgentResourceName("prompt", "support", "prompt-a"))
	promptSecond := terraformutils.TfSanitize(bedrockAgentResourceName("prompt", "support", "prompt-b"))
	if promptFirst == promptSecond {
		t.Fatalf("prompt resource names should include prompt identity: %q", promptFirst)
	}
}

func TestBedrockAgentShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := BedrockAgentGenerator{}
	for _, serviceName := range bedrockAgentResourceTypes {
		if !g.shouldLoadBedrockAgentResource(serviceName) {
			t.Fatalf("without typed filters, %s should be loaded", serviceName)
		}
	}

	for _, typedServiceName := range bedrockAgentResourceTypes {
		t.Run(typedServiceName, func(t *testing.T) {
			g.Filter = []terraformutils.ResourceFilter{{
				ServiceName:      typedServiceName,
				FieldPath:        "id",
				AcceptableValues: []string{"example-id"},
			}}
			for _, serviceName := range bedrockAgentResourceTypes {
				got := g.shouldLoadBedrockAgentResource(serviceName)
				want := serviceName == typedServiceName
				if got != want {
					t.Fatalf("shouldLoadBedrockAgentResource(%q) = %t, want %t for typed filter %q", serviceName, got, want, typedServiceName)
				}
			}
		})
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
			for _, serviceName := range bedrockAgentResourceTypes {
				if !g.shouldLoadBedrockAgentResource(serviceName) {
					t.Fatalf("untyped filter should keep %s discovery available", serviceName)
				}
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
	if _, ok := resource.InstanceState.Attributes["prepare_agent"]; ok {
		t.Fatal("prepared agent should not force prepare_agent")
	}

	unprepared, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("ABCDEFGHIJ"),
		AgentName:   aws.String("draft-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusNotPrepared,
	})
	assertBedrockAgentResource(t, unprepared, ok, "ABCDEFGHIJ", bedrockAgentAgentResourceType)
	if got := unprepared.InstanceState.Attributes["prepare_agent"]; got != "false" {
		t.Fatalf("prepare_agent attribute = %q, want false", got)
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

	dissociated, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("11AA22BB33"),
		AgentAliasName:   aws.String("offline"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusDissociated,
	})
	assertBedrockAgentResource(t, dissociated, ok, "11AA22BB33,GGRRAED6JP", bedrockAgentAgentAliasResourceType)

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

func TestNewBedrockAgentAgentActionGroupResource(t *testing.T) {
	resource, ok := newBedrockAgentAgentActionGroupResource(bedrockAgentTestAgentActionGroup(), bedrockagenttypes.AgentStatusPrepared)
	assertBedrockAgentResource(t, resource, ok, "ACTGP12345,GGRRAED6JP,DRAFT", bedrockAgentAgentActionGroupResourceType)
	if got := resource.InstanceState.Attributes["action_group_id"]; got != "ACTGP12345" {
		t.Fatalf("action_group_id attribute = %q, want ACTGP12345", got)
	}
	if got := resource.InstanceState.Attributes["action_group_name"]; got != "lookup-order" {
		t.Fatalf("action_group_name attribute = %q, want lookup-order", got)
	}
	if got := resource.InstanceState.Attributes["action_group_state"]; got != "ENABLED" {
		t.Fatalf("action_group_state attribute = %q, want ENABLED", got)
	}
	if got := resource.InstanceState.Attributes["agent_id"]; got != "GGRRAED6JP" {
		t.Fatalf("agent_id attribute = %q, want GGRRAED6JP", got)
	}
	if got := resource.InstanceState.Attributes["agent_version"]; got != bedrockAgentDraftVersion {
		t.Fatalf("agent_version attribute = %q, want %s", got, bedrockAgentDraftVersion)
	}
	if got := resource.InstanceState.Attributes["description"]; got != "order lookup" {
		t.Fatalf("description attribute = %q, want order lookup", got)
	}
	if _, ok := resource.InstanceState.Attributes["prepare_agent"]; ok {
		t.Fatal("prepared parent agent should not force prepare_agent")
	}

	unprepared, ok := newBedrockAgentAgentActionGroupResource(bedrockAgentTestAgentActionGroup(), bedrockagenttypes.AgentStatusNotPrepared)
	assertBedrockAgentResource(t, unprepared, ok, "ACTGP12345,GGRRAED6JP,DRAFT", bedrockAgentAgentActionGroupResourceType)
	if got := unprepared.InstanceState.Attributes["prepare_agent"]; got != "false" {
		t.Fatalf("prepare_agent attribute = %q, want false", got)
	}

	parentSignature := bedrockAgentTestAgentActionGroup()
	parentSignature.Description = nil
	parentSignature.ParentActionSignature = bedrockagenttypes.ActionGroupSignatureAmazonUserinput
	resource, ok = newBedrockAgentAgentActionGroupResource(parentSignature, bedrockagenttypes.AgentStatusPrepared)
	assertBedrockAgentResource(t, resource, ok, "ACTGP12345,GGRRAED6JP,DRAFT", bedrockAgentAgentActionGroupResourceType)
	if got := resource.InstanceState.Attributes["parent_action_group_signature"]; got != "AMAZON.UserInput" {
		t.Fatalf("parent_action_group_signature attribute = %q, want AMAZON.UserInput", got)
	}

	if _, ok := newBedrockAgentAgentActionGroupResource(nil, bedrockagenttypes.AgentStatusPrepared); ok {
		t.Fatal("nil action group should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagenttypes.AgentActionGroup){
		"action group ID":   func(actionGroup *bedrockagenttypes.AgentActionGroup) { actionGroup.ActionGroupId = nil },
		"action group name": func(actionGroup *bedrockagenttypes.AgentActionGroup) { actionGroup.ActionGroupName = nil },
		"agent ID":          func(actionGroup *bedrockagenttypes.AgentActionGroup) { actionGroup.AgentId = nil },
		"agent version":     func(actionGroup *bedrockagenttypes.AgentActionGroup) { actionGroup.AgentVersion = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			actionGroup := bedrockAgentTestAgentActionGroup()
			mutate(actionGroup)
			if _, ok := newBedrockAgentAgentActionGroupResource(actionGroup, bedrockagenttypes.AgentStatusPrepared); ok {
				t.Fatalf("action group without %s should be skipped", name)
			}
		})
	}

	unknownState := bedrockAgentTestAgentActionGroup()
	unknownState.ActionGroupState = ""
	if _, ok := newBedrockAgentAgentActionGroupResource(unknownState, bedrockagenttypes.AgentStatusPrepared); ok {
		t.Fatal("action group with empty state should be skipped")
	}
}

func TestNewBedrockAgentAgentCollaboratorResource(t *testing.T) {
	resource, ok := newBedrockAgentAgentCollaboratorResource(bedrockAgentTestAgentCollaborator(), bedrockagenttypes.AgentStatusPrepared)
	assertBedrockAgentResource(t, resource, ok, "GGRRAED6JP,DRAFT,COLLAB1234", bedrockAgentAgentCollaboratorResourceType)
	if got := resource.InstanceState.Attributes["agent_descriptor.0.alias_arn"]; got != "arn:aws:bedrock:us-east-1:123456789012:agent-alias/OTHERAGENT/ALIAS12345" {
		t.Fatalf("agent_descriptor.0.alias_arn attribute = %q, want collaborator alias ARN", got)
	}
	if got := resource.InstanceState.Attributes["agent_id"]; got != "GGRRAED6JP" {
		t.Fatalf("agent_id attribute = %q, want GGRRAED6JP", got)
	}
	if got := resource.InstanceState.Attributes["agent_version"]; got != bedrockAgentDraftVersion {
		t.Fatalf("agent_version attribute = %q, want %s", got, bedrockAgentDraftVersion)
	}
	if got := resource.InstanceState.Attributes["collaboration_instruction"]; got != "escalate billing questions" {
		t.Fatalf("collaboration_instruction attribute = %q, want escalate billing questions", got)
	}
	if got := resource.InstanceState.Attributes["collaborator_id"]; got != "COLLAB1234" {
		t.Fatalf("collaborator_id attribute = %q, want COLLAB1234", got)
	}
	if got := resource.InstanceState.Attributes["collaborator_name"]; got != "billing-helper" {
		t.Fatalf("collaborator_name attribute = %q, want billing-helper", got)
	}
	if got := resource.InstanceState.Attributes["relay_conversation_history"]; got != "TO_COLLABORATOR" {
		t.Fatalf("relay_conversation_history attribute = %q, want TO_COLLABORATOR", got)
	}
	if _, ok := resource.InstanceState.Attributes["prepare_agent"]; ok {
		t.Fatal("prepared parent agent should not force prepare_agent")
	}

	unprepared, ok := newBedrockAgentAgentCollaboratorResource(bedrockAgentTestAgentCollaborator(), bedrockagenttypes.AgentStatusNotPrepared)
	assertBedrockAgentResource(t, unprepared, ok, "GGRRAED6JP,DRAFT,COLLAB1234", bedrockAgentAgentCollaboratorResourceType)
	if got := unprepared.InstanceState.Attributes["prepare_agent"]; got != "false" {
		t.Fatalf("prepare_agent attribute = %q, want false", got)
	}

	if _, ok := newBedrockAgentAgentCollaboratorResource(nil, bedrockagenttypes.AgentStatusPrepared); ok {
		t.Fatal("nil collaborator should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagenttypes.AgentCollaborator){
		"agent descriptor":           func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.AgentDescriptor = nil },
		"agent descriptor alias ARN": func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.AgentDescriptor.AliasArn = nil },
		"agent ID":                   func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.AgentId = nil },
		"agent version":              func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.AgentVersion = nil },
		"collaborator ID":            func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.CollaboratorId = nil },
		"collaborator name":          func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.CollaboratorName = nil },
		"instruction":                func(collaborator *bedrockagenttypes.AgentCollaborator) { collaborator.CollaborationInstruction = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			collaborator := bedrockAgentTestAgentCollaborator()
			mutate(collaborator)
			if _, ok := newBedrockAgentAgentCollaboratorResource(collaborator, bedrockagenttypes.AgentStatusPrepared); ok {
				t.Fatalf("collaborator without %s should be skipped", name)
			}
		})
	}

	unknownHistory := bedrockAgentTestAgentCollaborator()
	unknownHistory.RelayConversationHistory = ""
	if _, ok := newBedrockAgentAgentCollaboratorResource(unknownHistory, bedrockagenttypes.AgentStatusPrepared); ok {
		t.Fatal("collaborator with empty relay conversation history should be skipped")
	}
}

func TestNewBedrockAgentAgentKnowledgeBaseAssociationResource(t *testing.T) {
	resource, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(bedrockAgentTestAgentKnowledgeBaseAssociation())
	assertBedrockAgentResource(t, resource, ok, "GGRRAED6JP,DRAFT,EMDPPAYPZI", bedrockAgentAgentKnowledgeBaseAssociationResourceType)
	if got := resource.InstanceState.Attributes["agent_id"]; got != "GGRRAED6JP" {
		t.Fatalf("agent_id attribute = %q, want GGRRAED6JP", got)
	}
	if got := resource.InstanceState.Attributes["agent_version"]; got != bedrockAgentDraftVersion {
		t.Fatalf("agent_version attribute = %q, want %s", got, bedrockAgentDraftVersion)
	}
	if got := resource.InstanceState.Attributes["knowledge_base_id"]; got != "EMDPPAYPZI" {
		t.Fatalf("knowledge_base_id attribute = %q, want EMDPPAYPZI", got)
	}
	if got := resource.InstanceState.Attributes["description"]; got != "customer knowledge" {
		t.Fatalf("description attribute = %q, want customer knowledge", got)
	}
	if got := resource.InstanceState.Attributes["knowledge_base_state"]; got != "ENABLED" {
		t.Fatalf("knowledge_base_state attribute = %q, want ENABLED", got)
	}

	disabled := bedrockAgentTestAgentKnowledgeBaseAssociation()
	disabled.KnowledgeBaseState = bedrockagenttypes.KnowledgeBaseStateDisabled
	resource, ok = newBedrockAgentAgentKnowledgeBaseAssociationResource(disabled)
	assertBedrockAgentResource(t, resource, ok, "GGRRAED6JP,DRAFT,EMDPPAYPZI", bedrockAgentAgentKnowledgeBaseAssociationResourceType)
	if got := resource.InstanceState.Attributes["knowledge_base_state"]; got != "DISABLED" {
		t.Fatalf("knowledge_base_state attribute = %q, want DISABLED", got)
	}

	if _, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(nil); ok {
		t.Fatal("nil association should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagenttypes.AgentKnowledgeBase){
		"agent ID":          func(association *bedrockagenttypes.AgentKnowledgeBase) { association.AgentId = nil },
		"agent version":     func(association *bedrockagenttypes.AgentKnowledgeBase) { association.AgentVersion = nil },
		"knowledge base ID": func(association *bedrockagenttypes.AgentKnowledgeBase) { association.KnowledgeBaseId = nil },
		"description":       func(association *bedrockagenttypes.AgentKnowledgeBase) { association.Description = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			association := bedrockAgentTestAgentKnowledgeBaseAssociation()
			mutate(association)
			if _, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(association); ok {
				t.Fatalf("association without %s should be skipped", name)
			}
		})
	}

	versioned := bedrockAgentTestAgentKnowledgeBaseAssociation()
	versioned.AgentVersion = aws.String("1")
	if _, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(versioned); ok {
		t.Fatal("non-DRAFT association should be skipped")
	}
	unknownState := bedrockAgentTestAgentKnowledgeBaseAssociation()
	unknownState.KnowledgeBaseState = ""
	if _, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(unknownState); ok {
		t.Fatal("association with empty state should be skipped")
	}
}

func TestNewBedrockAgentKnowledgeBaseResource(t *testing.T) {
	resource, ok := newBedrockAgentKnowledgeBaseResource(bedrockAgentTestKnowledgeBase())
	assertBedrockAgentResource(t, resource, ok, "EMDPPAYPZI", bedrockAgentKnowledgeBaseResourceType)
	if got := resource.InstanceState.Attributes["id"]; got != "EMDPPAYPZI" {
		t.Fatalf("id attribute = %q, want EMDPPAYPZI", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "customer-kb" {
		t.Fatalf("name attribute = %q, want customer-kb", got)
	}
	if got := resource.InstanceState.Attributes["role_arn"]; got != "arn:aws:iam::123456789012:role/bedrock-kb" {
		t.Fatalf("role_arn attribute = %q, want arn:aws:iam::123456789012:role/bedrock-kb", got)
	}

	if _, ok := newBedrockAgentKnowledgeBaseResource(nil); ok {
		t.Fatal("nil knowledge base should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagenttypes.KnowledgeBase){
		"ID":     func(knowledgeBase *bedrockagenttypes.KnowledgeBase) { knowledgeBase.KnowledgeBaseId = nil },
		"name":   func(knowledgeBase *bedrockagenttypes.KnowledgeBase) { knowledgeBase.Name = nil },
		"role":   func(knowledgeBase *bedrockagenttypes.KnowledgeBase) { knowledgeBase.RoleArn = nil },
		"config": func(knowledgeBase *bedrockagenttypes.KnowledgeBase) { knowledgeBase.KnowledgeBaseConfiguration = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			knowledgeBase := bedrockAgentTestKnowledgeBase()
			mutate(knowledgeBase)
			if _, ok := newBedrockAgentKnowledgeBaseResource(knowledgeBase); ok {
				t.Fatalf("knowledge base without %s should be skipped", name)
			}
		})
	}

	creating := bedrockAgentTestKnowledgeBase()
	creating.Status = bedrockagenttypes.KnowledgeBaseStatusCreating
	if _, ok := newBedrockAgentKnowledgeBaseResource(creating); ok {
		t.Fatal("creating knowledge base should be skipped")
	}
	vectorMissingConfig := bedrockAgentTestKnowledgeBase()
	vectorMissingConfig.KnowledgeBaseConfiguration.VectorKnowledgeBaseConfiguration = nil
	if _, ok := newBedrockAgentKnowledgeBaseResource(vectorMissingConfig); ok {
		t.Fatal("vector knowledge base without vector configuration should be skipped")
	}
	vectorMissingStorage := bedrockAgentTestKnowledgeBase()
	vectorMissingStorage.StorageConfiguration = nil
	if _, ok := newBedrockAgentKnowledgeBaseResource(vectorMissingStorage); ok {
		t.Fatal("vector knowledge base without storage configuration should be skipped")
	}
}

func TestNewBedrockAgentDataSourceResource(t *testing.T) {
	resource, ok := newBedrockAgentDataSourceResource(bedrockAgentTestDataSource())
	assertBedrockAgentResource(t, resource, ok, "GWCMFMQF6T,EMDPPAYPZI", bedrockAgentDataSourceResourceType)
	if got := resource.InstanceState.Attributes["data_source_id"]; got != "GWCMFMQF6T" {
		t.Fatalf("data_source_id attribute = %q, want GWCMFMQF6T", got)
	}
	if got := resource.InstanceState.Attributes["knowledge_base_id"]; got != "EMDPPAYPZI" {
		t.Fatalf("knowledge_base_id attribute = %q, want EMDPPAYPZI", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "support-docs" {
		t.Fatalf("name attribute = %q, want support-docs", got)
	}

	if _, ok := newBedrockAgentDataSourceResource(nil); ok {
		t.Fatal("nil data source should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagenttypes.DataSource){
		"ID":                func(dataSource *bedrockagenttypes.DataSource) { dataSource.DataSourceId = nil },
		"knowledge base ID": func(dataSource *bedrockagenttypes.DataSource) { dataSource.KnowledgeBaseId = nil },
		"name":              func(dataSource *bedrockagenttypes.DataSource) { dataSource.Name = nil },
		"config":            func(dataSource *bedrockagenttypes.DataSource) { dataSource.DataSourceConfiguration = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			dataSource := bedrockAgentTestDataSource()
			mutate(dataSource)
			if _, ok := newBedrockAgentDataSourceResource(dataSource); ok {
				t.Fatalf("data source without %s should be skipped", name)
			}
		})
	}

	deleting := bedrockAgentTestDataSource()
	deleting.Status = bedrockagenttypes.DataSourceStatusDeleting
	if _, ok := newBedrockAgentDataSourceResource(deleting); ok {
		t.Fatal("deleting data source should be skipped")
	}
	s3MissingConfig := bedrockAgentTestDataSource()
	s3MissingConfig.DataSourceConfiguration.S3Configuration = nil
	if _, ok := newBedrockAgentDataSourceResource(s3MissingConfig); ok {
		t.Fatal("S3 data source without S3 configuration should be skipped")
	}
}

func TestNewBedrockAgentFlowResource(t *testing.T) {
	resource, ok := newBedrockAgentFlowResource(bedrockAgentTestFlow())
	assertBedrockAgentResource(t, resource, ok, "FLOW123456", bedrockAgentFlowResourceType)
	if got := resource.InstanceState.Attributes["id"]; got != "FLOW123456" {
		t.Fatalf("id attribute = %q, want FLOW123456", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "support-flow" {
		t.Fatalf("name attribute = %q, want support-flow", got)
	}
	if got := resource.InstanceState.Attributes["execution_role_arn"]; got != "arn:aws:iam::123456789012:role/bedrock-flow" {
		t.Fatalf("execution_role_arn attribute = %q, want arn:aws:iam::123456789012:role/bedrock-flow", got)
	}
	if got := resource.InstanceState.Attributes["status"]; got != "Prepared" {
		t.Fatalf("status attribute = %q, want Prepared", got)
	}
	if got := resource.InstanceState.Attributes["description"]; got != "support workflow" {
		t.Fatalf("description attribute = %q, want support workflow", got)
	}
	if got := resource.InstanceState.Attributes["version"]; got != "DRAFT" {
		t.Fatalf("version attribute = %q, want DRAFT", got)
	}

	notPrepared := bedrockAgentTestFlow()
	notPrepared.Status = bedrockagenttypes.FlowStatusNotPrepared
	resource, ok = newBedrockAgentFlowResource(notPrepared)
	assertBedrockAgentResource(t, resource, ok, "FLOW123456", bedrockAgentFlowResourceType)
	if got := resource.InstanceState.Attributes["status"]; got != "NotPrepared" {
		t.Fatalf("status attribute = %q, want NotPrepared", got)
	}

	if _, ok := newBedrockAgentFlowResource(nil); ok {
		t.Fatal("nil flow should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagent.GetFlowOutput){
		"ID":                 func(flow *bedrockagent.GetFlowOutput) { flow.Id = nil },
		"name":               func(flow *bedrockagent.GetFlowOutput) { flow.Name = nil },
		"execution role ARN": func(flow *bedrockagent.GetFlowOutput) { flow.ExecutionRoleArn = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			flow := bedrockAgentTestFlow()
			mutate(flow)
			if _, ok := newBedrockAgentFlowResource(flow); ok {
				t.Fatalf("flow without %s should be skipped", name)
			}
		})
	}

	preparing := bedrockAgentTestFlow()
	preparing.Status = bedrockagenttypes.FlowStatusPreparing
	if _, ok := newBedrockAgentFlowResource(preparing); ok {
		t.Fatal("preparing flow should be skipped")
	}
	failed := bedrockAgentTestFlow()
	failed.Status = bedrockagenttypes.FlowStatusFailed
	if _, ok := newBedrockAgentFlowResource(failed); ok {
		t.Fatal("failed flow should be skipped")
	}
}

func TestNewBedrockAgentPromptResource(t *testing.T) {
	resource, ok := newBedrockAgentPromptResource(bedrockAgentTestPrompt())
	assertBedrockAgentResource(t, resource, ok, "PROMPT1234", bedrockAgentPromptResourceType)
	if got := resource.InstanceState.Attributes["id"]; got != "PROMPT1234" {
		t.Fatalf("id attribute = %q, want PROMPT1234", got)
	}
	if got := resource.InstanceState.Attributes["name"]; got != "support-prompt" {
		t.Fatalf("name attribute = %q, want support-prompt", got)
	}
	if got := resource.InstanceState.Attributes["default_variant"]; got != "primary" {
		t.Fatalf("default_variant attribute = %q, want primary", got)
	}
	if got := resource.InstanceState.Attributes["description"]; got != "support prompt" {
		t.Fatalf("description attribute = %q, want support prompt", got)
	}
	if got := resource.InstanceState.Attributes["version"]; got != "DRAFT" {
		t.Fatalf("version attribute = %q, want DRAFT", got)
	}

	if _, ok := newBedrockAgentPromptResource(nil); ok {
		t.Fatal("nil prompt should be skipped")
	}
	for name, mutate := range map[string]func(*bedrockagent.GetPromptOutput){
		"ID":   func(prompt *bedrockagent.GetPromptOutput) { prompt.Id = nil },
		"name": func(prompt *bedrockagent.GetPromptOutput) { prompt.Name = nil },
	} {
		t.Run("missing "+name, func(t *testing.T) {
			prompt := bedrockAgentTestPrompt()
			mutate(prompt)
			if _, ok := newBedrockAgentPromptResource(prompt); ok {
				t.Fatalf("prompt without %s should be skipped", name)
			}
		})
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

	for _, status := range []bedrockagenttypes.AgentAliasStatus{
		bedrockagenttypes.AgentAliasStatusPrepared,
		bedrockagenttypes.AgentAliasStatusDissociated,
	} {
		if !bedrockAgentAgentAliasImportable(status) {
			t.Fatalf("%s agent alias should be importable", status)
		}
	}
	for _, status := range []bedrockagenttypes.AgentAliasStatus{
		bedrockagenttypes.AgentAliasStatusCreating,
		bedrockagenttypes.AgentAliasStatusUpdating,
		bedrockagenttypes.AgentAliasStatusDeleting,
		bedrockagenttypes.AgentAliasStatusFailed,
	} {
		if bedrockAgentAgentAliasImportable(status) {
			t.Fatalf("%s agent alias should not be importable", status)
		}
	}

	for _, state := range []bedrockagenttypes.ActionGroupState{
		bedrockagenttypes.ActionGroupStateEnabled,
		bedrockagenttypes.ActionGroupStateDisabled,
	} {
		if !bedrockAgentActionGroupImportable(state) {
			t.Fatalf("%s action group should be importable", state)
		}
	}
	if bedrockAgentActionGroupImportable("") {
		t.Fatal("action group with empty state should not be importable")
	}

	for _, history := range []bedrockagenttypes.RelayConversationHistory{
		bedrockagenttypes.RelayConversationHistoryToCollaborator,
		bedrockagenttypes.RelayConversationHistoryDisabled,
	} {
		if !bedrockAgentRelayConversationHistoryImportable(history) {
			t.Fatalf("%s collaborator relay history should be importable", history)
		}
	}
	if bedrockAgentRelayConversationHistoryImportable("") {
		t.Fatal("collaborator with empty relay history should not be importable")
	}

	for _, state := range []bedrockagenttypes.KnowledgeBaseState{
		bedrockagenttypes.KnowledgeBaseStateEnabled,
		bedrockagenttypes.KnowledgeBaseStateDisabled,
	} {
		if !bedrockAgentAgentKnowledgeBaseAssociationImportable(bedrockAgentDraftVersion, state) {
			t.Fatalf("%s association should be importable for DRAFT agents", state)
		}
	}
	if bedrockAgentAgentKnowledgeBaseAssociationImportable("1", bedrockagenttypes.KnowledgeBaseStateEnabled) {
		t.Fatal("non-DRAFT association should not be importable")
	}
	if bedrockAgentAgentKnowledgeBaseAssociationImportable(bedrockAgentDraftVersion, "") {
		t.Fatal("association with empty state should not be importable")
	}

	if !bedrockAgentKnowledgeBaseImportable(bedrockagenttypes.KnowledgeBaseStatusActive) {
		t.Fatal("ACTIVE knowledge base should be importable")
	}
	for _, status := range []bedrockagenttypes.KnowledgeBaseStatus{
		bedrockagenttypes.KnowledgeBaseStatusCreating,
		bedrockagenttypes.KnowledgeBaseStatusUpdating,
		bedrockagenttypes.KnowledgeBaseStatusDeleting,
		bedrockagenttypes.KnowledgeBaseStatusFailed,
		bedrockagenttypes.KnowledgeBaseStatusDeleteUnsuccessful,
	} {
		if bedrockAgentKnowledgeBaseImportable(status) {
			t.Fatalf("%s knowledge base should not be importable", status)
		}
	}

	if !bedrockAgentDataSourceImportable(bedrockagenttypes.DataSourceStatusAvailable) {
		t.Fatal("AVAILABLE data source should be importable")
	}
	for _, status := range []bedrockagenttypes.DataSourceStatus{
		bedrockagenttypes.DataSourceStatusDeleting,
		bedrockagenttypes.DataSourceStatusDeleteUnsuccessful,
	} {
		if bedrockAgentDataSourceImportable(status) {
			t.Fatalf("%s data source should not be importable", status)
		}
	}

	for _, status := range []bedrockagenttypes.FlowStatus{
		bedrockagenttypes.FlowStatusPrepared,
		bedrockagenttypes.FlowStatusNotPrepared,
	} {
		if !bedrockAgentFlowImportable(status) {
			t.Fatalf("%s flow should be importable", status)
		}
	}
	for _, status := range []bedrockagenttypes.FlowStatus{
		bedrockagenttypes.FlowStatusPreparing,
		bedrockagenttypes.FlowStatusFailed,
	} {
		if bedrockAgentFlowImportable(status) {
			t.Fatalf("%s flow should not be importable", status)
		}
	}
}

func TestBedrockAgentKnowledgeBaseConfigurationImportable(t *testing.T) {
	tests := []struct {
		name   string
		config *bedrockagenttypes.KnowledgeBaseConfiguration
		want   bool
	}{
		{name: "nil", want: false},
		{
			name: "vector",
			config: &bedrockagenttypes.KnowledgeBaseConfiguration{
				Type:                             bedrockagenttypes.KnowledgeBaseTypeVector,
				VectorKnowledgeBaseConfiguration: &bedrockagenttypes.VectorKnowledgeBaseConfiguration{},
			},
			want: true,
		},
		{
			name: "vector missing config",
			config: &bedrockagenttypes.KnowledgeBaseConfiguration{
				Type: bedrockagenttypes.KnowledgeBaseTypeVector,
			},
			want: false,
		},
		{
			name: "kendra",
			config: &bedrockagenttypes.KnowledgeBaseConfiguration{
				Type:                             bedrockagenttypes.KnowledgeBaseTypeKendra,
				KendraKnowledgeBaseConfiguration: &bedrockagenttypes.KendraKnowledgeBaseConfiguration{},
			},
			want: true,
		},
		{
			name: "sql",
			config: &bedrockagenttypes.KnowledgeBaseConfiguration{
				Type:                          bedrockagenttypes.KnowledgeBaseTypeSql,
				SqlKnowledgeBaseConfiguration: &bedrockagenttypes.SqlKnowledgeBaseConfiguration{},
			},
			want: true,
		},
		{
			name: "unknown",
			config: &bedrockagenttypes.KnowledgeBaseConfiguration{
				Type: "UNKNOWN",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bedrockAgentKnowledgeBaseConfigurationImportable(tt.config); got != tt.want {
				t.Fatalf("bedrockAgentKnowledgeBaseConfigurationImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBedrockAgentKnowledgeBaseStorageConfigurationImportable(t *testing.T) {
	vectorConfig := &bedrockagenttypes.KnowledgeBaseConfiguration{
		Type:                             bedrockagenttypes.KnowledgeBaseTypeVector,
		VectorKnowledgeBaseConfiguration: &bedrockagenttypes.VectorKnowledgeBaseConfiguration{},
	}
	kendraConfig := &bedrockagenttypes.KnowledgeBaseConfiguration{
		Type:                             bedrockagenttypes.KnowledgeBaseTypeKendra,
		KendraKnowledgeBaseConfiguration: &bedrockagenttypes.KendraKnowledgeBaseConfiguration{},
	}
	tests := []struct {
		name    string
		config  *bedrockagenttypes.KnowledgeBaseConfiguration
		storage *bedrockagenttypes.StorageConfiguration
		want    bool
	}{
		{name: "nil config", want: false},
		{name: "Kendra no vector storage", config: kendraConfig, want: true},
		{name: "vector missing storage", config: vectorConfig, want: false},
		{
			name:   "OpenSearch Serverless",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                              bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchServerless,
				OpensearchServerlessConfiguration: &bedrockagenttypes.OpenSearchServerlessConfiguration{},
			},
			want: true,
		},
		{
			name:   "OpenSearch Serverless missing config",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type: bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchServerless,
			},
			want: false,
		},
		{
			name:   "Pinecone",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                  bedrockagenttypes.KnowledgeBaseStorageTypePinecone,
				PineconeConfiguration: &bedrockagenttypes.PineconeConfiguration{},
			},
			want: true,
		},
		{
			name:   "Redis Enterprise Cloud",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                              bedrockagenttypes.KnowledgeBaseStorageTypeRedisEnterpriseCloud,
				RedisEnterpriseCloudConfiguration: &bedrockagenttypes.RedisEnterpriseCloudConfiguration{},
			},
			want: true,
		},
		{
			name:   "RDS",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:             bedrockagenttypes.KnowledgeBaseStorageTypeRds,
				RdsConfiguration: &bedrockagenttypes.RdsConfiguration{},
			},
			want: true,
		},
		{
			name:   "MongoDB Atlas",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                      bedrockagenttypes.KnowledgeBaseStorageTypeMongoDbAtlas,
				MongoDbAtlasConfiguration: &bedrockagenttypes.MongoDbAtlasConfiguration{},
			},
			want: true,
		},
		{
			name:   "Neptune Analytics",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                          bedrockagenttypes.KnowledgeBaseStorageTypeNeptuneAnalytics,
				NeptuneAnalyticsConfiguration: &bedrockagenttypes.NeptuneAnalyticsConfiguration{},
			},
			want: true,
		},
		{
			name:   "OpenSearch Managed Cluster",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                                  bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchManagedCluster,
				OpensearchManagedClusterConfiguration: &bedrockagenttypes.OpenSearchManagedClusterConfiguration{},
			},
			want: true,
		},
		{
			name:   "S3 Vectors",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type:                   bedrockagenttypes.KnowledgeBaseStorageTypeS3Vectors,
				S3VectorsConfiguration: &bedrockagenttypes.S3VectorsConfiguration{},
			},
			want: true,
		},
		{
			name:   "unknown",
			config: vectorConfig,
			storage: &bedrockagenttypes.StorageConfiguration{
				Type: "UNKNOWN",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bedrockAgentKnowledgeBaseStorageConfigurationImportable(tt.config, tt.storage); got != tt.want {
				t.Fatalf("bedrockAgentKnowledgeBaseStorageConfigurationImportable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBedrockAgentDataSourceConfigurationImportable(t *testing.T) {
	tests := []struct {
		name   string
		config *bedrockagenttypes.DataSourceConfiguration
		want   bool
	}{
		{name: "nil", want: false},
		{
			name: "S3",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type:            bedrockagenttypes.DataSourceTypeS3,
				S3Configuration: &bedrockagenttypes.S3DataSourceConfiguration{},
			},
			want: true,
		},
		{
			name: "S3 missing config",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type: bedrockagenttypes.DataSourceTypeS3,
			},
			want: false,
		},
		{
			name: "web",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type:             bedrockagenttypes.DataSourceTypeWeb,
				WebConfiguration: &bedrockagenttypes.WebDataSourceConfiguration{},
			},
			want: true,
		},
		{
			name: "Confluence",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type:                    bedrockagenttypes.DataSourceTypeConfluence,
				ConfluenceConfiguration: &bedrockagenttypes.ConfluenceDataSourceConfiguration{},
			},
			want: true,
		},
		{
			name: "Salesforce",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type:                    bedrockagenttypes.DataSourceTypeSalesforce,
				SalesforceConfiguration: &bedrockagenttypes.SalesforceDataSourceConfiguration{},
			},
			want: true,
		},
		{
			name: "SharePoint",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type:                    bedrockagenttypes.DataSourceTypeSharepoint,
				SharePointConfiguration: &bedrockagenttypes.SharePointDataSourceConfiguration{},
			},
			want: true,
		},
		{
			name: "custom",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type: bedrockagenttypes.DataSourceTypeCustom,
			},
			want: true,
		},
		{
			name: "redshift metadata",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type: bedrockagenttypes.DataSourceTypeRedshiftMetadata,
			},
			want: true,
		},
		{
			name: "unknown",
			config: &bedrockagenttypes.DataSourceConfiguration{
				Type: "UNKNOWN",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bedrockAgentDataSourceConfigurationImportable(tt.config); got != tt.want {
				t.Fatalf("bedrockAgentDataSourceConfigurationImportable() = %t, want %t", got, tt.want)
			}
		})
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
	agent, actionGroup, alias, collaborator, association, dataSource, flow, knowledgeBase, prompt := bedrockAgentTestResources(t)
	g := BedrockAgentGenerator{}
	g.Resources = []terraformutils.Resource{agent, actionGroup, alias, collaborator, association, dataSource, flow, knowledgeBase, prompt}
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
	agent, actionGroup, alias, collaborator, association, dataSource, flow, knowledgeBase, prompt := bedrockAgentTestResources(t)
	g := BedrockAgentGenerator{}
	g.Resources = []terraformutils.Resource{agent, actionGroup, alias, collaborator, association, dataSource, flow, knowledgeBase, prompt}
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

	if len(g.Resources) != 9 {
		t.Fatalf("InitialCleanup() resources len = %d, want 9", len(g.Resources))
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

func bedrockAgentTestResources(t *testing.T) (terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource, terraformutils.Resource) {
	t.Helper()
	agent, ok := newBedrockAgentAgentResource(bedrockagenttypes.AgentSummary{
		AgentId:     aws.String("GGRRAED6JP"),
		AgentName:   aws.String("support-agent"),
		AgentStatus: bedrockagenttypes.AgentStatusPrepared,
	})
	if !ok {
		t.Fatal("newBedrockAgentAgentResource() should create agent")
	}
	actionGroup, ok := newBedrockAgentAgentActionGroupResource(bedrockAgentTestAgentActionGroup(), bedrockagenttypes.AgentStatusPrepared)
	if !ok {
		t.Fatal("newBedrockAgentAgentActionGroupResource() should create action group")
	}
	alias, ok := newBedrockAgentAgentAliasResource("GGRRAED6JP", bedrockagenttypes.AgentAliasSummary{
		AgentAliasId:     aws.String("66IVY0GUTF"),
		AgentAliasName:   aws.String("prod"),
		AgentAliasStatus: bedrockagenttypes.AgentAliasStatusPrepared,
	})
	if !ok {
		t.Fatal("newBedrockAgentAgentAliasResource() should create alias")
	}
	collaborator, ok := newBedrockAgentAgentCollaboratorResource(bedrockAgentTestAgentCollaborator(), bedrockagenttypes.AgentStatusPrepared)
	if !ok {
		t.Fatal("newBedrockAgentAgentCollaboratorResource() should create collaborator")
	}
	association, ok := newBedrockAgentAgentKnowledgeBaseAssociationResource(bedrockAgentTestAgentKnowledgeBaseAssociation())
	if !ok {
		t.Fatal("newBedrockAgentAgentKnowledgeBaseAssociationResource() should create association")
	}
	dataSource, ok := newBedrockAgentDataSourceResource(bedrockAgentTestDataSource())
	if !ok {
		t.Fatal("newBedrockAgentDataSourceResource() should create data source")
	}
	flow, ok := newBedrockAgentFlowResource(bedrockAgentTestFlow())
	if !ok {
		t.Fatal("newBedrockAgentFlowResource() should create flow")
	}
	knowledgeBase, ok := newBedrockAgentKnowledgeBaseResource(bedrockAgentTestKnowledgeBase())
	if !ok {
		t.Fatal("newBedrockAgentKnowledgeBaseResource() should create knowledge base")
	}
	prompt, ok := newBedrockAgentPromptResource(bedrockAgentTestPrompt())
	if !ok {
		t.Fatal("newBedrockAgentPromptResource() should create prompt")
	}
	return agent, actionGroup, alias, collaborator, association, dataSource, flow, knowledgeBase, prompt
}

func bedrockAgentTestAgentActionGroup() *bedrockagenttypes.AgentActionGroup {
	return &bedrockagenttypes.AgentActionGroup{
		ActionGroupId:    aws.String("ACTGP12345"),
		ActionGroupName:  aws.String("lookup-order"),
		ActionGroupState: bedrockagenttypes.ActionGroupStateEnabled,
		AgentId:          aws.String("GGRRAED6JP"),
		AgentVersion:     aws.String(bedrockAgentDraftVersion),
		Description:      aws.String("order lookup"),
	}
}

func bedrockAgentTestAgentCollaborator() *bedrockagenttypes.AgentCollaborator {
	return &bedrockagenttypes.AgentCollaborator{
		AgentDescriptor: &bedrockagenttypes.AgentDescriptor{
			AliasArn: aws.String("arn:aws:bedrock:us-east-1:123456789012:agent-alias/OTHERAGENT/ALIAS12345"),
		},
		AgentId:                  aws.String("GGRRAED6JP"),
		AgentVersion:             aws.String(bedrockAgentDraftVersion),
		CollaborationInstruction: aws.String("escalate billing questions"),
		CollaboratorId:           aws.String("COLLAB1234"),
		CollaboratorName:         aws.String("billing-helper"),
		RelayConversationHistory: bedrockagenttypes.RelayConversationHistoryToCollaborator,
	}
}

func bedrockAgentTestAgentKnowledgeBaseAssociation() *bedrockagenttypes.AgentKnowledgeBase {
	return &bedrockagenttypes.AgentKnowledgeBase{
		AgentId:            aws.String("GGRRAED6JP"),
		AgentVersion:       aws.String(bedrockAgentDraftVersion),
		Description:        aws.String("customer knowledge"),
		KnowledgeBaseId:    aws.String("EMDPPAYPZI"),
		KnowledgeBaseState: bedrockagenttypes.KnowledgeBaseStateEnabled,
	}
}

func bedrockAgentTestFlow() *bedrockagent.GetFlowOutput {
	return &bedrockagent.GetFlowOutput{
		Description:      aws.String("support workflow"),
		ExecutionRoleArn: aws.String("arn:aws:iam::123456789012:role/bedrock-flow"),
		Id:               aws.String("FLOW123456"),
		Name:             aws.String("support-flow"),
		Status:           bedrockagenttypes.FlowStatusPrepared,
		Version:          aws.String(bedrockAgentDraftVersion),
	}
}

func bedrockAgentTestKnowledgeBase() *bedrockagenttypes.KnowledgeBase {
	return &bedrockagenttypes.KnowledgeBase{
		KnowledgeBaseConfiguration: &bedrockagenttypes.KnowledgeBaseConfiguration{
			Type:                             bedrockagenttypes.KnowledgeBaseTypeVector,
			VectorKnowledgeBaseConfiguration: &bedrockagenttypes.VectorKnowledgeBaseConfiguration{},
		},
		KnowledgeBaseId: aws.String("EMDPPAYPZI"),
		Name:            aws.String("customer-kb"),
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/bedrock-kb"),
		StorageConfiguration: &bedrockagenttypes.StorageConfiguration{
			Type:                              bedrockagenttypes.KnowledgeBaseStorageTypeOpensearchServerless,
			OpensearchServerlessConfiguration: &bedrockagenttypes.OpenSearchServerlessConfiguration{},
		},
		Status: bedrockagenttypes.KnowledgeBaseStatusActive,
	}
}

func bedrockAgentTestPrompt() *bedrockagent.GetPromptOutput {
	return &bedrockagent.GetPromptOutput{
		DefaultVariant: aws.String("primary"),
		Description:    aws.String("support prompt"),
		Id:             aws.String("PROMPT1234"),
		Name:           aws.String("support-prompt"),
		Version:        aws.String(bedrockAgentDraftVersion),
	}
}

func bedrockAgentTestDataSource() *bedrockagenttypes.DataSource {
	return &bedrockagenttypes.DataSource{
		DataSourceConfiguration: &bedrockagenttypes.DataSourceConfiguration{
			Type:            bedrockagenttypes.DataSourceTypeS3,
			S3Configuration: &bedrockagenttypes.S3DataSourceConfiguration{},
		},
		DataSourceId:    aws.String("GWCMFMQF6T"),
		KnowledgeBaseId: aws.String("EMDPPAYPZI"),
		Name:            aws.String("support-docs"),
		Status:          bedrockagenttypes.DataSourceStatusAvailable,
	}
}
