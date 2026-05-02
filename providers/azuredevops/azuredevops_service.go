// SPDX-License-Identifier: Apache-2.0

//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package azuredevops

import (
	"context"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/graph"

	"github.com/chenrui333/terraformer/terraformutils"
)

type AzureDevOpsServiceGenerator interface {
	terraformutils.ServiceGenerator
	GetResourceConnections() map[string][]string
}

type AzureDevOpsService struct { //nolint
	terraformutils.Service
}

func (az *AzureDevOpsService) GetResourceConnections() map[string][]string {
	return nil
}

func (az *AzureDevOpsService) getConnection() *azuredevops.Connection {
	organizationURL := az.Args["organizationURL"].(string)
	personalAccessToken := az.Args["personalAccessToken"].(string)
	return azuredevops.NewPatConnection(organizationURL, personalAccessToken)
}

func (az *AzureDevOpsService) getCoreClient() (core.Client, error) {
	ctx := context.Background()
	client, err := core.NewClient(ctx, az.getConnection())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (az *AzureDevOpsService) getGraphClient() (graph.Client, error) {
	ctx := context.Background()
	client, err := graph.NewClient(ctx, az.getConnection())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (az *AzureDevOpsService) getGitClient() (git.Client, error) {
	ctx := context.Background()
	client, err := git.NewClient(ctx, az.getConnection())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (az *AzureDevOpsService) appendSimpleResource(id string, resourceName string, resourceType string) {
	newResource := terraformutils.NewSimpleResource(id, resourceName, resourceType, az.ProviderName, []string{})
	az.Resources = append(az.Resources, newResource)
}
