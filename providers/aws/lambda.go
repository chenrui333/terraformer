// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

var lambdaAllowEmptyValues = []string{"tags."}

const (
	lambdaFunctionRecursionConfigResourceType = "aws_lambda_function_recursion_config"
	lambdaRuntimeManagementConfigResourceType = "aws_lambda_runtime_management_config"
)

type LambdaGenerator struct {
	AWSService
}

type lambdaFunctionReference struct {
	name string
}

type lambdaOptionalResourceLoader struct {
	name string
	load func() error
}

type Statement struct {
	Sid string `json:"Sid"`
}

type Policy struct {
	Version   string       `json:"Version"`
	ID        string       `json:"Id"`
	Statement []*Statement `json:"Statement"`
}

func (g *LambdaGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := lambda.NewFromConfig(config)

	functions, err := g.addFunctions(svc)
	if err != nil {
		return err
	}
	g.getOptionalLambdaResources(
		lambdaOptionalResourceLoader{name: "aliases", load: func() error { return g.addAliases(svc, functions) }},
		lambdaOptionalResourceLoader{name: "function URLs", load: func() error { return g.addFunctionURLs(svc, functions) }},
		lambdaOptionalResourceLoader{name: "provisioned concurrency configs", load: func() error { return g.addProvisionedConcurrencyConfigs(svc, functions) }},
		lambdaOptionalResourceLoader{name: "function recursion configs", load: func() error { return g.addFunctionRecursionConfigs(svc, functions) }},
		lambdaOptionalResourceLoader{name: "runtime management configs", load: func() error { return g.addRuntimeManagementConfigs(svc, functions) }},
		lambdaOptionalResourceLoader{name: "code signing configs", load: func() error { return g.addCodeSigningConfigs(svc) }},
	)
	err = g.addEventSourceMappings(svc)
	if err != nil {
		return err
	}
	err = g.addLayerVersions(svc)
	return err
}

func (g *LambdaGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if _, exist := r.Item["environment"]; !exist {
			continue
		}
		variables := g.Resources[i].Item["environment"].([]interface{})[0].(map[string]interface{})["variables"]
		g.Resources[i].Item["environment"] = []interface{}{
			map[string]interface{}{
				"variables": []map[string]interface{}{variables.(map[string]interface{})},
			},
		}
	}
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_lambda_function_event_invoke_config" {
			continue
		}
		if r.InstanceState.Attributes["maximum_event_age_in_seconds"] == "0" {
			delete(r.Item, "maximum_event_age_in_seconds")
		}
	}
	return nil
}

func (g *LambdaGenerator) getOptionalLambdaResources(loaders ...lambdaOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping Lambda %s discovery: %v", loader.name, err)
		}
	}
}

func (g *LambdaGenerator) addFunctions(svc *lambda.Client) ([]lambdaFunctionReference, error) {
	functions := []lambdaFunctionReference{}
	p := lambda.NewListFunctionsPaginator(svc, &lambda.ListFunctionsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return functions, err
		}
		for _, function := range page.Functions {
			functionARN := StringValue(function.FunctionArn)
			functionName := StringValue(function.FunctionName)
			if functionARN == "" || functionName == "" {
				continue
			}
			functions = append(functions, lambdaFunctionReference{
				name: functionName,
			})
			g.Resources = append(g.Resources, terraformutils.NewResource(
				functionARN,
				functionName,
				"aws_lambda_function",
				"aws",
				map[string]string{
					"function_name": functionName,
				},
				lambdaAllowEmptyValues,
				map[string]interface{}{},
			))

			gp, err := svc.GetPolicy(context.TODO(), &lambda.GetPolicyInput{
				FunctionName: aws.String(functionARN),
			})

			if err != nil {
				// skip ResourceNotFoundException, because there may be only inline policy defined
				var apiErr smithy.APIError
				if !errors.As(err, &apiErr) || apiErr.ErrorCode() != "ResourceNotFoundException" {
					return functions, err
				}
			}

			if gp != nil {
				outputPolicy := *gp.Policy
				var policy Policy
				err = json.Unmarshal([]byte(outputPolicy), &policy)

				if err != nil {
					return functions, err
				}

				for _, statement := range policy.Statement {
					g.Resources = append(g.Resources, terraformutils.NewResource(
						statement.Sid,
						statement.Sid,
						"aws_lambda_permission",
						"aws",
						map[string]string{
							"statement_id":  statement.Sid,
							"function_name": functionARN,
						},
						lambdaAllowEmptyValues,
						map[string]interface{}{},
					))
				}
			}

			pi := lambda.NewListFunctionEventInvokeConfigsPaginator(svc,
				&lambda.ListFunctionEventInvokeConfigsInput{
					FunctionName: &functionName,
				})
			for pi.HasMorePages() {
				piage, err := pi.NextPage(context.TODO())
				if err != nil {
					return functions, err
				}
				for _, functionEventInvokeConfig := range piage.FunctionEventInvokeConfigs {
					functionEventInvokeConfigARN := StringValue(functionEventInvokeConfig.FunctionArn)
					if functionEventInvokeConfigARN == "" {
						continue
					}
					g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
						functionARN,
						"feic_"+functionEventInvokeConfigARN,
						"aws_lambda_function_event_invoke_config",
						"aws",
						lambdaAllowEmptyValues,
					))
				}
			}
		}
	}
	return functions, nil
}

