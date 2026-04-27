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

// MonitoringGenerator ...
type MonitoringGenerator struct {
	IBMService
}

// loadCloudMonitoring ...
func (g MonitoringGenerator) loadCloudMonitoring(cdID, cdName, service, region string) terraformutils.Resource {
	resources := terraformutils.NewResource(
		cdID,
		normalizeResourceName(cdName, true),
		"ibm_resource_instance",
		"ibm",
		map[string]string{
			"name":     cdName,
			"service":  service,
			"location": region,
		},
		[]string{},
		map[string]interface{}{})
	return resources
}

// InitResources ...
func (g *MonitoringGenerator) InitResources() error {
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

	serviceID, err := catalogClient.ResourceCatalog().FindByName("sysdig-monitor", true)
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
		if cd.RegionID == region && cd.Name != "" {
			g.Resources = append(g.Resources, g.loadCloudMonitoring(cd.ID, cd.Name, cd.ServiceName, cd.RegionID))
		}
	}

	return nil
}
