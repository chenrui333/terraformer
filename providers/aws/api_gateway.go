// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/terraformerstring"
)

const (
	apiGatewayAccountResourceType              = "aws_api_gateway_account"
	apiGatewayBasePathMappingResourceType      = "aws_api_gateway_base_path_mapping"
	apiGatewayClientCertificateResourceType    = "aws_api_gateway_client_certificate"
	apiGatewayDocumentationVersionResourceType = "aws_api_gateway_documentation_version"
	apiGatewayRequestValidatorResourceType     = "aws_api_gateway_request_validator"
	apiGatewayResourceResourceType             = "aws_api_gateway_resource"
	apiGatewayUsagePlanKeyResourceType         = "aws_api_gateway_usage_plan_key"
	apiGatewayEmptyBasePathMappingValue        = "(none)"
	apiGatewayResourceIDSeparator              = "/"
)

var apiGatewayAllowEmptyValues = []string{"tags.", "base_path", "parent_id", "path_part"}

type apiGatewayOptionalResourceLoader struct {
	name string
	load func() error
}

type APIGatewayGenerator struct {
	AWSService
	acceptedRestAPIIDs map[string]struct{}
}

func (g *APIGatewayGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := apigateway.NewFromConfig(config)

	g.loadOptionalResources([]apiGatewayOptionalResourceLoader{
		{name: "account", load: func() error { return g.loadAccount(svc) }},
		{name: "client certificates", load: func() error { return g.loadClientCertificates(svc) }},
	})

	if err := g.loadRestApis(svc); err != nil {
		return err
	}
	g.loadOptionalResources([]apiGatewayOptionalResourceLoader{
		{name: "base path mappings", load: func() error { return g.loadDomainBasePathMappings(svc) }},
	})
	if err := g.loadVpcLinks(svc); err != nil {
		return err
	}
	if err := g.loadUsagePlans(svc); err != nil {
		return err
	}
	if err := g.loadAPIKeys(svc); err != nil {
		return err
	}

	return nil
}

func (g *APIGatewayGenerator) loadOptionalResources(loaders []apiGatewayOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if apiGatewayResourceMissing(err) {
				continue
			}
			log.Printf("Skipping API Gateway %s: %v", loader.name, err)
		}
	}
}

