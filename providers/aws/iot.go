// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iot"
	"github.com/chenrui333/terraformer/terraformutils"
)

var iotAllowEmptyValues = []string{"tags."}

type IotGenerator struct {
	AWSService
}

func (g *IotGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := iot.NewFromConfig(config)

	if err := g.loadThingTypes(svc); err != nil {
		return err
	}
	if err := g.loadThings(svc); err != nil {
		return err
	}
	if err := g.loadTopicRules(svc); err != nil {
		return err
	}
	if err := g.loadRoleAliases(svc); err != nil {
		return err
	}

	return nil
}

func (g *IotGenerator) loadThingTypes(svc *iot.Client) error {
	output, err := svc.ListThingTypes(context.TODO(), &iot.ListThingTypesInput{})
	if err != nil {
		return err
	}
	for _, thingType := range output.ThingTypes {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			*thingType.ThingTypeName,
			*thingType.ThingTypeName,
			"aws_iot_thing_type",
			"aws",
			map[string]string{
				"name": *thingType.ThingTypeName,
			},
			iotAllowEmptyValues,
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *IotGenerator) loadThings(svc *iot.Client) error {
	output, err := svc.ListThings(context.TODO(), &iot.ListThingsInput{})
	if err != nil {
		return err
	}
	for _, thing := range output.Things {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			*thing.ThingName,
			*thing.ThingName,
			"aws_iot_thing",
			"aws",
			map[string]string{
				"name": *thing.ThingName,
			},
			iotAllowEmptyValues,
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *IotGenerator) loadTopicRules(svc *iot.Client) error {
	output, err := svc.ListTopicRules(context.TODO(), &iot.ListTopicRulesInput{})
	if err != nil {
		return err
	}
	for _, rule := range output.Rules {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*rule.RuleName,
			*rule.RuleName,
			"aws_iot_topic_rule",
			"aws",
			iotAllowEmptyValues))
	}
	return nil
}

func (g *IotGenerator) loadRoleAliases(svc *iot.Client) error {
	output, err := svc.ListRoleAliases(context.TODO(), &iot.ListRoleAliasesInput{})
	if err != nil {
		return err
	}
	for _, roleAlias := range output.RoleAliases {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			roleAlias,
			roleAlias,
			"aws_iot_role_alias",
			"aws",
			iotAllowEmptyValues))
	}
	return nil
}
