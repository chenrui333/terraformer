package azuredevops

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
)

type ProjectGenerator struct {
	AzureDevOpsService
}

func (az *ProjectGenerator) listResources() ([]core.TeamProjectReference, error) {
	client, fail := az.getCoreClient()
	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()
	var resources []core.TeamProjectReference
	pageArgs := core.GetProjectsArgs{}
	pages, err := client.GetProjects(ctx, pageArgs)
	for ; err == nil; pages, err = client.GetProjects(ctx, pageArgs) {
		fetched := *pages
		items := fetched.Value
		resources = append(resources, items...)
		if pages.ContinuationToken == "" {
			return resources, nil
		}
		pageArgs = core.GetProjectsArgs{
			ContinuationToken: &pages.ContinuationToken,
		}
	}
	return nil, err
}

func (az *ProjectGenerator) appendResource(resource *core.TeamProjectReference) error {
	if resource == nil {
		return fmt.Errorf("azuredevops_project resource is nil")
	}
	id, err := azureDevOpsRequiredUUID("azuredevops_project", "id", resource.Id)
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureDevOpsResourceName(resource.Name, id), "azuredevops_project")
	return nil
}

func (az *ProjectGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		if err := az.appendResource(&resource); err != nil {
			return err
		}
	}
	return nil
}
