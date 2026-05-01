// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var ecsAllowEmptyValues = []string{"tags."}

type EcsGenerator struct {
	AWSService
}

type ecsClusterReference struct {
	arn  string
	name string
}

type ecsServiceReference struct {
	name        string
	clusterARN  string
	clusterName string
}

type ecsOptionalResourceLoader struct {
	name string
	load func() error
}

func ecsCapacityProviderImportable(capacityProvider ecstypes.CapacityProvider) bool {
	return capacityProvider.AutoScalingGroupProvider != nil || capacityProvider.ManagedInstancesProvider != nil
}

func (g *EcsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := ecs.NewFromConfig(config)

	var clusters []ecsClusterReference
	var services []ecsServiceReference
	p := ecs.NewListClustersPaginator(svc, &ecs.ListClustersInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, clusterArn := range page.ClusterArns {
			clusterName := arnLastSegment(clusterArn, "/")
			clusters = append(clusters, ecsClusterReference{
				arn:  clusterArn,
				name: clusterName,
			})

			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				clusterArn,
				clusterName,
				"aws_ecs_cluster",
				"aws",
				ecsAllowEmptyValues,
			))

			servicePage := ecs.NewListServicesPaginator(svc, &ecs.ListServicesInput{
				Cluster: &clusterArn,
			})
			for servicePage.HasMorePages() {
				serviceNextPage, err := servicePage.NextPage(context.TODO())
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				for _, serviceArn := range serviceNextPage.ServiceArns {
					serviceName := arnLastSegment(serviceArn, "/")
					services = append(services, ecsServiceReference{
						name:        serviceName,
						clusterARN:  clusterArn,
						clusterName: clusterName,
					})

					serResp, err := svc.DescribeServices(context.TODO(), &ecs.DescribeServicesInput{
						Services: []string{
							serviceName,
						},
						Cluster: &clusterArn,
					})
					if err != nil {
						fmt.Println(err.Error())
						continue
					}
					serviceDetails := serResp.Services[0]

					g.Resources = append(g.Resources, terraformutils.NewResource(
						serviceArn,
						clusterName+"_"+serviceName,
						"aws_ecs_service",
						"aws",
						map[string]string{
							"task_definition": StringValue(serviceDetails.TaskDefinition),
							"cluster":         clusterName,
							"name":            serviceName,
							"id":              serviceArn,
						},
						ecsAllowEmptyValues,
						map[string]interface{}{},
					))
				}
			}
		}
	}

	g.getOptionalEcsResources(
		ecsOptionalResourceLoader{name: "capacity providers", load: func() error { return g.addCapacityProviders(svc) }},
		ecsOptionalResourceLoader{name: "cluster capacity providers", load: func() error { return g.addClusterCapacityProviders(svc, clusters) }},
		ecsOptionalResourceLoader{name: "task sets", load: func() error { return g.addTaskSets(svc, services) }},
	)

	taskDefinitionsMap := map[string]terraformutils.Resource{}
	taskDefinitionsPage := ecs.NewListTaskDefinitionsPaginator(svc, &ecs.ListTaskDefinitionsInput{})
	for taskDefinitionsPage.HasMorePages() {
		taskDefinitionsNextPage, e := taskDefinitionsPage.NextPage(context.TODO())
		if e != nil {
			fmt.Println(e.Error())
			continue
		}
		for _, taskDefinitionArn := range taskDefinitionsNextPage.TaskDefinitionArns {
			arnParts := strings.Split(taskDefinitionArn, ":")
			definitionWithFamily := arnParts[len(arnParts)-2]
			revision, _ := strconv.Atoi(arnParts[len(arnParts)-1])

			// fetch only latest revision of task definitions
			if val, ok := taskDefinitionsMap[definitionWithFamily]; !ok || val.AdditionalFields["revision"].(int) < revision {
				taskDefinitionsMap[definitionWithFamily] = terraformutils.NewResource(
					taskDefinitionArn,
					definitionWithFamily,
					"aws_ecs_task_definition",
					"aws",
					map[string]string{
						"task_definition":       taskDefinitionArn,
						"container_definitions": "{}",
						"family":                "test-task",
						"arn":                   taskDefinitionArn,
					},
					[]string{},
					map[string]interface{}{
						"revision": revision,
					},
				)
			}
		}
	}
	for _, v := range taskDefinitionsMap {
		delete(v.AdditionalFields, "revision")
		g.Resources = append(g.Resources, v)
	}

	return nil
}

