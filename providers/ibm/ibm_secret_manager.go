// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"os"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/catalog"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev2/controllerv2"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/chenrui333/terraformer/terraformutils"
)

type SecretsManagerGenerator struct {
	IBMService
}

func (g SecretsManagerGenerator) loadSM(smID, smName, servicePlan string, timeout map[string]string) terraformutils.Resource {
	resources := terraformutils.NewResource(
		smID,
		normalizeResourceName(smName, true),
		"ibm_resource_instance",
		"ibm",
		map[string]string{
			"plan": servicePlan,
		},
		[]string{},
		map[string]interface{}{
			"timeouts": timeout,
		})
	return resources
}

func (g *SecretsManagerGenerator) InitResources() error {
	bmxConfig := &bluemix.Config{
		BluemixAPIKey: os.Getenv("IC_API_KEY"),
	}

	sess, err := session.New(bmxConfig)
	if err != nil {
		return err
	}

	// Client creation
	catalogClient, err := catalog.New(sess)
	if err != nil {
		return err
	}

	controllerClient, err := controllerv2.New(sess)
	if err != nil {
		return err
	}

	// Get ServiceID of secret manager service
	serviceID, err := catalogClient.ResourceCatalog().FindByName("secrets-manager", true)
	if err != nil {
		return err
	}

	query := controllerv2.ServiceInstanceQuery{
		ServiceID: serviceID[0].ID,
	}

	// Get all Secret manager instances
	smInstances, err := controllerClient.ResourceServiceInstanceV2().ListInstances(query)
	if err != nil {
		return err
	}

	for _, smInstance := range smInstances {
		timeout := map[string]string{"create": "15m"}
		g.Resources = append(g.Resources, g.loadSM(smInstance.ID, smInstance.Name, smInstance.ServicePlanName, timeout))
	}
	return nil
}
