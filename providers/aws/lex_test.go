// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice"
	lexmodelstypes "github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice/types"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lexv2types "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestLexImportIDs(t *testing.T) {
	if got, want := lexBotAliasImportID("support", "prod"), "support:prod"; got != want {
		t.Fatalf("lexBotAliasImportID() = %q, want %q", got, want)
	}
	if got, want := lexV2BotLocaleImportID("en_US", "BOT123", "DRAFT"), "en_US,BOT123,DRAFT"; got != want {
		t.Fatalf("lexV2BotLocaleImportID() = %q, want %q", got, want)
	}
	if got, want := lexV2IntentImportID("INT123", "BOT123", "DRAFT", "en_US"), "INT123:BOT123:DRAFT:en_US"; got != want {
		t.Fatalf("lexV2IntentImportID() = %q, want %q", got, want)
	}
	if got, want := lexV2SlotImportID("BOT123", "DRAFT", "INT123", "en_US", "SLOT123"), "BOT123,DRAFT,INT123,en_US,SLOT123"; got != want {
		t.Fatalf("lexV2SlotImportID() = %q, want %q", got, want)
	}
	if got, want := lexV2SlotTypeImportID("BOT123", "DRAFT", "en_US", "TYPE123"), "BOT123,DRAFT,en_US,TYPE123"; got != want {
		t.Fatalf("lexV2SlotTypeImportID() = %q, want %q", got, want)
	}
}

func TestLexPagination(t *testing.T) {
	v1Client := &fakeLexGetBotsClient{
		pages: []*lexmodelbuildingservice.GetBotsOutput{
			{
				Bots:      []lexmodelstypes.BotMetadata{{Name: aws.String("support")}},
				NextToken: aws.String("next"),
			},
			{
				Bots: []lexmodelstypes.BotMetadata{{Name: aws.String("billing")}},
			},
		},
	}
	v1Bots, err := listLexBots(v1Client)
	if err != nil {
		t.Fatalf("listLexBots() error = %v", err)
	}
	if len(v1Bots) != 2 || v1Client.calls != 2 {
		t.Fatalf("listLexBots() len/calls = %d/%d, want 2/2", len(v1Bots), v1Client.calls)
	}

	v2Client := &fakeLexV2ListBotsClient{
		pages: []*lexmodelsv2.ListBotsOutput{
			{
				BotSummaries: []lexv2types.BotSummary{{BotId: aws.String("BOT123")}},
				NextToken:    aws.String("next"),
			},
			{
				BotSummaries: []lexv2types.BotSummary{{BotId: aws.String("BOT456")}},
			},
		},
	}
	v2Bots, err := listLexV2Bots(v2Client)
	if err != nil {
		t.Fatalf("listLexV2Bots() error = %v", err)
	}
	if len(v2Bots) != 2 || v2Client.calls != 2 {
		t.Fatalf("listLexV2Bots() len/calls = %d/%d, want 2/2", len(v2Bots), v2Client.calls)
	}
}

func TestNewLexResources(t *testing.T) {
	bot, ok := newLexBotResource(lexmodelstypes.BotMetadata{
		Name:   aws.String("support"),
		Status: lexmodelstypes.StatusReady,
	})
	assertLexResource(t, bot, ok, "support", lexBotResourceType)
	if got := bot.InstanceState.Attributes["create_version"]; got != "false" {
		t.Fatalf("create_version attribute = %q, want false", got)
	}

	alias, ok := newLexBotAliasResource("support", lexmodelstypes.BotAliasMetadata{Name: aws.String("prod")})
	assertLexResource(t, alias, ok, "support:prod", lexBotAliasResourceType)

	intent, ok := newLexIntentResource(lexmodelstypes.IntentMetadata{Name: aws.String("FallbackIntent")})
	assertLexResource(t, intent, ok, "FallbackIntent", lexIntentResourceType)

	slotType, ok := newLexSlotTypeResource(lexmodelstypes.SlotTypeMetadata{Name: aws.String("AccountType")})
	assertLexResource(t, slotType, ok, "AccountType", lexSlotTypeResourceType)

	if _, ok := newLexBotResource(lexmodelstypes.BotMetadata{
		Name:   aws.String("support"),
		Status: lexmodelstypes.StatusBuilding,
	}); ok {
		t.Fatal("building bot should be skipped")
	}
}