func (g *EcsGenerator) getOptionalEcsResources(loaders ...ecsOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("skipping ECS %s discovery: %v", loader.name, err)
		}
	}
}

func (g *EcsGenerator) addCapacityProviders(svc *ecs.Client) error {
	var nextToken *string
	for {
		output, err := svc.DescribeCapacityProviders(context.TODO(), &ecs.DescribeCapacityProvidersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, capacityProvider := range output.CapacityProviders {
			if !ecsCapacityProviderImportable(capacityProvider) {
				continue
			}
			capacityProviderARN := StringValue(capacityProvider.CapacityProviderArn)
			capacityProviderName := StringValue(capacityProvider.Name)
			if capacityProviderARN == "" || capacityProviderName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				capacityProviderARN,
				capacityProviderName,
				"aws_ecs_capacity_provider",
				"aws",
				map[string]string{
					"name": capacityProviderName,
				},
				ecsAllowEmptyValues,
				map[string]interface{}{},
			))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *EcsGenerator) addClusterCapacityProviders(svc *ecs.Client, clusters []ecsClusterReference) error {
	for _, cluster := range clusters {
		output, err := svc.DescribeClusters(context.TODO(), &ecs.DescribeClustersInput{
			Clusters: []string{cluster.arn},
		})
		if err != nil {
			if ecsClusterNotFound(err) {
				continue
			}
			return err
		}
		for _, describedCluster := range output.Clusters {
			clusterName := StringValue(describedCluster.ClusterName)
			if clusterName == "" {
				clusterName = cluster.name
			}
			if clusterName == "" || (len(describedCluster.CapacityProviders) == 0 && len(describedCluster.DefaultCapacityProviderStrategy) == 0) {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				clusterName,
				clusterName,
				"aws_ecs_cluster_capacity_providers",
				"aws",
				map[string]string{
					"cluster_name": clusterName,
				},
				ecsAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *EcsGenerator) addTaskSets(svc *ecs.Client, services []ecsServiceReference) error {
	for _, service := range services {
		output, err := svc.DescribeTaskSets(context.TODO(), &ecs.DescribeTaskSetsInput{
			Cluster: &service.clusterARN,
			Include: []ecstypes.TaskSetField{ecstypes.TaskSetFieldTags},
			Service: &service.name,
		})
		if err != nil {
			if ecsTaskSetScopeNotFound(err) {
				continue
			}
			return err
		}
		for _, taskSet := range output.TaskSets {
			taskSetID := StringValue(taskSet.Id)
			if taskSetID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewResource(
				ecsTaskSetImportID(taskSetID, service.name, service.clusterName),
				ecsResourceName(service.clusterName, service.name, taskSetID),
				"aws_ecs_task_set",
				"aws",
				map[string]string{
					"cluster": service.clusterName,
					"service": service.name,
				},
				ecsAllowEmptyValues,
				map[string]interface{}{},
			))
		}
	}
	return nil
}

func (g *EcsGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_ecs_service" {
			continue
		}
		if r.InstanceState.Attributes["propagate_tags"] == "NONE" {
			delete(r.Item, "propagate_tags")
		}
		delete(r.Item, "iam_role")
	}

	return nil
}

func ecsTaskSetImportID(taskSetID, service, cluster string) string {
	return strings.Join([]string{taskSetID, service, cluster}, ",")
}

func ecsResourceName(parts ...string) string {
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

func ecsClusterNotFound(err error) bool {
	var notFound *ecstypes.ClusterNotFoundException
	return errors.As(err, &notFound)
}

func ecsTaskSetScopeNotFound(err error) bool {
	var clusterNotFound *ecstypes.ClusterNotFoundException
	if errors.As(err, &clusterNotFound) {
		return true
	}
	var serviceNotFound *ecstypes.ServiceNotFoundException
	if errors.As(err, &serviceNotFound) {
		return true
	}
	var taskSetNotFound *ecstypes.TaskSetNotFoundException
	return errors.As(err, &taskSetNotFound)
}
