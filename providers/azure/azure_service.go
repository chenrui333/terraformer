// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/chenrui333/terraformer/terraformutils"
)

type AzureService struct { //nolint
	terraformutils.Service
}

func (az *AzureService) getClientArgs() (subscriptionID string, resourceGroup string, authorizer autorest.Authorizer, resourceManagerEndpoint string) {
	subs := az.Args["config"].(providerConfig).SubscriptionID
	auth := az.Args["authorizer"].(autorest.Authorizer)
	resg := az.Args["resource_group"].(string)
	rEndpoint := az.Args["config"].(providerConfig).CustomResourceManagerEndpoint
	return subs, resg, auth, rEndpoint
}

func (az *AzureService) AppendSimpleResource(id string, resourceName string, resourceType string) {
	newResource := terraformutils.NewSimpleResource(id, resourceName, resourceType, az.ProviderName, []string{})
	az.Resources = append(az.Resources, newResource)
}

func (az *AzureService) AppendSimpleResourceWithDuplicateCheck(id string, resourceName string, resourceType string) {
	tferexist, _ := az.DuplicateCheck(id, resourceName, resourceType)
	if !tferexist {
		resourceName = resourceName + "_" + id
	}
	newResource := terraformutils.NewSimpleResource(id, resourceName, resourceType, az.ProviderName, []string{})
	az.Resources = append(az.Resources, newResource)
}

// This method checks if same resource name(tfer) exists with
// same id
func (az *AzureService) DuplicateCheck(id string, resourceName string, _ string) (bool, bool) {
	var tferexist, idexist bool
	tferName := terraformutils.TfSanitize(resourceName)
	for _, resource := range az.Resources {
		if tferName == resource.ResourceName {
			if id == resource.InstanceState.ID {
				tferexist = true
				idexist = true
			} else {
				tferexist = true
				idexist = false
			}
		}
	}
	return tferexist, idexist
}

func (az *AzureService) appendSimpleAssociation(id string, linkedResourceName string, resourceName *string, resourceType string, attributes map[string]string) {
	var resourceName2 string
	if resourceName != nil {
		resourceName2 = *resourceName
	} else {
		resourceName0 := strings.ReplaceAll(resourceType, "azurerm_", "")
		resourceName1 := resourceName0[strings.IndexByte(resourceName0, '_'):]
		resourceName2 = linkedResourceName + resourceName1
	}
	newResource := terraformutils.NewResource(
		id, resourceName2, resourceType, az.ProviderName, attributes,
		[]string{"name"},
		map[string]interface{}{},
	)
	az.Resources = append(az.Resources, newResource)
}
