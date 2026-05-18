// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice"
	lexmodelstypes "github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice/types"
	"github.com/aws/aws-sdk-go-v2/service/lexmodelsv2"
	lexv2types "github.com/aws/aws-sdk-go-v2/service/lexmodelsv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	lexBotResourceType           = "aws_lex_bot"
	lexBotAliasResourceType      = "aws_lex_bot_alias"
	lexIntentResourceType        = "aws_lex_intent"
	lexSlotTypeResourceType      = "aws_lex_slot_type"
	lexBotAliasImportIDSeparator = ":"
	lexResourceNameFallback      = "lex-resource"
	lexV2BotResourceType         = "aws_lexv2models_bot"
	lexV2BotLocaleResourceType   = "aws_lexv2models_bot_locale"
	lexV2IntentResourceType      = "aws_lexv2models_intent"
	lexV2SlotResourceType        = "aws_lexv2models_slot"
	lexV2SlotTypeResourceType    = "aws_lexv2models_slot_type"
	lexV2DraftBotVersion         = "DRAFT"
	lexV2ImportIDSeparator       = ","
	lexV2IntentImportIDSeparator = ":"
	lexV2ResourceNameFallback    = "lexv2models-resource"
)

var (
	lexAllowEmptyValues = []string{"tags."}
	lexResourceTypes    = []string{
		lexServiceName(lexBotResourceType),
		lexServiceName(lexBotAliasResourceType),
		lexServiceName(lexIntentResourceType),
		lexServiceName(lexSlotTypeResourceType),
	}
	lexV2AllowEmptyValues = []string{"tags."}
	lexV2ResourceTypes    = []string{
		lexV2ServiceName(lexV2BotResourceType),
		lexV2ServiceName(lexV2BotLocaleResourceType),
		lexV2ServiceName(lexV2IntentResourceType),
		lexV2ServiceName(lexV2SlotResourceType),
		lexV2ServiceName(lexV2SlotTypeResourceType),
	}
)

type LexGenerator struct {
	AWSService
}

type LexV2ModelsGenerator struct {
	AWSService
}