func TestNewLexV2Resources(t *testing.T) {
	bot, ok := newLexV2BotResource(lexv2types.BotSummary{
		BotId:     aws.String("BOT123"),
		BotName:   aws.String("support"),
		BotStatus: lexv2types.BotStatusAvailable,
	})
	assertLexResource(t, bot, ok, "BOT123", lexV2BotResourceType)

	locale, ok := newLexV2BotLocaleResource("BOT123", lexv2types.BotLocaleSummary{
		BotLocaleStatus: lexv2types.BotLocaleStatusBuilt,
		LocaleId:        aws.String("en_US"),
	})
	assertLexResource(t, locale, ok, "en_US,BOT123,DRAFT", lexV2BotLocaleResourceType)

	intent, ok := newLexV2IntentResource("BOT123", "en_US", lexv2types.IntentSummary{
		IntentId:   aws.String("INT123"),
		IntentName: aws.String("FallbackIntent"),
	})
	assertLexResource(t, intent, ok, "INT123:BOT123:DRAFT:en_US", lexV2IntentResourceType)

	slot, ok := newLexV2SlotResource("BOT123", "en_US", "INT123", lexv2types.SlotSummary{
		SlotId:   aws.String("SLOT123"),
		SlotName: aws.String("AccountType"),
	})
	assertLexResource(t, slot, ok, "BOT123,DRAFT,INT123,en_US,SLOT123", lexV2SlotResourceType)

	slotType, ok := newLexV2SlotTypeResource("BOT123", "en_US", lexv2types.SlotTypeSummary{
		SlotTypeCategory: lexv2types.SlotTypeCategoryCustom,
		SlotTypeId:       aws.String("TYPE123"),
		SlotTypeName:     aws.String("AccountType"),
	})
	assertLexResource(t, slotType, ok, "BOT123,DRAFT,en_US,TYPE123", lexV2SlotTypeResourceType)

	if _, ok := newLexV2BotResource(lexv2types.BotSummary{
		BotId:     aws.String("BOT123"),
		BotName:   aws.String("support"),
		BotStatus: lexv2types.BotStatusUpdating,
	}); ok {
		t.Fatal("updating bot should be skipped")
	}
	if _, ok := newLexV2SlotTypeResource("BOT123", "en_US", lexv2types.SlotTypeSummary{
		SlotTypeCategory: lexv2types.SlotTypeCategoryExternalGrammar,
		SlotTypeId:       aws.String("TYPE123"),
		SlotTypeName:     aws.String("GrammarType"),
	}); ok {
		t.Fatal("external grammar slot type should be skipped")
	}
}

func TestLexImportableStatuses(t *testing.T) {
	if !lexBotImportable(lexmodelstypes.StatusReadyBasicTesting) || lexBotImportable(lexmodelstypes.StatusBuilding) {
		t.Fatal("Lex bot importability should allow ready states only")
	}
	if !lexV2BotImportable(lexv2types.BotStatusInactive) || lexV2BotImportable(lexv2types.BotStatusVersioning) {
		t.Fatal("Lex v2 bot importability should allow stable bot states only")
	}
	if !lexV2BotLocaleImportable(lexv2types.BotLocaleStatusReadyExpressTesting) || lexV2BotLocaleImportable(lexv2types.BotLocaleStatusBuilding) {
		t.Fatal("Lex v2 locale importability should allow stable locale states only")
	}
}

func TestLexShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := LexGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Filter: []terraformutils.ResourceFilter{{
					ServiceName:      "lex_bot",
					FieldPath:        "id",
					AcceptableValues: []string{"support"},
				}},
			},
		},
	}
	for _, serviceName := range lexResourceTypes {
		got := g.shouldLoadLexResource(serviceName)
		want := serviceName == "lex_bot"
		if got != want {
			t.Fatalf("shouldLoadLexResource(%q) = %t, want %t", serviceName, got, want)
		}
	}

	v2 := LexV2ModelsGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Filter: []terraformutils.ResourceFilter{{
					ServiceName:      "lexv2models_intent",
					FieldPath:        "id",
					AcceptableValues: []string{"INT123:BOT123:DRAFT:en_US"},
				}},
			},
		},
	}
	for _, serviceName := range lexV2ResourceTypes {
		got := v2.shouldLoadLexV2Resource(serviceName)
		want := serviceName == "lexv2models_intent"
		if got != want {
			t.Fatalf("shouldLoadLexV2Resource(%q) = %t, want %t", serviceName, got, want)
		}
	}
}

