package azuredevops

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

type GitRepositoryGenerator struct {
	AzureDevOpsService
}

func (az *GitRepositoryGenerator) listResources() ([]git.GitRepository, error) {
	client, err := az.getGitClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	resources, err := client.GetRepositories(ctx, git.GetRepositoriesArgs{})
	if err != nil {
		return nil, err
	}
	return *resources, nil
}

func (az *GitRepositoryGenerator) appendResource(resource *git.GitRepository) error {
	if resource == nil {
		return fmt.Errorf("azuredevops_git_repository resource is nil")
	}
	id, err := azureDevOpsRequiredUUID("azuredevops_git_repository", "id", resource.Id)
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureDevOpsResourceName(resource.Name, id), "azuredevops_git_repository")
	return nil
}

func (az *GitRepositoryGenerator) InitResources() error {
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

func (az *GitRepositoryGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"project": {"project_id", "id"},
	}
}
