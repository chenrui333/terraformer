// SPDX-License-Identifier: Apache-2.0

package azuread

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/manicminer/hamilton/msgraph"
)

type AzureADService struct { //nolint
	terraformutils.Service
}

type ServiceGenerator interface {
	terraformutils.ServiceGenerator
	GetResourceConnections() map[string][]string
}

func (az *AzureADService) getAuthorizer() (auth.Authorizer, error) {
	environment := environments.AzurePublic()
	ctx := context.Background()
	tenantID := az.Args["tenant_id"].(string)
	clientID := az.Args["client_id"].(string)
	clientSecret := az.Args["client_secret"].(string)

	credentials := auth.Credentials{
		Environment:                           *environment,
		TenantID:                              tenantID,
		ClientID:                              clientID,
		ClientSecret:                          clientSecret,
		EnableAuthenticatingUsingClientSecret: true,
	}
	authorizer, err := auth.NewAuthorizerFromCredentials(ctx, credentials, environment.MicrosoftGraph)
	if err != nil {
		return nil, err
	}
	return authorizer, nil
}

func (az *AzureADService) getUserClient() (*msgraph.UsersClient, error) {
	authorizer, err := az.getAuthorizer()
	if err != nil {
		return nil, err
	}

	client := msgraph.NewUsersClient()
	client.BaseClient.Authorizer = authorizer

	return client, nil
}

func (az *AzureADService) getApplicationsClient() (*msgraph.ApplicationsClient, error) {
	authorizer, err := az.getAuthorizer()
	if err != nil {
		return nil, err
	}

	client := msgraph.NewApplicationsClient()
	client.BaseClient.Authorizer = authorizer

	return client, nil
}

func (az *AzureADService) getGroupsClient() (*msgraph.GroupsClient, error) {
	authorizer, err := az.getAuthorizer()
	if err != nil {
		return nil, err
	}

	client := msgraph.NewGroupsClient()
	client.BaseClient.Authorizer = authorizer

	return client, nil
}

func (az *AzureADService) getServicePrincipalsClient() (*msgraph.ServicePrincipalsClient, error) {
	authorizer, err := az.getAuthorizer()
	if err != nil {
		return nil, err
	}

	client := msgraph.NewServicePrincipalsClient()
	client.BaseClient.Authorizer = authorizer

	return client, nil
}

func (az *AzureADService) getAppRoleAssignmentsClient() (*msgraph.AppRoleAssignedToClient, error) {
	authorizer, err := az.getAuthorizer()
	if err != nil {
		return nil, err
	}

	client := msgraph.NewAppRoleAssignedToClient()
	client.BaseClient.Authorizer = authorizer

	return client, nil
}

func (az *AzureADService) GetResourceConnections() map[string][]string {
	return nil
}

func (az *AzureADService) appendSimpleResource(id string, resourceName string, resourceType string) {
	newResource := terraformutils.NewResource(id, resourceName, resourceType, az.ProviderName, map[string]string{
		"id": id,
	}, []string{}, map[string]interface{}{})
	az.Resources = append(az.Resources, newResource)
}

func azureADRequiredString(resourceType, field string, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", fmt.Errorf("%s resource is missing %s", resourceType, field)
	}
	return *value, nil
}

func azureADStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func azureADResourceName(displayName *string, fallback string) string {
	if displayName != nil && *displayName != "" {
		return *displayName
	}
	return fallback
}

func azureADQualifiedResourceName(displayName *string, id string) string {
	name := azureADResourceName(displayName, id)
	if name == id {
		return id
	}
	return name + "-" + id
}
