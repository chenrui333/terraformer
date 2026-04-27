// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"os"
	"time"

	"github.com/IBM/go-sdk-core/v4/core"
	tg "github.com/IBM/networking-go-sdk/transitgatewayapisv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// TGGenerator ...
type TGGenerator struct {
	IBMService
}

func (g TGGenerator) createTransitGatewayResources(gatewayID, gatewayName string) terraformutils.Resource {
	resource := terraformutils.NewSimpleResource(
		gatewayID,
		normalizeResourceName(gatewayName, false),
		"ibm_tg_gateway",
		"ibm",
		[]string{})
	return resource
}

func (g TGGenerator) createTransitGatewayConnectionResources(gatewayID, connectionID, connectionName string, dependsOn []string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", gatewayID, connectionID),
		normalizeResourceName(connectionName, false),
		"ibm_tg_connection",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{
			"depends_on": dependsOn,
		})
	return resource
}

func (g TGGenerator) loadTransitGatewayRouterResource(gatewayID, routerID string, dependsOn []string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", gatewayID, routerID),
		normalizeResourceName(routerID, false),
		"ibm_tg_route_report",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{
			"depends_on": dependsOn,
		})
	return resource
}

// CreateVersionDate requires mandatory version attribute. Any date from 2019-12-13 up to the currentdate may be provided. Specify the current date to request the latest version.
func CreateVersionDate() *string {
	version := time.Now().Format("2006-01-02")
	return &version
}

// InitResources ...
func (g *TGGenerator) InitResources() error {
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("no API key set")
	}
	tgURL := "https://transit.cloud.ibm.com/v1"
	transitgatewayOptions := &tg.TransitGatewayApisV1Options{
		URL: envFallBack([]string{"IBMCLOUD_TG_API_ENDPOINT"}, tgURL),
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
		},
		Version: CreateVersionDate(),
	}

	tgclient, err := tg.NewTransitGatewayApisV1(transitgatewayOptions)
	if err != nil {
		return err
	}
	start := ""
	allrecs := []tg.TransitGateway{}
	for {
		listTransitGatewaysOptions := &tg.ListTransitGatewaysOptions{}
		if start != "" {
			listTransitGatewaysOptions.Start = &start
		}

		gateways, resp, err := tgclient.ListTransitGateways(listTransitGatewaysOptions)
		if err != nil {
			return fmt.Errorf("error listing Transit Gateways %w\n%s", err, resp)
		}
		start = GetNext(gateways.Next)
		allrecs = append(allrecs, gateways.TransitGateways...)
		if start == "" {
			break
		}
	}
	for _, gateway := range allrecs {
		g.Resources = append(g.Resources, g.createTransitGatewayResources(*gateway.ID, *gateway.Name))
		resourceName := g.Resources[len(g.Resources)-1:][0].ResourceName
		var dependsOn []string
		dependsOn = append(dependsOn,
			"ibm_tg_gateway."+resourceName)
		listTransitGatewayConnectionsOptions := &tg.ListTransitGatewayConnectionsOptions{
			TransitGatewayID: gateway.ID,
		}
		connections, response, err := tgclient.ListTransitGatewayConnections(listTransitGatewayConnectionsOptions)
		if err != nil {
			return fmt.Errorf("error listing Transit Gateway connections %w\n%s", err, response)
		}
		for _, connection := range connections.Connections {
			g.Resources = append(g.Resources, g.createTransitGatewayConnectionResources(*gateway.ID, *connection.ID, *connection.Name, dependsOn))
		}
		// Trying to get Transit Gateway reports
		listTransitGatewayRouteReportOptions := &tg.ListTransitGatewayRouteReportsOptions{
			TransitGatewayID: gateway.ID,
		}
		routeReports, response, err := tgclient.ListTransitGatewayRouteReports(listTransitGatewayRouteReportOptions)
		if err != nil {
			return fmt.Errorf("error listing Transit Gateway route reports %w\n%s", err, response)
		}
		for _, routeReport := range routeReports.RouteReports {
			g.Resources = append(g.Resources, g.loadTransitGatewayRouterResource(*gateway.ID, *routeReport.ID, dependsOn))
		}
	}
	return nil
}
