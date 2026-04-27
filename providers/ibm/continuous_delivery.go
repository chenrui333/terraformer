// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"os"

	"github.com/IBM-Cloud/bluemix-go"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev1/catalog"
	"github.com/IBM-Cloud/bluemix-go/api/resource/resourcev2/controllerv2"
	"github.com/IBM-Cloud/bluemix-go/session"
	"github.com/chenrui333/terraformer/terraformutils"
)

// DatabaseRedisGenerator ...
type ContinuousDeliveryGenerator struct {
	IBMService
}

// loadRedisDB ...
func (g ContinuousDeliveryGenerator) loadContinuousDelivery(cdID string, cdName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		cdID,
		normalizeResourceName(cdName, true),
		"ibm_resource_instance",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *ContinuousDeliveryGenerator) InitResources() error {
	region := g.Args["region"].(string)
	bmxConfig := &bluemix.Config{
		BluemixAPIKey: os.Getenv("IC_API_KEY"),
		Region:        region,
	}
	sess, err := session.New(bmxConfig)
	if err != nil {
		return err
	}

	catalogClient, err := catalog.New(sess)
	if err != nil {
		return err
	}

	controllerClient, err := controllerv2.New(sess)
	if err != nil {
		return err
	}

	serviceID, err := catalogClient.ResourceCatalog().FindByName("continuous-delivery", true)
	if err != nil {
		return err
	}
	query := controllerv2.ServiceInstanceQuery{
		ServiceID: serviceID[0].ID,
	}
	continuousDeliveryInstances, err := controllerClient.ResourceServiceInstanceV2().ListInstances(query)
	if err != nil {
		return err
	}

	for _, cd := range continuousDeliveryInstances {
		if cd.RegionID == region {
			g.Resources = append(g.Resources, g.loadContinuousDelivery(cd.ID, cd.Name))
		}
	}

	return nil
}
