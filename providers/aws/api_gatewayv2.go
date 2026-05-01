// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var apiGatewayV2AllowEmptyValues = []string{"tags."}

type apiGatewayV2OptionalResourceLoader struct {
	name string
	load func() error
}

type APIGatewayV2Generator struct {
	AWSService
}

func (g *APIGatewayV2Generator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := apigatewayv2.NewFromConfig(config)

	if err := g.loadRestApis(svc); err != nil {
		return err
	}
	if err := g.loadVpcLinks(svc); err != nil {
		return err
	}
	if err := g.loadDomainNames(svc); err != nil {
		if !apiGatewayV2ResourceMissing(err) {
			log.Printf("Skipping API Gateway V2 domain names: %v", err)
		}
	}
	return nil
}

func (g *APIGatewayV2Generator) loadOptionalResources(loaders []apiGatewayV2OptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if apiGatewayV2ResourceMissing(err) {
				continue
			}
			log.Printf("Skipping API Gateway V2 %s: %v", loader.name, err)
		}
	}
}

func (g *APIGatewayV2Generator) loadRestApis(svc *apigatewayv2.Client) error {
	output, err := svc.GetApis(context.TODO(), &apigatewayv2.GetApisInput{})
	if err != nil {
		return err
	}
	g.processRestApis(svc, output.Items)

	for output.NextToken != nil {
		output, err = svc.GetApis(context.TODO(), &apigatewayv2.GetApisInput{
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processRestApis(svc, output.Items)
	}

	return nil
}

func (g *APIGatewayV2Generator) processRestApis(svc *apigatewayv2.Client, output []types.Api) {
	for _, restAPI := range output {
		apiID := StringValue(restAPI.ApiId)
		if apiID == "" {
			continue
		}

		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			apiID,
			apiGatewayV2ResourceName(apiID, StringValue(restAPI.Name)),
			"aws_apigatewayv2_api",
			"aws",
			apiGatewayV2AllowEmptyValues,
		))

		g.loadOptionalResources([]apiGatewayV2OptionalResourceLoader{
			{name: "stages", load: func() error { return g.loadStages(svc, apiID) }},
			{name: "models", load: func() error { return g.loadModels(svc, apiID) }},
			{name: "routes", load: func() error { return g.loadRoutes(svc, apiID) }},
			{name: "integrations", load: func() error { return g.loadIntegrations(svc, apiID) }},
			{name: "deployments", load: func() error { return g.loadDeployments(svc, apiID) }},
			{name: "authorizers", load: func() error { return g.loadAuthorizers(svc, apiID) }},
		})
	}
}

