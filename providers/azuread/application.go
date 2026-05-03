// ApplicationServiceGenerator
package azuread

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
)

type ApplicationServiceGenerator struct {
	AzureADService
}

func (az *ApplicationServiceGenerator) listResources() ([]msgraph.Application, error) {
	client, fail := az.getApplicationsClient()
	client.BaseClient.DisableRetries = true

	var resources []msgraph.Application

	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()

	applications, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return nil, err
	}

	resources = append(resources, *applications...)

	return resources, nil
}

func (az *ApplicationServiceGenerator) appendResource(resource *msgraph.Application) error {
	if resource == nil {
		return fmt.Errorf("azuread_application resource is nil")
	}
	id, err := azureADRequiredString("azuread_application", "id", resource.ID())
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureADResourceName(resource.DisplayName, id), "azuread_application")
	return nil
}

func (az *ApplicationServiceGenerator) InitResources() error {
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

func (az *ApplicationServiceGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"application": {"id"},
	}
}