func (g *LambdaGenerator) addAliases(svc *lambda.Client, functions []lambdaFunctionReference) error {
	for _, function := range functions {
		p := lambda.NewListAliasesPaginator(svc, &lambda.ListAliasesInput{
			FunctionName: &function.name,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if lambdaResourceNotFound(err) {
					break
				}
				return err
			}
			for _, alias := range page.Aliases {
				aliasName := StringValue(alias.Name)
				if aliasName == "" {
					continue
				}
				g.Resources = append(g.Resources, terraformutils.NewResource(
					lambdaAliasImportID(function.name, aliasName),
					lambdaResourceName(function.name, aliasName),
					"aws_lambda_alias",
					"aws",
					map[string]string{
						"function_name": function.name,
						"name":          aliasName,
					},
					lambdaAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}

func (g *LambdaGenerator) addFunctionURLs(svc *lambda.Client, functions []lambdaFunctionReference) error {
	for _, function := range functions {
		p := lambda.NewListFunctionUrlConfigsPaginator(svc, &lambda.ListFunctionUrlConfigsInput{
			FunctionName: &function.name,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if lambdaResourceNotFound(err) {
					break
				}
				return err
			}
			for _, functionURL := range page.FunctionUrlConfigs {
				qualifier := lambdaQualifierFromFunctionARN(StringValue(functionURL.FunctionArn), function.name)
				attributes := map[string]string{
					"function_name": function.name,
				}
				if qualifier != "" {
					attributes["qualifier"] = qualifier
				}
				g.Resources = append(g.Resources, terraformutils.NewResource(
					lambdaFunctionURLImportID(function.name, qualifier),
					lambdaResourceName(function.name, "url", qualifier),
					"aws_lambda_function_url",
					"aws",
					attributes,
					lambdaAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}

func (g *LambdaGenerator) addFunctionRecursionConfigs(svc *lambda.Client, functions []lambdaFunctionReference) error {
	for _, function := range functions {
		config, err := svc.GetFunctionRecursionConfig(context.TODO(), &lambda.GetFunctionRecursionConfigInput{
			FunctionName: &function.name,
		})
		if err != nil {
			if lambdaResourceNotFound(err) {
				continue
			}
			return err
		}
		resource, ok := newLambdaFunctionRecursionConfigResource(function.name, config)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newLambdaFunctionRecursionConfigResource(functionName string, config *lambda.GetFunctionRecursionConfigOutput) (terraformutils.Resource, bool) {
	if functionName == "" || config == nil {
		return terraformutils.Resource{}, false
	}
	recursiveLoop := string(config.RecursiveLoop)
	if recursiveLoop == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		lambdaFunctionRecursionConfigImportID(functionName),
		lambdaResourceNameWithLengths("function_recursion_config", functionName),
		lambdaFunctionRecursionConfigResourceType,
		"aws",
		map[string]string{
			"function_name":  functionName,
			"recursive_loop": recursiveLoop,
		},
		lambdaAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func (g *LambdaGenerator) addRuntimeManagementConfigs(svc *lambda.Client, functions []lambdaFunctionReference) error {
	for _, function := range functions {
		config, err := svc.GetRuntimeManagementConfig(context.TODO(), &lambda.GetRuntimeManagementConfigInput{
			FunctionName: &function.name,
		})
		if err != nil {
			if lambdaResourceNotFound(err) {
				continue
			}
			return err
		}
		resource, ok := newLambdaRuntimeManagementConfigResource(function.name, "", config)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func newLambdaRuntimeManagementConfigResource(functionName, qualifier string, config *lambda.GetRuntimeManagementConfigOutput) (terraformutils.Resource, bool) {
	if functionName == "" || config == nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"function_name": functionName,
		"qualifier":     qualifier,
	}
	if runtimeVersionARN := StringValue(config.RuntimeVersionArn); runtimeVersionARN != "" {
		attributes["runtime_version_arn"] = runtimeVersionARN
	}
	if updateRuntimeOn := string(config.UpdateRuntimeOn); updateRuntimeOn != "" {
		attributes["update_runtime_on"] = updateRuntimeOn
	}
	return terraformutils.NewResource(
		lambdaRuntimeManagementConfigImportID(functionName, qualifier),
		lambdaResourceNameWithLengths("runtime_management_config", functionName, qualifier),
		lambdaRuntimeManagementConfigResourceType,
		"aws",
		attributes,
		lambdaAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func (g *LambdaGenerator) addProvisionedConcurrencyConfigs(svc *lambda.Client, functions []lambdaFunctionReference) error {
	for _, function := range functions {
		p := lambda.NewListProvisionedConcurrencyConfigsPaginator(svc, &lambda.ListProvisionedConcurrencyConfigsInput{
			FunctionName: &function.name,
		})
		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if lambdaResourceNotFound(err) {
					break
				}
				return err
			}
			for _, config := range page.ProvisionedConcurrencyConfigs {
				qualifier := lambdaQualifierFromFunctionARN(StringValue(config.FunctionArn), function.name)
				if qualifier == "" {
					continue
				}
				g.Resources = append(g.Resources, terraformutils.NewResource(
					lambdaProvisionedConcurrencyConfigImportID(function.name, qualifier),
					lambdaResourceName(function.name, "provisioned_concurrency", qualifier),
					"aws_lambda_provisioned_concurrency_config",
					"aws",
					map[string]string{
						"function_name":                     function.name,
						"qualifier":                         qualifier,
						"provisioned_concurrent_executions": strconv.FormatInt(int64(aws.ToInt32(config.AllocatedProvisionedConcurrentExecutions)), 10),
					},
					lambdaAllowEmptyValues,
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}

func (g *LambdaGenerator) addCodeSigningConfigs(svc *lambda.Client) error {
	p := lambda.NewListCodeSigningConfigsPaginator(svc, &lambda.ListCodeSigningConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.CodeSigningConfigs {
			configARN := StringValue(config.CodeSigningConfigArn)
			if configARN == "" {
				continue
			}
			resourceName := StringValue(config.CodeSigningConfigId)
			if resourceName == "" {
				resourceName = configARN
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				configARN,
				lambdaResourceName("code_signing", resourceName),
				"aws_lambda_code_signing_config",
				"aws",
				lambdaAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *LambdaGenerator) addEventSourceMappings(svc *lambda.Client) error {
	p := lambda.NewListEventSourceMappingsPaginator(svc, &lambda.ListEventSourceMappingsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, mapping := range page.EventSourceMappings {
			mappingUUID := StringValue(mapping.UUID)
			eventSourceARN := StringValue(mapping.EventSourceArn)
			functionARN := StringValue(mapping.FunctionArn)
			if mappingUUID == "" || functionARN == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				mappingUUID,
				mappingUUID,
				"aws_lambda_event_source_mapping",
				"aws",
				lambdaEventSourceMappingAttributes(functionARN, eventSourceARN),
				lambdaAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func lambdaEventSourceMappingAttributes(functionARN, eventSourceARN string) map[string]string {
	attributes := map[string]string{
		"function_name": functionARN,
	}
	if eventSourceARN != "" {
		attributes["event_source_arn"] = eventSourceARN
	}
	return attributes
}

func lambdaAliasImportID(functionName, aliasName string) string {
	return functionName + "/" + aliasName
}

func lambdaFunctionURLImportID(functionName, qualifier string) string {
	if qualifier == "" {
		return functionName
	}
	return functionName + "/" + qualifier
}

func lambdaProvisionedConcurrencyConfigImportID(functionName, qualifier string) string {
	return functionName + "," + qualifier
}

func lambdaFunctionRecursionConfigImportID(functionName string) string {
	return functionName
}

func lambdaRuntimeManagementConfigImportID(functionName, qualifier string) string {
	return functionName + "," + qualifier
}

func lambdaQualifierFromFunctionARN(functionARN, functionName string) string {
	_, qualifiedName, found := strings.Cut(functionARN, ":function:")
	if !found || qualifiedName == "" || qualifiedName == functionName {
		return ""
	}

	prefix := functionName + ":"
	if strings.HasPrefix(qualifiedName, prefix) {
		return strings.TrimPrefix(qualifiedName, prefix)
	}

	_, qualifier, found := strings.Cut(qualifiedName, ":")
	if !found {
		return ""
	}
	return qualifier
}

func lambdaResourceName(parts ...string) string {
	var name string
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name != "" {
			name += "_"
		}
		name += part
	}
	return name
}

func lambdaResourceNameWithLengths(parts ...string) string {
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

func lambdaResourceNotFound(err error) bool {
	var notFound *lambdatypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}

func (g *LambdaGenerator) addLayerVersions(svc *lambda.Client) error {
	pl := lambda.NewListLayersPaginator(svc, &lambda.ListLayersInput{})
	for pl.HasMorePages() {
		plage, err := pl.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, layer := range plage.Layers {
			pv := lambda.NewListLayerVersionsPaginator(svc, &lambda.ListLayerVersionsInput{
				LayerName: layer.LayerName,
			})
			for pv.HasMorePages() {
				pvage, err := pv.NextPage(context.TODO())
				if err != nil {
					return err
				}
				for _, layerVersion := range pvage.LayerVersions {
					g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
						*layerVersion.LayerVersionArn,
						*layerVersion.LayerVersionArn,
						"aws_lambda_layer_version",
						"aws",
						lambdaAllowEmptyValues,
					))
				}
			}
		}
	}
	return nil
}
