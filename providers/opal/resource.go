package opal

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	opalsdk "github.com/opalsecurity/opal-go"
)

type ResourceGenerator struct {
	OpalService
}

func opalResourceNameSuffix(resourceID string) string {
	if len(resourceID) > 8 {
		return resourceID[:8]
	}
	return resourceID
}

func (g *ResourceGenerator) createResources(opalResources []*opalsdk.Resource) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opalResourceByID := make(map[string]*opalsdk.Resource)
	for _, resource := range opalResources {
		if resource == nil {
			return nil, fmt.Errorf("opal_resource resource is nil")
		}
		resourceID, err := opalRequiredString("opal_resource", "resource_id", resource.ResourceId)
		if err != nil {
			return nil, err
		}
		opalResourceByID[resourceID] = resource
	}

	seenNames := make(map[string]int)
	for _, resource := range opalResources {
		resourceID, err := opalRequiredString("opal_resource", "resource_id", resource.ResourceId)
		if err != nil {
			return nil, err
		}
		tfname := opalResourceDisplayName(resource.Name, resourceID)
		if resource.ResourceType != nil &&
			*resource.ResourceType == opalsdk.RESOURCETYPEENUM_AWS_SSO_PERMISSION_SET {
			parentResourceID, err := opalRequiredStringPtr("opal_resource", "parent_resource_id", resource.ParentResourceId)
			if err != nil {
				return nil, err
			}
			parentAccount, ok := opalResourceByID[parentResourceID]
			if !ok {
				return nil, fmt.Errorf("could not find account for permission set %q: parent resource %q", resourceID, parentResourceID)
			}
			parentID, err := opalRequiredString("opal_resource parent", "resource_id", parentAccount.ResourceId)
			if err != nil {
				return nil, err
			}
			tfname = fmt.Sprintf("%s_%s", opalResourceDisplayName(parentAccount.Name, parentID), tfname)
		}

		tfname = opalUniqueResourceNameWithSuffix(tfname, opalResourceNameSuffix(resourceID), seenNames)

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			tfname,
			"opal_resource",
			"opal",
			[]string{},
		))
	}

	return resources, nil
}

func (g *ResourceGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to list opal resources: %w", err)
	}

	resources, _, err := client.ResourcesAPI.GetResources(context.TODO()).Execute()
	if err != nil {
		return fmt.Errorf("unable to list opal resources: %w", err)
	}

	var opalResources []*opalsdk.Resource
	for {
		for _, resource := range resources.Results {
			resourceRef := resource
			opalResources = append(opalResources, &resourceRef)
		}

		if !resources.HasNext() || resources.Next == nil {
			break
		}

		resources, _, err = client.ResourcesAPI.GetResources(context.TODO()).Cursor(*resources.Next).Execute()
		if err != nil {
			return fmt.Errorf("unable to list opal resources: %w", err)
		}
	}

	resourcesList, err := g.createResources(opalResources)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resourcesList...)

	return nil
}