func (g *LexGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := lexServiceName(resource.InstanceInfo.Type)
		if g.hasTypedLexFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			if filter.ServiceName != "" && filter.ServiceName != serviceName {
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

func (g *LexGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := lexmodelbuildingservice.NewFromConfig(config)

	loadBots := g.shouldLoadLexResource(lexServiceName(lexBotResourceType))
	loadAliases := g.shouldLoadLexResource(lexServiceName(lexBotAliasResourceType))
	if loadBots || loadAliases {
		bots, err := listLexBots(svc)
		if err != nil {
			return err
		}
		if loadBots {
			g.loadBots(bots)
		}
		if loadAliases {
			if err := g.loadBotAliases(svc, bots); err != nil {
				return err
			}
		}
	}
	if g.shouldLoadLexResource(lexServiceName(lexIntentResourceType)) {
		if err := g.loadIntents(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadLexResource(lexServiceName(lexSlotTypeResourceType)) {
		if err := g.loadSlotTypes(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *LexGenerator) shouldLoadLexResource(serviceName string) bool {
	if !g.hasTypedLexFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *LexGenerator) hasTypedLexFilter() bool {
	for _, serviceName := range lexResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *LexGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *LexGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func lexServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listLexBots(svc lexmodelbuildingservice.GetBotsAPIClient) ([]lexmodelstypes.BotMetadata, error) {
	p := lexmodelbuildingservice.NewGetBotsPaginator(svc, &lexmodelbuildingservice.GetBotsInput{})
	bots := []lexmodelstypes.BotMetadata{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		bots = append(bots, page.Bots...)
	}
	return bots, nil
}

func (g *LexGenerator) loadBots(bots []lexmodelstypes.BotMetadata) {
	for _, bot := range bots {
		if resource, ok := newLexBotResource(bot); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *LexGenerator) loadBotAliases(svc lexmodelbuildingservice.GetBotAliasesAPIClient, bots []lexmodelstypes.BotMetadata) error {
	for _, bot := range bots {
		botName := StringValue(bot.Name)
		if botName == "" {
			continue
		}
		aliases, err := listLexBotAliases(svc, botName)
		if err != nil {
			return err
		}
		for _, alias := range aliases {
			if resource, ok := newLexBotAliasResource(botName, alias); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listLexBotAliases(svc lexmodelbuildingservice.GetBotAliasesAPIClient, botName string) ([]lexmodelstypes.BotAliasMetadata, error) {
	p := lexmodelbuildingservice.NewGetBotAliasesPaginator(svc, &lexmodelbuildingservice.GetBotAliasesInput{BotName: &botName})
	aliases := []lexmodelstypes.BotAliasMetadata{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, page.BotAliases...)
	}
	return aliases, nil
}

func (g *LexGenerator) loadIntents(svc *lexmodelbuildingservice.Client) error {
	intents, err := listLexIntents(svc)
	if err != nil {
		return err
	}
	for _, intent := range intents {
		if resource, ok := newLexIntentResource(intent); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func listLexIntents(svc lexmodelbuildingservice.GetIntentsAPIClient) ([]lexmodelstypes.IntentMetadata, error) {
	p := lexmodelbuildingservice.NewGetIntentsPaginator(svc, &lexmodelbuildingservice.GetIntentsInput{})
	intents := []lexmodelstypes.IntentMetadata{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		intents = append(intents, page.Intents...)
	}
	return intents, nil
}

func (g *LexGenerator) loadSlotTypes(svc *lexmodelbuildingservice.Client) error {
	slotTypes, err := listLexSlotTypes(svc)
	if err != nil {
		return err
	}
	for _, slotType := range slotTypes {
		if resource, ok := newLexSlotTypeResource(slotType); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func listLexSlotTypes(svc lexmodelbuildingservice.GetSlotTypesAPIClient) ([]lexmodelstypes.SlotTypeMetadata, error) {
	p := lexmodelbuildingservice.NewGetSlotTypesPaginator(svc, &lexmodelbuildingservice.GetSlotTypesInput{})
	slotTypes := []lexmodelstypes.SlotTypeMetadata{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		slotTypes = append(slotTypes, page.SlotTypes...)
	}
	return slotTypes, nil
}

func newLexBotResource(bot lexmodelstypes.BotMetadata) (terraformutils.Resource, bool) {
	name := StringValue(bot.Name)
	if name == "" || !lexBotImportable(bot.Status) {
		return terraformutils.Resource{}, false
	}
	return lexResource(name, lexResourceName("bot", name), lexBotResourceType, map[string]string{
		"create_version": "false",
		"name":           name,
	})
}

func newLexBotAliasResource(botName string, alias lexmodelstypes.BotAliasMetadata) (terraformutils.Resource, bool) {
	aliasName := StringValue(alias.Name)
	if botName == "" || aliasName == "" {
		return terraformutils.Resource{}, false
	}
	return lexResource(lexBotAliasImportID(botName, aliasName), lexResourceName("bot-alias", botName, aliasName), lexBotAliasResourceType, map[string]string{
		"bot_name": botName,
		"name":     aliasName,
	})
}

func newLexIntentResource(intent lexmodelstypes.IntentMetadata) (terraformutils.Resource, bool) {
	name := StringValue(intent.Name)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return lexResource(name, lexResourceName("intent", name), lexIntentResourceType, map[string]string{
		"name": name,
	})
}

func newLexSlotTypeResource(slotType lexmodelstypes.SlotTypeMetadata) (terraformutils.Resource, bool) {
	name := StringValue(slotType.Name)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return lexResource(name, lexResourceName("slot-type", name), lexSlotTypeResourceType, map[string]string{
		"name": name,
	})
}

func lexResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" || resourceType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		lexAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func lexBotAliasImportID(botName, aliasName string) string {
	return strings.Join([]string{botName, aliasName}, lexBotAliasImportIDSeparator)
}

func lexResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return lexResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func lexBotImportable(status lexmodelstypes.Status) bool {
	return status == lexmodelstypes.StatusReady ||
		status == lexmodelstypes.StatusReadyBasicTesting ||
		status == lexmodelstypes.StatusNotBuilt
}

func (g *LexV2ModelsGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := lexV2ServiceName(resource.InstanceInfo.Type)
		if g.hasTypedLexV2Filter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			if filter.ServiceName != "" && filter.ServiceName != serviceName {
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

func (g *LexV2ModelsGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := lexmodelsv2.NewFromConfig(config)

	loadBots := g.shouldLoadLexV2Resource(lexV2ServiceName(lexV2BotResourceType))
	loadBotLocales := g.shouldLoadLexV2Resource(lexV2ServiceName(lexV2BotLocaleResourceType))
	loadIntents := g.shouldLoadLexV2Resource(lexV2ServiceName(lexV2IntentResourceType))
	loadSlots := g.shouldLoadLexV2Resource(lexV2ServiceName(lexV2SlotResourceType))
	loadSlotTypes := g.shouldLoadLexV2Resource(lexV2ServiceName(lexV2SlotTypeResourceType))
	if !loadBots && !loadBotLocales && !loadIntents && !loadSlots && !loadSlotTypes {
		return nil
	}

	bots, err := listLexV2Bots(svc)
	if err != nil {
		return err
	}
	if loadBots {
		g.loadLexV2Bots(bots)
	}
	if loadBotLocales || loadIntents || loadSlots || loadSlotTypes {
		return g.loadLexV2BotChildren(svc, bots, loadBotLocales, loadIntents, loadSlots, loadSlotTypes)
	}
	return nil
}

func (g *LexV2ModelsGenerator) shouldLoadLexV2Resource(serviceName string) bool {
	if !g.hasTypedLexV2Filter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *LexV2ModelsGenerator) hasTypedLexV2Filter() bool {
	for _, serviceName := range lexV2ResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *LexV2ModelsGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *LexV2ModelsGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func lexV2ServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listLexV2Bots(svc lexmodelsv2.ListBotsAPIClient) ([]lexv2types.BotSummary, error) {
	p := lexmodelsv2.NewListBotsPaginator(svc, &lexmodelsv2.ListBotsInput{})
	bots := []lexv2types.BotSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		bots = append(bots, page.BotSummaries...)
	}
	return bots, nil
}

func (g *LexV2ModelsGenerator) loadLexV2Bots(bots []lexv2types.BotSummary) {
	for _, bot := range bots {
		if resource, ok := newLexV2BotResource(bot); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *LexV2ModelsGenerator) loadLexV2BotChildren(svc *lexmodelsv2.Client, bots []lexv2types.BotSummary, loadBotLocales, loadIntents, loadSlots, loadSlotTypes bool) error {
	for _, bot := range bots {
		botID := StringValue(bot.BotId)
		if botID == "" || !lexV2BotImportable(bot.BotStatus) {
			continue
		}
		locales, err := listLexV2BotLocales(svc, botID, lexV2DraftBotVersion)
		if err != nil {
			return err
		}
		for _, locale := range locales {
			localeID := StringValue(locale.LocaleId)
			if localeID == "" || !lexV2BotLocaleImportable(locale.BotLocaleStatus) {
				continue
			}
			if loadBotLocales {
				if resource, ok := newLexV2BotLocaleResource(botID, locale); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
			if loadSlotTypes {
				slotTypes, err := listLexV2SlotTypes(svc, botID, lexV2DraftBotVersion, localeID)
				if err != nil {
					return err
				}
				for _, slotType := range slotTypes {
					if resource, ok := newLexV2SlotTypeResource(botID, localeID, slotType); ok {
						g.Resources = append(g.Resources, resource)
					}
				}
			}
			if loadIntents || loadSlots {
				intents, err := listLexV2Intents(svc, botID, lexV2DraftBotVersion, localeID)
				if err != nil {
					return err
				}
				if loadIntents {
					for _, intent := range intents {
						if resource, ok := newLexV2IntentResource(botID, localeID, intent); ok {
							g.Resources = append(g.Resources, resource)
						}
					}
				}
				if loadSlots {
					if err := g.loadLexV2Slots(svc, botID, localeID, intents); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func listLexV2BotLocales(svc lexmodelsv2.ListBotLocalesAPIClient, botID, botVersion string) ([]lexv2types.BotLocaleSummary, error) {
	p := lexmodelsv2.NewListBotLocalesPaginator(svc, &lexmodelsv2.ListBotLocalesInput{
		BotId:      &botID,
		BotVersion: &botVersion,
	})
	locales := []lexv2types.BotLocaleSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		locales = append(locales, page.BotLocaleSummaries...)
	}
	return locales, nil
}

func listLexV2Intents(svc lexmodelsv2.ListIntentsAPIClient, botID, botVersion, localeID string) ([]lexv2types.IntentSummary, error) {
	p := lexmodelsv2.NewListIntentsPaginator(svc, &lexmodelsv2.ListIntentsInput{
		BotId:      &botID,
		BotVersion: &botVersion,
		LocaleId:   &localeID,
	})
	intents := []lexv2types.IntentSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		intents = append(intents, page.IntentSummaries...)
	}
	return intents, nil
}

func (g *LexV2ModelsGenerator) loadLexV2Slots(svc *lexmodelsv2.Client, botID, localeID string, intents []lexv2types.IntentSummary) error {
	for _, intent := range intents {
		intentID := StringValue(intent.IntentId)
		if intentID == "" {
			continue
		}
		slots, err := listLexV2Slots(svc, botID, lexV2DraftBotVersion, localeID, intentID)
		if err != nil {
			return err
		}
		for _, slot := range slots {
			if resource, ok := newLexV2SlotResource(botID, localeID, intentID, slot); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func listLexV2Slots(svc lexmodelsv2.ListSlotsAPIClient, botID, botVersion, localeID, intentID string) ([]lexv2types.SlotSummary, error) {
	p := lexmodelsv2.NewListSlotsPaginator(svc, &lexmodelsv2.ListSlotsInput{
		BotId:      &botID,
		BotVersion: &botVersion,
		IntentId:   &intentID,
		LocaleId:   &localeID,
	})
	slots := []lexv2types.SlotSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		slots = append(slots, page.SlotSummaries...)
	}
	return slots, nil
}

func listLexV2SlotTypes(svc lexmodelsv2.ListSlotTypesAPIClient, botID, botVersion, localeID string) ([]lexv2types.SlotTypeSummary, error) {
	p := lexmodelsv2.NewListSlotTypesPaginator(svc, &lexmodelsv2.ListSlotTypesInput{
		BotId:      &botID,
		BotVersion: &botVersion,
		LocaleId:   &localeID,
	})
	slotTypes := []lexv2types.SlotTypeSummary{}
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		slotTypes = append(slotTypes, page.SlotTypeSummaries...)
	}
	return slotTypes, nil
}

func newLexV2BotResource(bot lexv2types.BotSummary) (terraformutils.Resource, bool) {
	botID := StringValue(bot.BotId)
	name := StringValue(bot.BotName)
	if botID == "" || name == "" || !lexV2BotImportable(bot.BotStatus) {
		return terraformutils.Resource{}, false
	}
	return lexV2Resource(botID, lexV2ResourceName("bot", name, botID), lexV2BotResourceType, map[string]string{
		"id":   botID,
		"name": name,
	})
}

func newLexV2BotLocaleResource(botID string, locale lexv2types.BotLocaleSummary) (terraformutils.Resource, bool) {
	localeID := StringValue(locale.LocaleId)
	if botID == "" || localeID == "" || !lexV2BotLocaleImportable(locale.BotLocaleStatus) {
		return terraformutils.Resource{}, false
	}
	return lexV2Resource(lexV2BotLocaleImportID(localeID, botID, lexV2DraftBotVersion), lexV2ResourceName("bot-locale", botID, localeID), lexV2BotLocaleResourceType, map[string]string{
		"bot_id":      botID,
		"bot_version": lexV2DraftBotVersion,
		"locale_id":   localeID,
	})
}

func newLexV2IntentResource(botID, localeID string, intent lexv2types.IntentSummary) (terraformutils.Resource, bool) {
	intentID := StringValue(intent.IntentId)
	name := StringValue(intent.IntentName)
	if botID == "" || localeID == "" || intentID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return lexV2Resource(lexV2IntentImportID(intentID, botID, lexV2DraftBotVersion, localeID), lexV2ResourceName("intent", botID, localeID, name, intentID), lexV2IntentResourceType, map[string]string{
		"bot_id":      botID,
		"bot_version": lexV2DraftBotVersion,
		"intent_id":   intentID,
		"locale_id":   localeID,
		"name":        name,
	})
}

func newLexV2SlotResource(botID, localeID, intentID string, slot lexv2types.SlotSummary) (terraformutils.Resource, bool) {
	slotID := StringValue(slot.SlotId)
	name := StringValue(slot.SlotName)
	if botID == "" || localeID == "" || intentID == "" || slotID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return lexV2Resource(lexV2SlotImportID(botID, lexV2DraftBotVersion, intentID, localeID, slotID), lexV2ResourceName("slot", botID, localeID, intentID, name, slotID), lexV2SlotResourceType, map[string]string{
		"bot_id":      botID,
		"bot_version": lexV2DraftBotVersion,
		"intent_id":   intentID,
		"locale_id":   localeID,
		"name":        name,
		"slot_id":     slotID,
	})
}

func newLexV2SlotTypeResource(botID, localeID string, slotType lexv2types.SlotTypeSummary) (terraformutils.Resource, bool) {
	slotTypeID := StringValue(slotType.SlotTypeId)
	name := StringValue(slotType.SlotTypeName)
	if botID == "" || localeID == "" || slotTypeID == "" || name == "" || !lexV2SlotTypeImportable(slotType.SlotTypeCategory) {
		return terraformutils.Resource{}, false
	}
	return lexV2Resource(lexV2SlotTypeImportID(botID, lexV2DraftBotVersion, localeID, slotTypeID), lexV2ResourceName("slot-type", botID, localeID, name, slotTypeID), lexV2SlotTypeResourceType, map[string]string{
		"bot_id":       botID,
		"bot_version":  lexV2DraftBotVersion,
		"locale_id":    localeID,
		"name":         name,
		"slot_type_id": slotTypeID,
	})
}

func lexV2Resource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" || resourceType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		lexV2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func lexV2BotLocaleImportID(localeID, botID, botVersion string) string {
	return strings.Join([]string{localeID, botID, botVersion}, lexV2ImportIDSeparator)
}

func lexV2IntentImportID(intentID, botID, botVersion, localeID string) string {
	return strings.Join([]string{intentID, botID, botVersion, localeID}, lexV2IntentImportIDSeparator)
}

func lexV2SlotImportID(botID, botVersion, intentID, localeID, slotID string) string {
	return strings.Join([]string{botID, botVersion, intentID, localeID, slotID}, lexV2ImportIDSeparator)
}

func lexV2SlotTypeImportID(botID, botVersion, localeID, slotTypeID string) string {
	return strings.Join([]string{botID, botVersion, localeID, slotTypeID}, lexV2ImportIDSeparator)
}

func lexV2ResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return lexV2ResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func lexV2BotImportable(status lexv2types.BotStatus) bool {
	return status == lexv2types.BotStatusAvailable ||
		status == lexv2types.BotStatusInactive
}

func lexV2BotLocaleImportable(status lexv2types.BotLocaleStatus) bool {
	return status == lexv2types.BotLocaleStatusBuilt ||
		status == lexv2types.BotLocaleStatusReadyExpressTesting ||
		status == lexv2types.BotLocaleStatusNotBuilt
}

func lexV2SlotTypeImportable(category lexv2types.SlotTypeCategory) bool {
	return category != lexv2types.SlotTypeCategoryExternalGrammar
}
