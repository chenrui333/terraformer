// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// SoftwareCatalogAllowEmptyValues ...
	SoftwareCatalogAllowEmptyValues = []string{}
)

// SoftwareCatalogGenerator ...
type SoftwareCatalogGenerator struct {
	DatadogService
}

func (g *SoftwareCatalogGenerator) createResource(entityRef string) (terraformutils.Resource, error) {
	if entityRef == "" {
		return terraformutils.Resource{}, fmt.Errorf("software catalog entity missing id")
	}

	return terraformutils.NewSimpleResource(
		entityRef,
		fmt.Sprintf("software_catalog_%s", entityRef),
		"datadog_software_catalog",
		"datadog",
		SoftwareCatalogAllowEmptyValues,
	), nil
}

func (g *SoftwareCatalogGenerator) createResources(entityRefs []string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, entityRef := range entityRefs {
		resource, err := g.createResource(entityRef)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each software_catalog entity create 1 TerraformResource.
// Need catalog entity reference as ID for terraform resource.
func (g *SoftwareCatalogGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewSoftwareCatalogApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, api)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	entityRefs, err := listImportableSoftwareCatalogEntityRefs(auth, api)
	if err != nil {
		return err
	}
	g.Resources, err = g.createResources(entityRefs)
	return err
}

func (g *SoftwareCatalogGenerator) filteredResources(auth context.Context, api *datadogV2.SoftwareCatalogApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	matchedIDFilter := false
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("software_catalog") {
			continue
		}
		matchedIDFilter = true
		for _, value := range filter.AcceptableValues {
			entityRefs, err := listSoftwareCatalogEntityRefsByFilter(auth, api, value)
			if err != nil {
				return nil, false, err
			}
			if len(entityRefs) == 0 {
				return nil, false, fmt.Errorf("software catalog entity %q not found with reconstructable raw schema", value)
			}
			for _, entityRef := range entityRefs {
				resource, err := g.createResource(entityRef)
				if err != nil {
					return nil, false, err
				}
				resources = append(resources, resource)
			}
		}
	}
	return resources, matchedIDFilter, nil
}

func listSoftwareCatalogEntityRefsByFilter(auth context.Context, api *datadogV2.SoftwareCatalogApi, entityRef string) ([]string, error) {
	optionalParams := catalogEntityListOptionalParameters().WithFilterRef(entityRef)
	response, httpResp, err := api.ListCatalogEntity(auth, *optionalParams)
	closeDatadogResponseBody(httpResp)
	if err != nil {
		return nil, err
	}
	return importableSoftwareCatalogEntityRefs(response)
}

func listImportableSoftwareCatalogEntityRefs(auth context.Context, api *datadogV2.SoftwareCatalogApi) ([]string, error) {
	pageSize := int64(100)
	offset := int64(0)
	entityRefs := []string{}

	for {
		optionalParams := catalogEntityListOptionalParameters().
			WithPageLimit(pageSize).
			WithPageOffset(offset)
		response, httpResp, err := api.ListCatalogEntity(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		refs, err := importableSoftwareCatalogEntityRefs(response)
		if err != nil {
			return nil, err
		}
		entityRefs = append(entityRefs, refs...)

		if len(response.GetData()) < int(pageSize) {
			break
		}
		offset += pageSize
	}
	return entityRefs, nil
}

func catalogEntityListOptionalParameters() *datadogV2.ListCatalogEntityOptionalParameters {
	return datadogV2.NewListCatalogEntityOptionalParameters().
		WithInclude(datadogV2.INCLUDETYPE_RAW_SCHEMA).
		WithIncludeDiscovered(false).
		WithFilterExcludeSnapshot("true")
}

func importableSoftwareCatalogEntityRefs(response datadogV2.ListEntityCatalogResponse) ([]string, error) {
	rawSchemaIDs := softwareCatalogRawSchemaIDs(response)
	refs := []string{}
	for _, entity := range response.GetData() {
		if !softwareCatalogEntityHasRawSchema(entity, rawSchemaIDs) {
			continue
		}
		ref, err := softwareCatalogEntityRef(entity)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func softwareCatalogRawSchemaIDs(response datadogV2.ListEntityCatalogResponse) map[string]struct{} {
	rawSchemaIDs := map[string]struct{}{}
	for _, included := range response.GetIncluded() {
		rawSchema := included.EntityResponseIncludedRawSchema
		if rawSchema == nil || rawSchema.GetId() == "" {
			continue
		}
		rawSchemaAttributes := rawSchema.GetAttributes()
		if rawSchemaAttributes.GetRawSchema() == "" {
			continue
		}
		rawSchemaIDs[rawSchema.GetId()] = struct{}{}
	}
	return rawSchemaIDs
}

func softwareCatalogEntityHasRawSchema(entity datadogV2.EntityData, rawSchemaIDs map[string]struct{}) bool {
	relationships, ok := entity.GetRelationshipsOk()
	if !ok {
		return false
	}
	rawSchema, ok := relationships.GetRawSchemaOk()
	if !ok {
		return false
	}
	rawSchemaData, ok := rawSchema.GetDataOk()
	if !ok {
		return false
	}
	_, ok = rawSchemaIDs[rawSchemaData.GetId()]
	return ok
}

func softwareCatalogEntityRef(entity datadogV2.EntityData) (string, error) {
	attributes, ok := entity.GetAttributesOk()
	if !ok {
		return "", fmt.Errorf("software catalog entity missing attributes")
	}
	kind := attributes.GetKind()
	name := attributes.GetName()
	namespace := attributes.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}
	if kind == "" || name == "" {
		return "", fmt.Errorf("software catalog entity missing kind or name")
	}
	return fmt.Sprintf("%s:%s/%s", kind, namespace, name), nil
}
