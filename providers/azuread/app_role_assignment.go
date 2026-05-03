// AppRoleAssignmentServiceGenerator
package azuread

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
)

type AppRoleAssignmentServiceGenerator struct {
	AzureADService
}

func (az *AppRoleAssignmentServiceGenerator) listResources() ([]msgraph.AppRoleAssignment, error) {
	client, fail := az.getAppRoleAssignmentsClient()
	servicePrincipalsClient, err := az.getServicePrincipalsClient()
	if err != nil {
		return nil, err
	}
	client.BaseClient.DisableRetries = true

	var resources []msgraph.AppRoleAssignment

	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()

	servicePrincipals, _, spErr := servicePrincipalsClient.List(ctx, odata.Query{})
	if spErr != nil {
		return nil, spErr
	}

	for _, sp := range *servicePrincipals {
		spID := sp.ID()
		if spID == nil {
			continue
		}
		appRoleAssignments, _, araErr := client.List(ctx, *spID, odata.Query{})
		if araErr != nil {
			return nil, araErr
		}
		if appRoleAssignments == nil {
			continue
		}
		for _, assignment := range *appRoleAssignments {
			if assignment.PrincipalType == nil || *assignment.PrincipalType != "ServicePrincipal" {
				continue
			}
			if assignment.Id != nil {
				resources = append(resources, assignment)
			}
		}
	}

	return resources, nil
}

func (az *AppRoleAssignmentServiceGenerator) appendResource(resource *msgraph.AppRoleAssignment) error {
	if resource == nil {
		return fmt.Errorf("azuread_app_role_assignment resource is nil")
	}
	// {objectId}/{type}/{subId}
	principalID, err := azureADRequiredString("azuread_app_role_assignment", "principalId", resource.PrincipalId)
	if err != nil {
		return err
	}
	assignmentID, err := azureADRequiredString("azuread_app_role_assignment", "id", resource.Id)
	if err != nil {
		return err
	}
	id := fmt.Sprintf("%s/appRoleAssignment/%s", principalID, assignmentID)
	az.appendSimpleResource(id, azureADQualifiedResourceName(resource.PrincipalDisplayName, id), "azuread_app_role_assignment")
	return nil
}

func (az *AppRoleAssignmentServiceGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		log.Println(azureADResourceName(resource.PrincipalDisplayName, azureADStringValue(resource.PrincipalId)))
		if err := az.appendResource(&resource); err != nil {
			return err
		}
	}
	return nil
}

func (az *AppRoleAssignmentServiceGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"app_role_assignment": {"id"},
	}
}
