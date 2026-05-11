// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/transcribe"
	transcribetypes "github.com/aws/aws-sdk-go-v2/service/transcribe/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	transcribeLanguageModelResourceType     = "aws_transcribe_language_model"
	transcribeMedicalVocabularyResourceType = "aws_transcribe_medical_vocabulary"
	transcribeVocabularyResourceType        = "aws_transcribe_vocabulary"
	transcribeVocabularyFilterResourceType  = "aws_transcribe_vocabulary_filter"
	transcribeResourceNameFallback          = "transcribe-resource"
)

var (
	transcribeAllowEmptyValues = []string{"tags."}
	transcribeResourceTypes    = []string{
		transcribeServiceName(transcribeLanguageModelResourceType),
		transcribeServiceName(transcribeMedicalVocabularyResourceType),
		transcribeServiceName(transcribeVocabularyFilterResourceType),
		transcribeServiceName(transcribeVocabularyResourceType),
	}
)

type TranscribeGenerator struct {
	AWSService
}

func (g *TranscribeGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := transcribeServiceName(resource.InstanceInfo.Type)
		if g.hasTypedTranscribeFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
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

func (g *TranscribeGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := transcribe.NewFromConfig(config)

	if g.shouldLoadTranscribeResource(transcribeServiceName(transcribeLanguageModelResourceType)) {
		if err := g.loadLanguageModels(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTranscribeResource(transcribeServiceName(transcribeMedicalVocabularyResourceType)) {
		if err := g.loadMedicalVocabularies(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTranscribeResource(transcribeServiceName(transcribeVocabularyResourceType)) {
		if err := g.loadVocabularies(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadTranscribeResource(transcribeServiceName(transcribeVocabularyFilterResourceType)) {
		if err := g.loadVocabularyFilters(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *TranscribeGenerator) shouldLoadTranscribeResource(serviceName string) bool {
	if !g.hasTypedTranscribeFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *TranscribeGenerator) hasTypedTranscribeFilter() bool {
	for _, serviceName := range transcribeResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *TranscribeGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *TranscribeGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func transcribeServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func (g *TranscribeGenerator) loadLanguageModels(svc *transcribe.Client) error {
	p := transcribe.NewListLanguageModelsPaginator(svc, &transcribe.ListLanguageModelsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if transcribeResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, model := range page.Models {
			if resource, ok := newTranscribeLanguageModelResource(model); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TranscribeGenerator) loadMedicalVocabularies(svc *transcribe.Client) error {
	p := transcribe.NewListMedicalVocabulariesPaginator(svc, &transcribe.ListMedicalVocabulariesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if transcribeResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, vocabulary := range page.Vocabularies {
			if resource, ok := newTranscribeMedicalVocabularyResource(vocabulary); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TranscribeGenerator) loadVocabularies(svc *transcribe.Client) error {
	p := transcribe.NewListVocabulariesPaginator(svc, &transcribe.ListVocabulariesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if transcribeResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, vocabulary := range page.Vocabularies {
			if resource, ok := newTranscribeVocabularyResource(vocabulary); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *TranscribeGenerator) loadVocabularyFilters(svc *transcribe.Client) error {
	p := transcribe.NewListVocabularyFiltersPaginator(svc, &transcribe.ListVocabularyFiltersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if transcribeResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, filter := range page.VocabularyFilters {
			if resource, ok := newTranscribeVocabularyFilterResource(filter); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newTranscribeLanguageModelResource(model transcribetypes.LanguageModel) (terraformutils.Resource, bool) {
	name := StringValue(model.ModelName)
	if name == "" || !transcribeLanguageModelImportable(model.ModelStatus) {
		return terraformutils.Resource{}, false
	}
	return transcribeResource(name, transcribeResourceName("language-model", name), transcribeLanguageModelResourceType, map[string]string{
		"model_name": name,
	})
}

func newTranscribeMedicalVocabularyResource(vocabulary transcribetypes.VocabularyInfo) (terraformutils.Resource, bool) {
	name := StringValue(vocabulary.VocabularyName)
	if name == "" || !transcribeVocabularyImportable(vocabulary.VocabularyState) {
		return terraformutils.Resource{}, false
	}
	return transcribeResource(name, transcribeResourceName("medical-vocabulary", name), transcribeMedicalVocabularyResourceType, map[string]string{
		"vocabulary_name": name,
	})
}

func newTranscribeVocabularyResource(vocabulary transcribetypes.VocabularyInfo) (terraformutils.Resource, bool) {
	name := StringValue(vocabulary.VocabularyName)
	if name == "" || !transcribeVocabularyImportable(vocabulary.VocabularyState) {
		return terraformutils.Resource{}, false
	}
	return transcribeResource(name, transcribeResourceName("vocabulary", name), transcribeVocabularyResourceType, map[string]string{
		"vocabulary_name": name,
	})
}

func newTranscribeVocabularyFilterResource(filter transcribetypes.VocabularyFilterInfo) (terraformutils.Resource, bool) {
	name := StringValue(filter.VocabularyFilterName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	return transcribeResource(name, transcribeResourceName("vocabulary-filter", name), transcribeVocabularyFilterResourceType, map[string]string{
		"vocabulary_filter_name": name,
	})
}

func transcribeResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		transcribeAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func transcribeResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return transcribeResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func transcribeLanguageModelImportable(status transcribetypes.ModelStatus) bool {
	return status == transcribetypes.ModelStatusCompleted
}

func transcribeVocabularyImportable(status transcribetypes.VocabularyState) bool {
	return status == transcribetypes.VocabularyStateReady
}

func transcribeResourceNotFound(err error) bool {
	var notFound *transcribetypes.NotFoundException
	return errors.As(err, &notFound)
}
