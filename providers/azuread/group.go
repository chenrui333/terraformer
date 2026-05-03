// UserServiceGenerator
package azuread

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
)

type GroupServiceGenerator struct {
	AzureADService
}

func (az *GroupServiceGenerator) listResources() ([]msgraph.Group, error) {
	client, fail := az.getGroupsClient()
	client.BaseClient.DisableRetries = true

	var resources []msgraph.Group

	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()

	groups, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return nil, err
	}

	resources = append(resources, *groups...)

	return resources, nil
}

func (az *GroupServiceGenerator) appendResource(resource *msgraph.Group) error {
	if resource == nil {
		return fmt.Errorf("azuread_group resource is nil")
	}
	id, err := azureADRequiredString("azuread_group", "id", resource.ID())
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureADQualifiedResourceName(resource.DisplayName, id), "azuread_group")
	return nil
}

func (az *GroupServiceGenerator) InitResources() error {
	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		log.Println(azureADResourceName(resource.DisplayName, azureADStringValue(resource.ID())))
		if err := az.appendResource(&resource); err != nil {
			return err
		}
	}
	return nil
}

func (az *GroupServiceGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"group": {"id"},
	}
}
