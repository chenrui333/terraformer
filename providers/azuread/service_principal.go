// ServicePrincipalServiceGenerator
package azuread

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
)

type ServicePrincipalServiceGenerator struct {
	AzureADService
}

func (az *ServicePrincipalServiceGenerator) listResources() ([]msgraph.ServicePrincipal, error) {
	client, fail := az.getServicePrincipalsClient()
	client.BaseClient.DisableRetries = true

	var resources []msgraph.ServicePrincipal

	if fail != nil {
		return nil, fail
	}
	ctx := context.Background()

	servicePrincipal, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return nil, err
	}

	resources = append(resources, *servicePrincipal...)

	return resources, nil
}

func (az *ServicePrincipalServiceGenerator) appendResource(resource *msgraph.ServicePrincipal) error {
	if resource == nil {
		return fmt.Errorf("azuread_service_principal resource is nil")
	}
	id, err := azureADRequiredString("azuread_service_principal", "id", resource.ID())
	if err != nil {
		return err
	}
	az.appendSimpleResource(id, azureADQualifiedResourceName(resource.DisplayName, id), "azuread_service_principal")
	return nil
}

func (az *ServicePrincipalServiceGenerator) InitResources() error {
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

func (az *ServicePrincipalServiceGenerator) GetResourceConnections() map[string][]string {
	return map[string][]string{
		"servicePrincipal": {"id"},
	}
}
