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

// LogAnalysisGenerator ..
type WatsonStudioGenerator struct {
	IBMService
}

// loadWatsonStudio ..
func (g WatsonStudioGenerator) loadWatsonStudio(wsID string, wsName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		wsID,
		normalizeResourceName(wsName, false),
		"ibm_resource_instance",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *WatsonStudioGenerator) InitResources() error {
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

	serviceID, err := catalogClient.ResourceCatalog().FindByName("data-science-experience", true)
	if err != nil {
		return err
	}
	query := controllerv2.ServiceInstanceQuery{
		ServiceID: serviceID[0].ID,
	}
	watsonStudioInstances, err := controllerClient.ResourceServiceInstanceV2().ListInstances(query)
	if err != nil {
		return err
	}

	for _, ws := range watsonStudioInstances {
		if ws.RegionID == region {
			g.Resources = append(g.Resources, g.loadWatsonStudio(ws.ID, ws.Name))
		}
	}

	return nil
}