func (g *APIGatewayV2Generator) loadStages(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetStages(context.TODO(), &apigatewayv2.GetStagesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processStages(output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetStages(context.TODO(), &apigatewayv2.GetStagesInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processStages(output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processStages(output []types.Stage, apiID string) {
	for _, stage := range output {
		if apiGatewayV2Managed(stage.ApiGatewayManaged) {
			continue
		}
		stageName := StringValue(stage.StageName)
		if stageName == "" {
			continue
		}
		stageID := apiGatewayV2ImportID(apiID, stageName)
		g.Resources = append(g.Resources, terraformutils.NewResource(
			stageID,
			stageID,
			"aws_apigatewayv2_stage",
			"aws",
			map[string]string{
				"api_id": apiID,
				"name":   stageName,
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadModels(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetModels(context.TODO(), &apigatewayv2.GetModelsInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processModels(output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetModels(context.TODO(), &apigatewayv2.GetModelsInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processModels(output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processModels(output []types.Model, apiID string) {
	for _, model := range output {
		modelID := StringValue(model.ModelId)
		if modelID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, modelID),
			apiGatewayV2ResourceName(apiID, modelID),
			"aws_apigatewayv2_model",
			"aws",
			map[string]string{
				"api_id":       apiID,
				"content_type": StringValue(model.ContentType),
				"name":         StringValue(model.Name),
				"schema":       StringValue(model.Schema),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadRoutes(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetRoutes(context.TODO(), &apigatewayv2.GetRoutesInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processRoutes(svc, output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetRoutes(context.TODO(), &apigatewayv2.GetRoutesInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processRoutes(svc, output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processRoutes(svc *apigatewayv2.Client, output []types.Route, apiID string) {
	for _, route := range output {
		if apiGatewayV2Managed(route.ApiGatewayManaged) {
			continue
		}
		routeID := StringValue(route.RouteId)
		if routeID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, routeID),
			apiGatewayV2ResourceName(apiID, routeID),
			"aws_apigatewayv2_route",
			"aws",
			map[string]string{
				"api_id":    apiID,
				"route_key": StringValue(route.RouteKey),
				"target":    StringValue(route.Target),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
		if err := g.loadRouteResponses(svc, apiID, routeID); err != nil {
			if !apiGatewayV2ResourceMissing(err) {
				log.Printf("Skipping API Gateway V2 route responses for %s/%s: %v", apiID, routeID, err)
			}
		}
	}
}

func (g *APIGatewayV2Generator) loadRouteResponses(svc *apigatewayv2.Client, apiID string, routeID string) error {
	output, err := svc.GetRouteResponses(context.TODO(), &apigatewayv2.GetRouteResponsesInput{
		ApiId:   aws.String(apiID),
		RouteId: aws.String(routeID),
	})
	if err != nil {
		return err
	}
	g.processRouteResponses(output.Items, apiID, routeID)

	for output.NextToken != nil {
		output, err = svc.GetRouteResponses(context.TODO(), &apigatewayv2.GetRouteResponsesInput{
			ApiId:     aws.String(apiID),
			RouteId:   aws.String(routeID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processRouteResponses(output.Items, apiID, routeID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processRouteResponses(output []types.RouteResponse, apiID string, routeID string) {
	for _, response := range output {
		responseID := StringValue(response.RouteResponseId)
		if responseID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, routeID, responseID),
			apiGatewayV2ResourceName(apiID, routeID, responseID),
			"aws_apigatewayv2_route_response",
			"aws",
			map[string]string{
				"api_id":             apiID,
				"route_id":           routeID,
				"route_response_key": StringValue(response.RouteResponseKey),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadIntegrations(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetIntegrations(context.TODO(), &apigatewayv2.GetIntegrationsInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processIntegrations(svc, output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetIntegrations(context.TODO(), &apigatewayv2.GetIntegrationsInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processIntegrations(svc, output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processIntegrations(svc *apigatewayv2.Client, output []types.Integration, apiID string) {
	for _, integration := range output {
		if apiGatewayV2Managed(integration.ApiGatewayManaged) {
			continue
		}
		integrationID := StringValue(integration.IntegrationId)
		if integrationID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, integrationID),
			apiGatewayV2ResourceName(apiID, integrationID),
			"aws_apigatewayv2_integration",
			"aws",
			map[string]string{
				"api_id":             apiID,
				"integration_method": StringValue(integration.IntegrationMethod),
				"integration_type":   string(integration.IntegrationType),
				"integration_uri":    StringValue(integration.IntegrationUri),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
		if err := g.loadIntegrationResponses(svc, apiID, integrationID); err != nil {
			if !apiGatewayV2ResourceMissing(err) {
				log.Printf("Skipping API Gateway V2 integration responses for %s/%s: %v", apiID, integrationID, err)
			}
		}
	}
}

func (g *APIGatewayV2Generator) loadIntegrationResponses(svc *apigatewayv2.Client, apiID string, integrationID string) error {
	output, err := svc.GetIntegrationResponses(context.TODO(), &apigatewayv2.GetIntegrationResponsesInput{
		ApiId:         aws.String(apiID),
		IntegrationId: aws.String(integrationID),
	})
	if err != nil {
		return err
	}
	g.processIntegrationResponses(output.Items, apiID, integrationID)

	for output.NextToken != nil {
		output, err = svc.GetIntegrationResponses(context.TODO(), &apigatewayv2.GetIntegrationResponsesInput{
			ApiId:         aws.String(apiID),
			IntegrationId: aws.String(integrationID),
			NextToken:     output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processIntegrationResponses(output.Items, apiID, integrationID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processIntegrationResponses(output []types.IntegrationResponse, apiID string, integrationID string) {
	for _, response := range output {
		responseID := StringValue(response.IntegrationResponseId)
		if responseID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, integrationID, responseID),
			apiGatewayV2ResourceName(apiID, integrationID, responseID),
			"aws_apigatewayv2_integration_response",
			"aws",
			map[string]string{
				"api_id":                   apiID,
				"integration_id":           integrationID,
				"integration_response_key": StringValue(response.IntegrationResponseKey),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadDeployments(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetDeployments(context.TODO(), &apigatewayv2.GetDeploymentsInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processDeployments(output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetDeployments(context.TODO(), &apigatewayv2.GetDeploymentsInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processDeployments(output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processDeployments(output []types.Deployment, apiID string) {
	for _, deployment := range output {
		deploymentID := StringValue(deployment.DeploymentId)
		if deploymentID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, deploymentID),
			apiGatewayV2ResourceName(apiID, deploymentID),
			"aws_apigatewayv2_deployment",
			"aws",
			map[string]string{
				"api_id": apiID,
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadAuthorizers(svc *apigatewayv2.Client, apiID string) error {
	output, err := svc.GetAuthorizers(context.TODO(), &apigatewayv2.GetAuthorizersInput{
		ApiId: aws.String(apiID),
	})
	if err != nil {
		return err
	}
	g.processAuthorizers(output.Items, apiID)

	for output.NextToken != nil {
		output, err = svc.GetAuthorizers(context.TODO(), &apigatewayv2.GetAuthorizersInput{
			ApiId:     aws.String(apiID),
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processAuthorizers(output.Items, apiID)
	}

	return nil
}

func (g *APIGatewayV2Generator) processAuthorizers(output []types.Authorizer, apiID string) {
	for _, authorizer := range output {
		authorizerID := StringValue(authorizer.AuthorizerId)
		if authorizerID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(apiID, authorizerID),
			apiGatewayV2ResourceName(apiID, authorizerID),
			"aws_apigatewayv2_authorizer",
			"aws",
			map[string]string{
				"api_id":          apiID,
				"authorizer_type": string(authorizer.AuthorizerType),
				"name":            StringValue(authorizer.Name),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func (g *APIGatewayV2Generator) loadVpcLinks(svc *apigatewayv2.Client) error {
	output, err := svc.GetVpcLinks(context.TODO(), &apigatewayv2.GetVpcLinksInput{})
	if err != nil {
		return err
	}
	g.processVpcLinks(output.Items)

	for output.NextToken != nil {
		output, err = svc.GetVpcLinks(context.TODO(), &apigatewayv2.GetVpcLinksInput{
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processVpcLinks(output.Items)
	}

	return nil
}

func (g *APIGatewayV2Generator) processVpcLinks(output []types.VpcLink) {
	for _, vpcLink := range output {
		vpcLinkID := StringValue(vpcLink.VpcLinkId)
		if vpcLinkID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			vpcLinkID,
			vpcLinkID,
			"aws_apigatewayv2_vpc_link",
			"aws",
			apiGatewayV2AllowEmptyValues,
		))
	}
}

func (g *APIGatewayV2Generator) loadDomainNames(svc *apigatewayv2.Client) error {
	output, err := svc.GetDomainNames(context.TODO(), &apigatewayv2.GetDomainNamesInput{})
	if err != nil {
		return err
	}
	g.processDomainNames(svc, output.Items)

	for output.NextToken != nil {
		output, err = svc.GetDomainNames(context.TODO(), &apigatewayv2.GetDomainNamesInput{
			NextToken: output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processDomainNames(svc, output.Items)
	}

	return nil
}

func (g *APIGatewayV2Generator) processDomainNames(svc *apigatewayv2.Client, output []types.DomainName) {
	for _, domainName := range output {
		domain := StringValue(domainName.DomainName)
		if domain == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			domain,
			domain,
			"aws_apigatewayv2_domain_name",
			"aws",
			map[string]string{
				"domain_name": domain,
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
		if err := g.loadAPIMappings(svc, domain); err != nil {
			if !apiGatewayV2ResourceMissing(err) {
				log.Printf("Skipping API Gateway V2 API mappings for %s: %v", domain, err)
			}
		}
	}
}

func (g *APIGatewayV2Generator) loadAPIMappings(svc *apigatewayv2.Client, domainName string) error {
	output, err := svc.GetApiMappings(context.TODO(), &apigatewayv2.GetApiMappingsInput{
		DomainName: aws.String(domainName),
	})
	if err != nil {
		return err
	}
	g.processAPIMappings(output.Items, domainName)

	for output.NextToken != nil {
		output, err = svc.GetApiMappings(context.TODO(), &apigatewayv2.GetApiMappingsInput{
			DomainName: aws.String(domainName),
			NextToken:  output.NextToken,
		})
		if err != nil {
			return err
		}
		g.processAPIMappings(output.Items, domainName)
	}

	return nil
}

func (g *APIGatewayV2Generator) processAPIMappings(output []types.ApiMapping, domainName string) {
	for _, mapping := range output {
		mappingID := StringValue(mapping.ApiMappingId)
		if mappingID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			apiGatewayV2ImportID(mappingID, domainName),
			apiGatewayV2ResourceName(domainName, mappingID),
			"aws_apigatewayv2_api_mapping",
			"aws",
			map[string]string{
				"api_id":          StringValue(mapping.ApiId),
				"api_mapping_key": StringValue(mapping.ApiMappingKey),
				"domain_name":     domainName,
				"stage":           StringValue(mapping.Stage),
			},
			apiGatewayV2AllowEmptyValues,
			map[string]interface{}{},
		))
	}
}

func apiGatewayV2ImportID(parts ...string) string {
	return strings.Join(parts, "/")
}

func apiGatewayV2ResourceName(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "/")
}

func apiGatewayV2Managed(managed *bool) bool {
	return managed != nil && *managed
}

func apiGatewayV2ResourceMissing(err error) bool {
	var notFound *types.NotFoundException
	return errors.As(err, &notFound)
}