func (g *APIGatewayGenerator) loadRestApis(svc *apigateway.Client) error {
	p := apigateway.NewGetRestApisPaginator(svc, &apigateway.GetRestApisInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, restAPI := range page.Items {
			if g.shouldFilterRestAPI(restAPI) {
				continue
			}
			restAPIID := StringValue(restAPI.Id)
			if restAPIID == "" {
				continue
			}
			restAPIName := StringValue(restAPI.Name)
			if restAPIName == "" {
				restAPIName = restAPIID
			}
			g.rememberAcceptedRestAPIID(restAPIID)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				restAPIID,
				restAPIID+"_"+restAPIName,
				"aws_api_gateway_rest_api",
				"aws",
				apiGatewayAllowEmptyValues))
			if err := g.loadStages(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadResources(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadModels(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadResponses(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadDocumentationParts(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadDocumentationVersions(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadAuthorizers(svc, restAPI.Id); err != nil {
				return err
			}
			if err := g.loadRequestValidators(svc, restAPI.Id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) shouldFilterRestAPI(restAPI types.RestApi) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("api_gateway_rest_api") {
			if !terraformerstring.ContainsString(filter.AcceptableValues, StringValue(restAPI.Id)) {
				return true
			}
			continue
		}
		if strings.HasPrefix(filter.FieldPath, "tags.") && filter.IsApplicable("api_gateway_rest_api") {
			tagName := strings.Replace(filter.FieldPath, "tags.", "", 1)
			if val, ok := restAPI.Tags[tagName]; ok {
				if !terraformerstring.ContainsString(filter.AcceptableValues, val) {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

func (g *APIGatewayGenerator) hasRestAPIFilter() bool {
	for _, filter := range g.Filter {
		if filter.IsApplicable("api_gateway_rest_api") &&
			(filter.FieldPath == "id" || strings.HasPrefix(filter.FieldPath, "tags.")) {
			return true
		}
	}
	return false
}

func (g *APIGatewayGenerator) rememberAcceptedRestAPIID(restAPIID string) {
	if restAPIID == "" {
		return
	}
	if g.acceptedRestAPIIDs == nil {
		g.acceptedRestAPIIDs = map[string]struct{}{}
	}
	g.acceptedRestAPIIDs[restAPIID] = struct{}{}
}

func (g *APIGatewayGenerator) shouldFilterBasePathMapping(mapping types.BasePathMapping) bool {
	if !g.hasRestAPIFilter() {
		return false
	}
	restAPIID := StringValue(mapping.RestApiId)
	if restAPIID == "" {
		return true
	}
	_, ok := g.acceptedRestAPIIDs[restAPIID]
	return !ok
}

func (g *APIGatewayGenerator) loadAccount(svc *apigateway.Client) error {
	output, err := svc.GetAccount(context.TODO(), &apigateway.GetAccountInput{})
	if err != nil {
		return err
	}
	cloudwatchRoleARN := ""
	if output != nil {
		cloudwatchRoleARN = StringValue(output.CloudwatchRoleArn)
	}
	if cloudwatchRoleARN == "" {
		return nil
	}
	accountID, ok := apiGatewayAccountIDFromRoleARN(cloudwatchRoleARN)
	if !ok {
		log.Printf("Skipping API Gateway account: unable to parse account ID from CloudWatch role ARN %q", cloudwatchRoleARN)
		return nil
	}
	if resource, ok := newAPIGatewayAccountResource(accountID, output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *APIGatewayGenerator) loadStages(svc *apigateway.Client, restAPIID *string) error {
	output, err := svc.GetStages(context.TODO(), &apigateway.GetStagesInput{
		RestApiId: restAPIID,
	})
	if err != nil {
		return err
	}
	for _, stage := range output.Item {
		stageID := *restAPIID + "/" + StringValue(stage.StageName)
		g.Resources = append(g.Resources, terraformutils.NewResource(
			stageID,
			stageID,
			"aws_api_gateway_stage",
			"aws",
			map[string]string{
				"rest_api_id": *restAPIID,
				"stage_name":  *stage.StageName,
			},
			apiGatewayAllowEmptyValues,
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *APIGatewayGenerator) loadResources(svc *apigateway.Client, restAPIID *string) error {
	p := apigateway.NewGetResourcesPaginator(svc, &apigateway.GetResourcesInput{
		RestApiId: restAPIID,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, resource := range page.Items {
			if tfResource, ok := newAPIGatewayResourceResource(StringValue(restAPIID), resource); ok {
				g.Resources = append(g.Resources, tfResource)
			}
			err := g.loadResourceMethods(svc, restAPIID, resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadModels(svc *apigateway.Client, restAPIID *string) error {
	p := apigateway.NewGetModelsPaginator(svc, &apigateway.GetModelsInput{
		RestApiId: restAPIID,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, model := range page.Items {
			resourceID := *restAPIID + "/" + *model.Id
			g.Resources = append(g.Resources, terraformutils.NewResource(
				resourceID,
				resourceID,
				"aws_api_gateway_model",
				"aws",
				map[string]string{
					"name":         StringValue(model.Name),
					"content_type": StringValue(model.ContentType),
					"schema":       StringValue(model.Schema),
					"rest_api_id":  StringValue(restAPIID),
				},
				apiGatewayAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadResourceMethods(svc *apigateway.Client, restAPIID *string, resource types.Resource) error {
	for httpMethod, method := range resource.ResourceMethods {
		methodID := *restAPIID + "/" + *resource.Id + "/" + httpMethod
		authorizationType := "NONE"
		if method.AuthorizationType != nil {
			authorizationType = *method.AuthorizationType
		}

		g.Resources = append(g.Resources, terraformutils.NewResource(
			methodID,
			methodID,
			"aws_api_gateway_method",
			"aws",
			map[string]string{
				"rest_api_id":   *restAPIID,
				"resource_id":   *resource.Id,
				"http_method":   httpMethod,
				"authorization": authorizationType,
			},
			apiGatewayAllowEmptyValues,
			map[string]interface{}{},
		))

		methodDetails, err := svc.GetMethod(context.TODO(), &apigateway.GetMethodInput{
			HttpMethod: &httpMethod,
			ResourceId: resource.Id,
			RestApiId:  restAPIID,
		})
		if err != nil {
			return err
		}

		if methodDetails.MethodIntegration != nil {
			typeString := string(methodDetails.MethodIntegration.Type)
			g.Resources = append(g.Resources, terraformutils.NewResource(
				methodID,
				methodID,
				"aws_api_gateway_integration",
				"aws",
				map[string]string{
					"rest_api_id": *restAPIID,
					"resource_id": *resource.Id,
					"http_method": httpMethod,
					"type":        typeString,
				},
				apiGatewayAllowEmptyValues,
				map[string]interface{}{},
			))
			integrationDetails, err := svc.GetIntegration(context.TODO(), &apigateway.GetIntegrationInput{
				HttpMethod: &httpMethod,
				ResourceId: resource.Id,
				RestApiId:  restAPIID,
			})
			if err != nil {
				return err
			}

			for responseCode := range integrationDetails.IntegrationResponses {
				integrationResponseID := *restAPIID + "/" + *resource.Id + "/" + httpMethod + "/" + responseCode
				g.Resources = append(g.Resources, terraformutils.NewResource(
					integrationResponseID,
					integrationResponseID,
					"aws_api_gateway_integration_response",
					"aws",
					map[string]string{
						"rest_api_id": *restAPIID,
						"resource_id": *resource.Id,
						"http_method": httpMethod,
						"status_code": responseCode,
					},
					apiGatewayAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
		for responseCode := range methodDetails.MethodResponses {
			responseID := *restAPIID + "/" + *resource.Id + "/" + httpMethod + "/" + responseCode

			g.Resources = append(g.Resources, terraformutils.NewResource(
				responseID,
				responseID,
				"aws_api_gateway_method_response",
				"aws",
				map[string]string{
					"rest_api_id": *restAPIID,
					"resource_id": *resource.Id,
					"http_method": httpMethod,
					"status_code": responseCode,
				},
				apiGatewayAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadResponses(svc *apigateway.Client, restAPIID *string) error {
	var position *string
	for {
		response, err := svc.GetGatewayResponses(context.TODO(), &apigateway.GetGatewayResponsesInput{
			RestApiId: restAPIID,
			Position:  position,
		})
		if err != nil {
			return err
		}
		for _, response := range response.Items {
			if response.DefaultResponse {
				continue
			}
			responseTypeString := string(response.ResponseType)
			responseID := *restAPIID + "/" + responseTypeString
			g.Resources = append(g.Resources, terraformutils.NewResource(
				responseID,
				responseID,
				"aws_api_gateway_gateway_response",
				"aws",
				map[string]string{
					"rest_api_id":   *restAPIID,
					"response_type": responseTypeString,
				},
				apiGatewayAllowEmptyValues,
				map[string]interface{}{},
			))
		}
		position = response.Position
		if position == nil {
			break
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadDocumentationParts(svc *apigateway.Client, restAPIID *string) error {
	var position *string
	for {
		response, err := svc.GetDocumentationParts(context.TODO(), &apigateway.GetDocumentationPartsInput{
			RestApiId: restAPIID,
			Position:  position,
		})
		if err != nil {
			return err
		}
		for _, documentationPart := range response.Items {
			documentationPartID := *restAPIID + "/" + *documentationPart.Id
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				documentationPartID,
				documentationPartID,
				"aws_api_gateway_documentation_part",
				"aws",
				apiGatewayAllowEmptyValues,
			))
		}
		position = response.Position
		if position == nil {
			break
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadDocumentationVersions(svc *apigateway.Client, restAPIID *string) error {
	var position *string
	for {
		response, err := svc.GetDocumentationVersions(context.TODO(), &apigateway.GetDocumentationVersionsInput{
			RestApiId: restAPIID,
			Position:  position,
		})
		if err != nil {
			return err
		}
		for _, documentationVersion := range response.Items {
			if resource, ok := newAPIGatewayDocumentationVersionResource(StringValue(restAPIID), documentationVersion); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		position = response.Position
		if position == nil {
			break
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadAuthorizers(svc *apigateway.Client, restAPIID *string) error {
	var position *string
	for {
		response, err := svc.GetAuthorizers(context.TODO(), &apigateway.GetAuthorizersInput{
			RestApiId: restAPIID,
			Position:  position,
		})
		if err != nil {
			return err
		}
		for _, authorizer := range response.Items {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				*authorizer.Id,
				*authorizer.Id,
				"aws_api_gateway_authorizer",
				"aws",
				map[string]string{
					"rest_api_id": *restAPIID,
					"name":        StringValue(authorizer.Name),
				},
				apiGatewayAllowEmptyValues,
				map[string]interface{}{},
			))
		}
		position = response.Position
		if position == nil {
			break
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadRequestValidators(svc *apigateway.Client, restAPIID *string) error {
	var position *string
	for {
		response, err := svc.GetRequestValidators(context.TODO(), &apigateway.GetRequestValidatorsInput{
			RestApiId: restAPIID,
			Position:  position,
		})
		if err != nil {
			return err
		}
		for _, validator := range response.Items {
			if resource, ok := newAPIGatewayRequestValidatorResource(StringValue(restAPIID), validator); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		position = response.Position
		if position == nil {
			break
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadVpcLinks(svc *apigateway.Client) error {
	p := apigateway.NewGetVpcLinksPaginator(svc, &apigateway.GetVpcLinksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, vpcLink := range page.Items {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*vpcLink.Id,
				*vpcLink.Name,
				"aws_api_gateway_vpc_link",
				"aws",
				apiGatewayAllowEmptyValues))
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadUsagePlans(svc *apigateway.Client) error {
	p := apigateway.NewGetUsagePlansPaginator(svc, &apigateway.GetUsagePlansInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, usagePlan := range page.Items {
			usagePlanID := StringValue(usagePlan.Id)
			if usagePlanID == "" {
				continue
			}
			usagePlanName := StringValue(usagePlan.Name)
			if usagePlanName == "" {
				usagePlanName = usagePlanID
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				usagePlanID,
				usagePlanName,
				"aws_api_gateway_usage_plan",
				"aws",
				apiGatewayAllowEmptyValues))
			if err := g.loadUsagePlanKeys(svc, aws.String(usagePlanID)); err != nil {
				if !apiGatewayResourceMissing(err) {
					log.Printf("Skipping API Gateway usage plan keys for %s: %v", usagePlanID, err)
				}
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadUsagePlanKeys(svc *apigateway.Client, usagePlanID *string) error {
	usagePlanIDValue := StringValue(usagePlanID)
	if usagePlanIDValue == "" {
		return nil
	}
	p := apigateway.NewGetUsagePlanKeysPaginator(svc, &apigateway.GetUsagePlanKeysInput{
		UsagePlanId: aws.String(usagePlanIDValue),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, usagePlanKey := range page.Items {
			if resource, ok := newAPIGatewayUsagePlanKeyResource(usagePlanIDValue, usagePlanKey); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadAPIKeys(svc *apigateway.Client) error {
	p := apigateway.NewGetApiKeysPaginator(svc, &apigateway.GetApiKeysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, apiKey := range page.Items {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*apiKey.Id,
				*apiKey.Name,
				"aws_api_gateway_api_key",
				"aws",
				apiGatewayAllowEmptyValues))
		}
	}

	return nil
}

func (g *APIGatewayGenerator) loadClientCertificates(svc *apigateway.Client) error {
	p := apigateway.NewGetClientCertificatesPaginator(svc, &apigateway.GetClientCertificatesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, certificate := range page.Items {
			if resource, ok := newAPIGatewayClientCertificateResource(certificate); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadDomainBasePathMappings(svc *apigateway.Client) error {
	p := apigateway.NewGetDomainNamesPaginator(svc, &apigateway.GetDomainNamesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, domainName := range page.Items {
			domainNameValue := StringValue(domainName.DomainName)
			if domainNameValue == "" {
				continue
			}
			if err := g.loadBasePathMappings(svc, domainNameValue, StringValue(domainName.DomainNameId)); err != nil {
				if !apiGatewayResourceMissing(err) {
					log.Printf("Skipping API Gateway base path mappings for %s: %v", domainNameValue, err)
				}
			}
		}
	}
	return nil
}

func (g *APIGatewayGenerator) loadBasePathMappings(svc *apigateway.Client, domainName, domainNameID string) error {
	input := &apigateway.GetBasePathMappingsInput{
		DomainName: aws.String(domainName),
	}
	if domainNameID != "" {
		input.DomainNameId = aws.String(domainNameID)
	}
	p := apigateway.NewGetBasePathMappingsPaginator(svc, input)
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, mapping := range page.Items {
			if g.shouldFilterBasePathMapping(mapping) {
				continue
			}
			if resource, ok := newAPIGatewayBasePathMappingResource(domainName, domainNameID, mapping); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newAPIGatewayAccountResource(accountID string, output *apigateway.GetAccountOutput) (terraformutils.Resource, bool) {
	cloudwatchRoleARN := ""
	if output != nil {
		cloudwatchRoleARN = StringValue(output.CloudwatchRoleArn)
	}
	if accountID == "" || cloudwatchRoleARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		apiGatewayAccountImportID(accountID),
		apiGatewayResourceName("account", accountID),
		apiGatewayAccountResourceType,
		"aws",
		map[string]string{
			"cloudwatch_role_arn": cloudwatchRoleARN,
		},
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayBasePathMappingResource(domainName, domainNameID string, mapping types.BasePathMapping) (terraformutils.Resource, bool) {
	basePath := apiGatewayNormalizeBasePathMappingValue(StringValue(mapping.BasePath))
	if domainName == "" || StringValue(mapping.RestApiId) == "" || !apiGatewayBasePathMappingImportable(basePath) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"api_id":      StringValue(mapping.RestApiId),
		"base_path":   basePath,
		"domain_name": domainName,
	}
	if domainNameID != "" {
		attributes["domain_name_id"] = domainNameID
	}
	if stageName := StringValue(mapping.Stage); stageName != "" {
		attributes["stage_name"] = stageName
	}
	return terraformutils.NewResource(
		apiGatewayBasePathMappingImportID(domainName, basePath, domainNameID),
		apiGatewayResourceName("base_path_mapping", domainName, basePath, domainNameID),
		apiGatewayBasePathMappingResourceType,
		"aws",
		attributes,
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayClientCertificateResource(certificate types.ClientCertificate) (terraformutils.Resource, bool) {
	certificateID := StringValue(certificate.ClientCertificateId)
	if certificateID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{}
	if description := StringValue(certificate.Description); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		apiGatewayClientCertificateImportID(certificateID),
		apiGatewayResourceName("client_certificate", certificateID),
		apiGatewayClientCertificateResourceType,
		"aws",
		attributes,
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayDocumentationVersionResource(restAPIID string, documentationVersion types.DocumentationVersion) (terraformutils.Resource, bool) {
	version := StringValue(documentationVersion.Version)
	if restAPIID == "" || version == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"rest_api_id": restAPIID,
		"version":     version,
	}
	if description := StringValue(documentationVersion.Description); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		apiGatewayDocumentationVersionImportID(restAPIID, version),
		apiGatewayResourceName("documentation_version", restAPIID, version),
		apiGatewayDocumentationVersionResourceType,
		"aws",
		attributes,
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayRequestValidatorResource(restAPIID string, validator types.RequestValidator) (terraformutils.Resource, bool) {
	validatorID := StringValue(validator.Id)
	validatorName := StringValue(validator.Name)
	if restAPIID == "" || validatorID == "" || validatorName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		apiGatewayRequestValidatorStateID(validatorID),
		apiGatewayResourceName("request_validator", restAPIID, validatorID),
		apiGatewayRequestValidatorResourceType,
		"aws",
		map[string]string{
			"name":                        validatorName,
			"rest_api_id":                 restAPIID,
			"validate_request_body":       strconv.FormatBool(validator.ValidateRequestBody),
			"validate_request_parameters": strconv.FormatBool(validator.ValidateRequestParameters),
		},
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayResourceResource(restAPIID string, resource types.Resource) (terraformutils.Resource, bool) {
	resourceID := StringValue(resource.Id)
	parentID := StringValue(resource.ParentId)
	pathPart := StringValue(resource.PathPart)
	if restAPIID == "" || resourceID == "" || parentID == "" || pathPart == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		apiGatewayResourceStateID(resourceID),
		apiGatewayResourceName("resource", restAPIID, resourceID),
		apiGatewayResourceResourceType,
		"aws",
		map[string]string{
			"path":        StringValue(resource.Path),
			"path_part":   pathPart,
			"parent_id":   parentID,
			"rest_api_id": restAPIID,
		},
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newAPIGatewayUsagePlanKeyResource(usagePlanID string, usagePlanKey types.UsagePlanKey) (terraformutils.Resource, bool) {
	keyID := StringValue(usagePlanKey.Id)
	keyType := StringValue(usagePlanKey.Type)
	if usagePlanID == "" || keyID == "" || keyType == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		apiGatewayUsagePlanKeyStateID(keyID),
		apiGatewayResourceName("usage_plan_key", usagePlanID, keyID),
		apiGatewayUsagePlanKeyResourceType,
		"aws",
		map[string]string{
			"key_id":        keyID,
			"key_type":      keyType,
			"usage_plan_id": usagePlanID,
		},
		apiGatewayAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func apiGatewayResourceMissing(err error) bool {
	var notFound *types.NotFoundException
	return errors.As(err, &notFound)
}

func apiGatewayAccountImportID(accountID string) string {
	return accountID
}

func apiGatewayAccountIDFromRoleARN(roleARN string) (string, bool) {
	parsedARN, err := arn.Parse(roleARN)
	if err != nil || !apiGatewayAWSAccountID(parsedARN.AccountID) {
		return "", false
	}
	return parsedARN.AccountID, true
}

func apiGatewayAWSAccountID(accountID string) bool {
	if len(accountID) != 12 {
		return false
	}
	for _, r := range accountID {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func apiGatewayBasePathMappingImportID(domainName, basePath, domainNameID string) string {
	parts := []string{domainName, basePath}
	if domainNameID != "" {
		parts = append(parts, domainNameID)
	}
	return strings.Join(parts, apiGatewayResourceIDSeparator)
}

func apiGatewayClientCertificateImportID(certificateID string) string {
	return certificateID
}

func apiGatewayDocumentationVersionImportID(restAPIID, version string) string {
	return strings.Join([]string{restAPIID, version}, apiGatewayResourceIDSeparator)
}

func apiGatewayRequestValidatorImportID(restAPIID, validatorID string) string {
	return strings.Join([]string{restAPIID, validatorID}, apiGatewayResourceIDSeparator)
}

// Terraformer seeds provider state directly before refresh. These IDs match the
// Terraform AWS provider read IDs after its importers parse composite import IDs
// and seed the required parent attributes.
func apiGatewayRequestValidatorStateID(validatorID string) string {
	return validatorID
}

func apiGatewayResourceStateID(resourceID string) string {
	return resourceID
}

func apiGatewayUsagePlanKeyImportID(usagePlanID, keyID string) string {
	return strings.Join([]string{usagePlanID, keyID}, apiGatewayResourceIDSeparator)
}

func apiGatewayUsagePlanKeyStateID(keyID string) string {
	return keyID
}

func apiGatewayNormalizeBasePathMappingValue(basePath string) string {
	if basePath == apiGatewayEmptyBasePathMappingValue {
		return ""
	}
	return basePath
}

func apiGatewayBasePathMappingImportable(basePath string) bool {
	return !strings.Contains(basePath, apiGatewayResourceIDSeparator)
}

func apiGatewayResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}
