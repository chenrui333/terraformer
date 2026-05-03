// UserServiceGenerator
package azuread

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
)

type UserServiceGenerator struct {
	AzureADService
}

func (az *UserServiceGenerator) listResources() ([]msgraph.User, error) {
	client, fail := az.getUserClient()
	client.BaseClient.DisableRetries = true

	var resources []msgraph.User

	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()

	users, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return nil, err
	}

	resources = append(resources, *users...)

	return resources, nil
}

func (az *UserServiceGenerator) appendResource(resource *msgraph.User) error {
	if resource == nil {
		return fmt.Errorf("azuread_user resource is nil")
	}
	id, err := azureADRequiredString("azuread_user", "id", resource.ID())
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureADResourceName(resource.DisplayName, id), "azuread_user")
	return nil
}

func (az *UserServiceGenerator) InitResources() error {
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

func (az *UserServiceGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"user": {"id"},
	}
}
