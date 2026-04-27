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

// DatabaseElasticSearchGenerator ...
type DatabaseElasticSearchGenerator struct {
	IBMService
}

// loadElasticSearchDB ...
func (g DatabaseElasticSearchGenerator) loadElasticSearchDB(dbID string, dbName string) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		dbID,
		normalizeResourceName(dbName, false),
		"ibm_database",
		"ibm",
		[]string{})

	resource.IgnoreKeys = append(resource.IgnoreKeys,
		"^node_count$",
		"^members_memory_allocation_mb$",
		"^node_memory_allocation_mb$",
		"^members_disk_allocation_mb$",
		"^members_cpu_allocation_count$",
		"^node_cpu_allocation_count$",
		"^node_disk_allocation_mb$",
	)

	return resource
}

// InitResources ...
func (g *DatabaseElasticSearchGenerator) InitResources() error {
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

	serviceID, err := catalogClient.ResourceCatalog().FindByName("databases-for-elasticsearch", true)
	if err != nil {
		return err
	}
	query := controllerv2.ServiceInstanceQuery{
		ServiceID: serviceID[0].ID,
	}
	elasticSearchInstances, err := controllerClient.ResourceServiceInstanceV2().ListInstances(query)
	if err != nil {
		return err
	}
	for _, db := range elasticSearchInstances {
		if db.RegionID == region {
			g.Resources = append(g.Resources, g.loadElasticSearchDB(db.ID, db.Name))
		}
	}

	return nil
}