func TestLexInitialCleanupHonorsTypedFilters(t *testing.T) {
	bot, ok := newLexBotResource(lexmodelstypes.BotMetadata{Name: aws.String("support"), Status: lexmodelstypes.StatusReady})
	assertLexResource(t, bot, ok, "support", lexBotResourceType)
	alias, ok := newLexBotAliasResource("support", lexmodelstypes.BotAliasMetadata{Name: aws.String("prod")})
	assertLexResource(t, alias, ok, "support:prod", lexBotAliasResourceType)
	billing, ok := newLexBotResource(lexmodelstypes.BotMetadata{Name: aws.String("billing"), Status: lexmodelstypes.StatusReady})
	assertLexResource(t, billing, ok, "billing", lexBotResourceType)

	g := LexGenerator{}
	g.Resources = []terraformutils.Resource{bot, alias, billing}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "lex_bot",
		FieldPath:        "name",
		AcceptableValues: []string{"support"},
	}}
	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceState.Attributes["name"]; got != "support" {
		t.Fatalf("InitialCleanup() kept resource name = %q, want support", got)
	}

	intent, ok := newLexV2IntentResource("BOT123", "en_US", lexv2types.IntentSummary{
		IntentId:   aws.String("INT123"),
		IntentName: aws.String("FallbackIntent"),
	})
	assertLexResource(t, intent, ok, "INT123:BOT123:DRAFT:en_US", lexV2IntentResourceType)
	otherIntent, ok := newLexV2IntentResource("BOT123", "en_US", lexv2types.IntentSummary{
		IntentId:   aws.String("INT456"),
		IntentName: aws.String("OrderFlowers"),
	})
	assertLexResource(t, otherIntent, ok, "INT456:BOT123:DRAFT:en_US", lexV2IntentResourceType)

	v2 := LexV2ModelsGenerator{}
	v2.Resources = []terraformutils.Resource{intent, otherIntent}
	v2.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "lexv2models_intent",
		FieldPath:        "name",
		AcceptableValues: []string{"FallbackIntent"},
	}}
	v2.InitialCleanup()

	if len(v2.Resources) != 1 {
		t.Fatalf("LexV2 InitialCleanup() resources len = %d, want 1", len(v2.Resources))
	}
	if got := v2.Resources[0].InstanceState.Attributes["name"]; got != "FallbackIntent" {
		t.Fatalf("LexV2 InitialCleanup() kept resource name = %q, want FallbackIntent", got)
	}
}

func TestLexLoadBotAliasesDoesNotGateOnBotStatus(t *testing.T) {
	g := LexGenerator{}
	client := &fakeLexGetBotAliasesClient{
		pagesByBotName: map[string][]*lexmodelbuildingservice.GetBotAliasesOutput{
			"support": {
				{
					BotAliases: []lexmodelstypes.BotAliasMetadata{{Name: aws.String("prod")}},
				},
			},
		},
	}
	err := g.loadBotAliases(client, []lexmodelstypes.BotMetadata{{
		Name:   aws.String("support"),
		Status: lexmodelstypes.StatusBuilding,
	}})
	if err != nil {
		t.Fatalf("loadBotAliases() error = %v", err)
	}
	if len(g.Resources) != 1 {
		t.Fatalf("loadBotAliases() resources len = %d, want 1", len(g.Resources))
	}
	assertLexResource(t, g.Resources[0], true, "support:prod", lexBotAliasResourceType)
}

func TestLexResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(lexResourceName("ab", "c"))
	second := terraformutils.TfSanitize(lexResourceName("a", "bc"))
	if first == second {
		t.Fatalf("lexResourceName() collision after sanitize: %q", first)
	}
	v2First := terraformutils.TfSanitize(lexV2ResourceName("intent", "BOT123", "en_US", "ab", "c"))
	v2Second := terraformutils.TfSanitize(lexV2ResourceName("intent", "BOT123", "en_US", "a", "bc"))
	if v2First == v2Second {
		t.Fatalf("lexV2ResourceName() collision after sanitize: %q", v2First)
	}
}

type fakeLexGetBotsClient struct {
	pages []*lexmodelbuildingservice.GetBotsOutput
	calls int
}

func (f *fakeLexGetBotsClient) GetBots(context.Context, *lexmodelbuildingservice.GetBotsInput, ...func(*lexmodelbuildingservice.Options)) (*lexmodelbuildingservice.GetBotsOutput, error) {
	if f.calls >= len(f.pages) {
		return &lexmodelbuildingservice.GetBotsOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	return page, nil
}

type fakeLexGetBotAliasesClient struct {
	pagesByBotName map[string][]*lexmodelbuildingservice.GetBotAliasesOutput
	calls          int
}

func (f *fakeLexGetBotAliasesClient) GetBotAliases(_ context.Context, input *lexmodelbuildingservice.GetBotAliasesInput, _ ...func(*lexmodelbuildingservice.Options)) (*lexmodelbuildingservice.GetBotAliasesOutput, error) {
	f.calls++
	pages := f.pagesByBotName[StringValue(input.BotName)]
	if f.calls > len(pages) {
		return &lexmodelbuildingservice.GetBotAliasesOutput{}, nil
	}
	return pages[f.calls-1], nil
}

type fakeLexV2ListBotsClient struct {
	pages []*lexmodelsv2.ListBotsOutput
	calls int
}

func (f *fakeLexV2ListBotsClient) ListBots(context.Context, *lexmodelsv2.ListBotsInput, ...func(*lexmodelsv2.Options)) (*lexmodelsv2.ListBotsOutput, error) {
	if f.calls >= len(f.pages) {
		return &lexmodelsv2.ListBotsOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	return page, nil
}

func assertLexResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantType string) {
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
