// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk"
)

var beanstalkAllowEmptyValues = []string{"tags."}

type BeanstalkGenerator struct {
	AWSService
}

func (g *BeanstalkGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	client := elasticbeanstalk.NewFromConfig(config)

	err := g.addApplications(client)
	if err != nil {
		return err
	}
	err = g.addEnvironments(client)
	return err
}

func (g *BeanstalkGenerator) addApplications(client *elasticbeanstalk.Client) error {
	response, err := client.DescribeApplications(context.TODO(), &elasticbeanstalk.DescribeApplicationsInput{})
	if err != nil {
		return err
	}
	for _, application := range response.Applications {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*application.ApplicationName,
			*application.ApplicationName,
			"aws_elastic_beanstalk_application",
			"aws",
			beanstalkAllowEmptyValues,
		))
	}
	return nil
}

func (g *BeanstalkGenerator) addEnvironments(client *elasticbeanstalk.Client) error {
	response, err := client.DescribeEnvironments(context.TODO(), &elasticbeanstalk.DescribeEnvironmentsInput{})
	if err != nil {
		return err
	}
	for _, environment := range response.Environments {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			*environment.EnvironmentId,
			*environment.EnvironmentName,
			"aws_elastic_beanstalk_environment",
			"aws",
			beanstalkAllowEmptyValues,
		))
	}
	return nil
}
