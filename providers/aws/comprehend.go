// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/comprehend"
	comprehendtypes "github.com/aws/aws-sdk-go-v2/service/comprehend/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	comprehendDocumentClassifierResourceType = "aws_comprehend_document_classifier"
	comprehendEntityRecognizerResourceType   = "aws_comprehend_entity_recognizer"
	comprehendResourceNameFallback           = "comprehend-resource"
)

var (
	comprehendAllowEmptyValues = []string{"tags."}
	comprehendResourceTypes    = []string{
		comprehendServiceName(comprehendDocumentClassifierResourceType),
		comprehendServiceName(comprehendEntityRecognizerResourceType),
	}
)

type ComprehendGenerator struct {
	AWSService
}

func (g *ComprehendGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := comprehendServiceName(resource.InstanceInfo.Type)
		if g.hasTypedComprehendFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
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

func (g *ComprehendGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := comprehend.NewFromConfig(config)

	if g.shouldLoadComprehendResource(comprehendServiceName(comprehendDocumentClassifierResourceType)) {
		if err := g.loadDocumentClassifiers(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadComprehendResource(comprehendServiceName(comprehendEntityRecognizerResourceType)) {
		if err := g.loadEntityRecognizers(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *ComprehendGenerator) shouldLoadComprehendResource(serviceName string) bool {
	if !g.hasTypedComprehendFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *ComprehendGenerator) hasTypedComprehendFilter() bool {
	for _, serviceName := range comprehendResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *ComprehendGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *ComprehendGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func comprehendServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func (g *ComprehendGenerator) loadDocumentClassifiers(svc *comprehend.Client) error {
	p := comprehend.NewListDocumentClassifiersPaginator(svc, &comprehend.ListDocumentClassifiersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if comprehendResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, classifier := range page.DocumentClassifierPropertiesList {
			if resource, ok := newComprehendDocumentClassifierResource(classifier); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *ComprehendGenerator) loadEntityRecognizers(svc *comprehend.Client) error {
	p := comprehend.NewListEntityRecognizersPaginator(svc, &comprehend.ListEntityRecognizersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if comprehendResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, recognizer := range page.EntityRecognizerPropertiesList {
			if resource, ok := newComprehendEntityRecognizerResource(recognizer); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newComprehendDocumentClassifierResource(classifier comprehendtypes.DocumentClassifierProperties) (terraformutils.Resource, bool) {
	classifierARN := StringValue(classifier.DocumentClassifierArn)
	if classifierARN == "" || !comprehendModelImportable(classifier.Status) {
		return terraformutils.Resource{}, false
	}
	name := comprehendModelNameFromARN(classifierARN, "document-classifier")
	resource, ok := comprehendResource(classifierARN, comprehendResourceName("document-classifier", name, classifierARN), comprehendDocumentClassifierResourceType, map[string]string{
		"arn":  classifierARN,
		"name": name,
	})
	if ok {
		resource.IgnoreKeys = append(resource.IgnoreKeys, "^version_name_prefix$")
	}
	return resource, ok
}

func newComprehendEntityRecognizerResource(recognizer comprehendtypes.EntityRecognizerProperties) (terraformutils.Resource, bool) {
	recognizerARN := StringValue(recognizer.EntityRecognizerArn)
	if recognizerARN == "" || !comprehendModelImportable(recognizer.Status) {
		return terraformutils.Resource{}, false
	}
	name := comprehendModelNameFromARN(recognizerARN, "entity-recognizer")
	resource, ok := comprehendResource(recognizerARN, comprehendResourceName("entity-recognizer", name, recognizerARN), comprehendEntityRecognizerResourceType, map[string]string{
		"arn":  recognizerARN,
		"name": name,
	})
	if ok {
		resource.IgnoreKeys = append(resource.IgnoreKeys, "^version_name_prefix$")
	}
	return resource, ok
}

func comprehendResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		comprehendAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func comprehendResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return comprehendResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func comprehendModelNameFromARN(modelARN, modelType string) string {
	marker := modelType + "/"
	index := strings.Index(modelARN, marker)
	if index == -1 {
		return arnLastSegment(modelARN, "/")
	}
	nameAndVersion := modelARN[index+len(marker):]
	name := strings.Split(nameAndVersion, "/")[0]
	return name
}

func comprehendModelImportable(status comprehendtypes.ModelStatus) bool {
	return status == comprehendtypes.ModelStatusTrained || status == comprehendtypes.ModelStatusTrainedWithWarning
}

func comprehendResourceNotFound(err error) bool {
	var notFound *comprehendtypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
