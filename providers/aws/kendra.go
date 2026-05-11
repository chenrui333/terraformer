// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kendra"
	kendratypes "github.com/aws/aws-sdk-go-v2/service/kendra/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	kendraDataSourceResourceType                = "aws_kendra_data_source"
	kendraExperienceResourceType                = "aws_kendra_experience"
	kendraFaqResourceType                       = "aws_kendra_faq"
	kendraIndexResourceType                     = "aws_kendra_index"
	kendraQuerySuggestionsBlockListResourceType = "aws_kendra_query_suggestions_block_list"
	kendraThesaurusResourceType                 = "aws_kendra_thesaurus"
	kendraImportIDSeparator                     = "/"
	kendraResourceNameFallback                  = "kendra-resource"
)

var (
	kendraAllowEmptyValues = []string{"tags."}
	kendraResourceTypes    = []string{
		kendraServiceName(kendraDataSourceResourceType),
		kendraServiceName(kendraExperienceResourceType),
		kendraServiceName(kendraFaqResourceType),
		kendraServiceName(kendraIndexResourceType),
		kendraServiceName(kendraQuerySuggestionsBlockListResourceType),
		kendraServiceName(kendraThesaurusResourceType),
	}
)

type KendraGenerator struct {
	AWSService
}

func (g *KendraGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := kendraServiceName(resource.InstanceInfo.Type)
		if g.hasTypedKendraFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
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

func (g *KendraGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := kendra.NewFromConfig(config)

	loadIndexes := g.shouldLoadKendraResource(kendraServiceName(kendraIndexResourceType))
	loadDataSources := g.shouldLoadKendraResource(kendraServiceName(kendraDataSourceResourceType))
	loadFaqs := g.shouldLoadKendraResource(kendraServiceName(kendraFaqResourceType))
	loadQuerySuggestionsBlockLists := g.shouldLoadKendraResource(kendraServiceName(kendraQuerySuggestionsBlockListResourceType))
	loadThesauri := g.shouldLoadKendraResource(kendraServiceName(kendraThesaurusResourceType))
	loadExperiences := g.shouldLoadKendraResource(kendraServiceName(kendraExperienceResourceType))

	if loadIndexes || loadDataSources || loadFaqs || loadQuerySuggestionsBlockLists || loadThesauri || loadExperiences {
		indices, err := listKendraIndices(svc)
		if err != nil {
			return err
		}
		if loadIndexes {
			g.loadIndices(indices)
		}
		if loadDataSources {
			if err := g.loadDataSources(svc, indices); err != nil {
				return err
			}
		}
		if loadFaqs {
			if err := g.loadFaqs(svc, indices); err != nil {
				return err
			}
		}
		if loadQuerySuggestionsBlockLists {
			if err := g.loadQuerySuggestionsBlockLists(svc, indices); err != nil {
				return err
			}
		}
		if loadThesauri {
			if err := g.loadThesauri(svc, indices); err != nil {
				return err
			}
		}
		if loadExperiences {
			if err := g.loadExperiences(svc, indices); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *KendraGenerator) shouldLoadKendraResource(serviceName string) bool {
	if !g.hasTypedKendraFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *KendraGenerator) hasTypedKendraFilter() bool {
	for _, serviceName := range kendraResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *KendraGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *KendraGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func kendraServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func listKendraIndices(svc *kendra.Client) ([]kendratypes.IndexConfigurationSummary, error) {
	indices := []kendratypes.IndexConfigurationSummary{}
	p := kendra.NewListIndicesPaginator(svc, &kendra.ListIndicesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		indices = append(indices, page.IndexConfigurationSummaryItems...)
	}
	return indices, nil
}

func (g *KendraGenerator) loadIndices(indices []kendratypes.IndexConfigurationSummary) {
	for _, index := range indices {
		if resource, ok := newKendraIndexResource(index); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
}

func (g *KendraGenerator) loadDataSources(svc *kendra.Client, indices []kendratypes.IndexConfigurationSummary) error {
	for _, index := range indices {
		indexID := StringValue(index.Id)
		if indexID == "" || !kendraIndexImportable(index.Status) {
			continue
		}
		p := kendra.NewListDataSourcesPaginator(svc, &kendra.ListDataSourcesInput{IndexId: aws.String(indexID)})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if kendraResourceNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			for _, dataSource := range page.SummaryItems {
				if resource, ok := newKendraDataSourceResource(indexID, dataSource); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *KendraGenerator) loadFaqs(svc *kendra.Client, indices []kendratypes.IndexConfigurationSummary) error {
	for _, index := range indices {
		indexID := StringValue(index.Id)
		if indexID == "" || !kendraIndexImportable(index.Status) {
			continue
		}
		p := kendra.NewListFaqsPaginator(svc, &kendra.ListFaqsInput{IndexId: aws.String(indexID)})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if kendraResourceNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			for _, faq := range page.FaqSummaryItems {
				if resource, ok := newKendraFaqResource(indexID, faq); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *KendraGenerator) loadQuerySuggestionsBlockLists(svc *kendra.Client, indices []kendratypes.IndexConfigurationSummary) error {
	for _, index := range indices {
		indexID := StringValue(index.Id)
		if indexID == "" || !kendraIndexImportable(index.Status) {
			continue
		}
		p := kendra.NewListQuerySuggestionsBlockListsPaginator(svc, &kendra.ListQuerySuggestionsBlockListsInput{IndexId: aws.String(indexID)})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if kendraResourceNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			for _, blockList := range page.BlockListSummaryItems {
				if resource, ok := newKendraQuerySuggestionsBlockListResource(indexID, blockList); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *KendraGenerator) loadThesauri(svc *kendra.Client, indices []kendratypes.IndexConfigurationSummary) error {
	for _, index := range indices {
		indexID := StringValue(index.Id)
		if indexID == "" || !kendraIndexImportable(index.Status) {
			continue
		}
		p := kendra.NewListThesauriPaginator(svc, &kendra.ListThesauriInput{IndexId: aws.String(indexID)})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if kendraResourceNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			for _, thesaurus := range page.ThesaurusSummaryItems {
				if resource, ok := newKendraThesaurusResource(indexID, thesaurus); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *KendraGenerator) loadExperiences(svc *kendra.Client, indices []kendratypes.IndexConfigurationSummary) error {
	for _, index := range indices {
		indexID := StringValue(index.Id)
		if indexID == "" || !kendraIndexImportable(index.Status) {
			continue
		}
		p := kendra.NewListExperiencesPaginator(svc, &kendra.ListExperiencesInput{IndexId: aws.String(indexID)})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if kendraResourceNotFound(err) {
				break
			}
			if err != nil {
				return err
			}
			for _, experience := range page.SummaryItems {
				if resource, ok := newKendraExperienceResource(indexID, experience); ok {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func newKendraIndexResource(index kendratypes.IndexConfigurationSummary) (terraformutils.Resource, bool) {
	indexID := StringValue(index.Id)
	if indexID == "" || !kendraIndexImportable(index.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(index.Name), indexID)
	return kendraResource(indexID, kendraResourceName("index", indexID, name), kendraIndexResourceType, map[string]string{
		"id":   indexID,
		"name": name,
	})
}

func newKendraDataSourceResource(indexID string, dataSource kendratypes.DataSourceSummary) (terraformutils.Resource, bool) {
	dataSourceID := StringValue(dataSource.Id)
	if indexID == "" || dataSourceID == "" || !kendraDataSourceImportable(dataSource.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(dataSource.Name), dataSourceID)
	return kendraResource(kendraChildImportID(dataSourceID, indexID), kendraResourceName("data-source", indexID, dataSourceID, name), kendraDataSourceResourceType, map[string]string{
		"data_source_id": dataSourceID,
		"index_id":       indexID,
		"name":           name,
	})
}

func newKendraFaqResource(indexID string, faq kendratypes.FaqSummary) (terraformutils.Resource, bool) {
	faqID := StringValue(faq.Id)
	if indexID == "" || faqID == "" || !kendraFaqImportable(faq.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(faq.Name), faqID)
	return kendraResource(kendraChildImportID(faqID, indexID), kendraResourceName("faq", indexID, faqID, name), kendraFaqResourceType, map[string]string{
		"faq_id":   faqID,
		"index_id": indexID,
		"name":     name,
	})
}

func newKendraQuerySuggestionsBlockListResource(indexID string, blockList kendratypes.QuerySuggestionsBlockListSummary) (terraformutils.Resource, bool) {
	blockListID := StringValue(blockList.Id)
	if indexID == "" || blockListID == "" || !kendraQuerySuggestionsBlockListImportable(blockList.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(blockList.Name), blockListID)
	return kendraResource(kendraChildImportID(blockListID, indexID), kendraResourceName("query-suggestions-block-list", indexID, blockListID, name), kendraQuerySuggestionsBlockListResourceType, map[string]string{
		"query_suggestions_block_list_id": blockListID,
		"index_id":                        indexID,
		"name":                            name,
	})
}

func newKendraThesaurusResource(indexID string, thesaurus kendratypes.ThesaurusSummary) (terraformutils.Resource, bool) {
	thesaurusID := StringValue(thesaurus.Id)
	if indexID == "" || thesaurusID == "" || !kendraThesaurusImportable(thesaurus.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(thesaurus.Name), thesaurusID)
	return kendraResource(kendraChildImportID(thesaurusID, indexID), kendraResourceName("thesaurus", indexID, thesaurusID, name), kendraThesaurusResourceType, map[string]string{
		"thesaurus_id": thesaurusID,
		"index_id":     indexID,
		"name":         name,
	})
}

func newKendraExperienceResource(indexID string, experience kendratypes.ExperiencesSummary) (terraformutils.Resource, bool) {
	experienceID := StringValue(experience.Id)
	if indexID == "" || experienceID == "" || !kendraExperienceImportable(experience.Status) {
		return terraformutils.Resource{}, false
	}
	name := firstNonEmpty(StringValue(experience.Name), experienceID)
	return kendraResource(kendraChildImportID(experienceID, indexID), kendraResourceName("experience", indexID, experienceID, name), kendraExperienceResourceType, map[string]string{
		"experience_id": experienceID,
		"index_id":      indexID,
		"name":          name,
	})
}

func kendraResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		kendraAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func kendraChildImportID(childID, indexID string) string {
	return strings.Join([]string{childID, indexID}, kendraImportIDSeparator)
}

func kendraResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return kendraResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func kendraIndexImportable(status kendratypes.IndexStatus) bool {
	return status == kendratypes.IndexStatusActive
}

func kendraDataSourceImportable(status kendratypes.DataSourceStatus) bool {
	return status == kendratypes.DataSourceStatusActive
}

func kendraFaqImportable(status kendratypes.FaqStatus) bool {
	return status == kendratypes.FaqStatusActive
}

func kendraQuerySuggestionsBlockListImportable(status kendratypes.QuerySuggestionsBlockListStatus) bool {
	return status == kendratypes.QuerySuggestionsBlockListStatusActive
}

func kendraThesaurusImportable(status kendratypes.ThesaurusStatus) bool {
	return status == kendratypes.ThesaurusStatusActive
}

func kendraExperienceImportable(status kendratypes.ExperienceStatus) bool {
	return status == kendratypes.ExperienceStatusActive
}

func kendraResourceNotFound(err error) bool {
	var notFound *kendratypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
