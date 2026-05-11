// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	rekognitiontypes "github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	rekognitionCollectionResourceType      = "aws_rekognition_collection"
	rekognitionProjectResourceType         = "aws_rekognition_project"
	rekognitionStreamProcessorResourceType = "aws_rekognition_stream_processor"
	rekognitionResourceNameFallback        = "rekognition-resource"
)

var (
	rekognitionAllowEmptyValues = []string{"tags."}
	rekognitionResourceTypes    = []string{
		rekognitionServiceName(rekognitionCollectionResourceType),
		rekognitionServiceName(rekognitionProjectResourceType),
		rekognitionServiceName(rekognitionStreamProcessorResourceType),
	}
)

type RekognitionGenerator struct {
	AWSService
}

func (g *RekognitionGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := rekognitionServiceName(resource.InstanceInfo.Type)
		if g.hasTypedRekognitionFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedFilter() {
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

func (g *RekognitionGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := rekognition.NewFromConfig(config)

	if g.shouldLoadRekognitionResource(rekognitionServiceName(rekognitionCollectionResourceType)) {
		if err := g.loadCollections(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadRekognitionResource(rekognitionServiceName(rekognitionProjectResourceType)) {
		if err := g.loadProjects(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadRekognitionResource(rekognitionServiceName(rekognitionStreamProcessorResourceType)) {
		if err := g.loadStreamProcessors(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *RekognitionGenerator) shouldLoadRekognitionResource(serviceName string) bool {
	if !g.hasTypedRekognitionFilter() {
		return true
	}
	return g.hasTypedFilterFor(serviceName) || g.hasUntypedFilter()
}

func (g *RekognitionGenerator) hasTypedRekognitionFilter() bool {
	for _, serviceName := range rekognitionResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *RekognitionGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *RekognitionGenerator) hasUntypedFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" {
			return true
		}
	}
	return false
}

func rekognitionServiceName(resourceType string) string {
	return strings.TrimPrefix(resourceType, "aws_")
}

func (g *RekognitionGenerator) loadCollections(svc *rekognition.Client) error {
	p := rekognition.NewListCollectionsPaginator(svc, &rekognition.ListCollectionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if rekognitionResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, collectionID := range page.CollectionIds {
			if resource, ok := newRekognitionCollectionResource(collectionID); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RekognitionGenerator) loadProjects(svc *rekognition.Client) error {
	p := rekognition.NewDescribeProjectsPaginator(svc, &rekognition.DescribeProjectsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if rekognitionResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, project := range page.ProjectDescriptions {
			if resource, ok := newRekognitionProjectResource(project); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *RekognitionGenerator) loadStreamProcessors(svc *rekognition.Client) error {
	p := rekognition.NewListStreamProcessorsPaginator(svc, &rekognition.ListStreamProcessorsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if rekognitionResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, processor := range page.StreamProcessors {
			if resource, ok := newRekognitionStreamProcessorResource(processor); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newRekognitionCollectionResource(collectionID string) (terraformutils.Resource, bool) {
	if collectionID == "" {
		return terraformutils.Resource{}, false
	}
	return rekognitionResource(collectionID, rekognitionResourceName("collection", collectionID), rekognitionCollectionResourceType, map[string]string{
		"collection_id": collectionID,
	})
}

func newRekognitionProjectResource(project rekognitiontypes.ProjectDescription) (terraformutils.Resource, bool) {
	projectArn := StringValue(project.ProjectArn)
	projectName := rekognitionProjectNameFromARN(projectArn)
	if projectName == "" || !rekognitionProjectImportable(project.Status) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name": projectName,
	}
	if project.Feature != "" {
		attributes["feature"] = string(project.Feature)
	}
	return rekognitionResource(projectName, rekognitionResourceName("project", projectName, string(project.Feature)), rekognitionProjectResourceType, attributes)
}

func newRekognitionStreamProcessorResource(processor rekognitiontypes.StreamProcessor) (terraformutils.Resource, bool) {
	name := StringValue(processor.Name)
	if name == "" || !rekognitionStreamProcessorImportable(processor.Status) {
		return terraformutils.Resource{}, false
	}
	return rekognitionResource(name, rekognitionResourceName("stream-processor", name), rekognitionStreamProcessorResourceType, map[string]string{
		"name": name,
	})
}

func rekognitionResource(importID, name, resourceType string, attributes map[string]string) (terraformutils.Resource, bool) {
	if importID == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		importID,
		name,
		resourceType,
		"aws",
		attributes,
		rekognitionAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func rekognitionResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d-%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return rekognitionResourceNameFallback
	}
	return strings.Join(cleanParts, "/")
}

func rekognitionProjectNameFromARN(projectARN string) string {
	parts := strings.Split(projectARN, "/")
	if len(parts) < 3 {
		return ""
	}
	return parts[len(parts)-2]
}

func rekognitionProjectImportable(status rekognitiontypes.ProjectStatus) bool {
	return status == rekognitiontypes.ProjectStatusCreated
}

func rekognitionStreamProcessorImportable(status rekognitiontypes.StreamProcessorStatus) bool {
	return status == rekognitiontypes.StreamProcessorStatusRunning || status == rekognitiontypes.StreamProcessorStatusStopped
}

func rekognitionResourceNotFound(err error) bool {
	var notFound *rekognitiontypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
