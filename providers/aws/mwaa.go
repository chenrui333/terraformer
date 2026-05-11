// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mwaa"
	mwaatypes "github.com/aws/aws-sdk-go-v2/service/mwaa/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const mwaaEnvironmentResourceType = "aws_mwaa_environment"

var mwaaAllowEmptyValues = []string{"tags."}

type MwaaGenerator struct {
	AWSService
}

func (g *MwaaGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mwaa.NewFromConfig(config)

	return g.loadEnvironments(svc)
}

func (g *MwaaGenerator) loadEnvironments(svc *mwaa.Client) error {
	paginator := mwaa.NewListEnvironmentsPaginator(svc, &mwaa.ListEnvironmentsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if mwaaResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, name := range page.Environments {
			environment, err := getMwaaEnvironment(svc, name)
			if mwaaResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newMwaaEnvironmentResource(environment); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func getMwaaEnvironment(svc *mwaa.Client, name string) (*mwaatypes.Environment, error) {
	if name == "" {
		return nil, nil
	}
	output, err := svc.GetEnvironment(context.TODO(), &mwaa.GetEnvironmentInput{
		Name: aws.String(name),
	})
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, nil
	}
	return output.Environment, nil
}

func newMwaaEnvironmentResource(environment *mwaatypes.Environment) (terraformutils.Resource, bool) {
	if !mwaaEnvironmentImportable(environment) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(environment.Name)
	return terraformutils.NewResource(
		mwaaEnvironmentImportID(name),
		mwaaResourceName("environment", name),
		mwaaEnvironmentResourceType,
		"aws",
		mwaaEnvironmentAttributes(environment),
		mwaaAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func mwaaEnvironmentImportable(environment *mwaatypes.Environment) bool {
	if environment == nil || !mwaaEnvironmentStatusImportable(environment.Status) {
		return false
	}
	// Terraform AWS provider v6.43/v6.44 dereference LastUpdate during Read.
	if StringValue(environment.Name) == "" ||
		StringValue(environment.DagS3Path) == "" ||
		StringValue(environment.ExecutionRoleArn) == "" ||
		StringValue(environment.SourceBucketArn) == "" ||
		environment.LastUpdate == nil {
		return false
	}
	return mwaaEnvironmentNetworkConfigurationComplete(environment.NetworkConfiguration)
}

func mwaaEnvironmentNetworkConfigurationComplete(config *mwaatypes.NetworkConfiguration) bool {
	return config != nil && len(config.SecurityGroupIds) > 0 && len(config.SubnetIds) > 1
}

func mwaaEnvironmentAttributes(environment *mwaatypes.Environment) map[string]string {
	return map[string]string{
		"dag_s3_path":        StringValue(environment.DagS3Path),
		"execution_role_arn": StringValue(environment.ExecutionRoleArn),
		"name":               StringValue(environment.Name),
		"source_bucket_arn":  StringValue(environment.SourceBucketArn),
	}
}

func mwaaEnvironmentImportID(name string) string {
	return name
}

func mwaaEnvironmentStatusImportable(status mwaatypes.EnvironmentStatus) bool {
	switch status {
	case "", mwaatypes.EnvironmentStatusDeleting, mwaatypes.EnvironmentStatusDeleted:
		return false
	default:
		return true
	}
}

func mwaaResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "mwaa-resource"
	}
	return strings.Join(cleanParts, "/")
}

func mwaaResourceNotFound(err error) bool {
	var notFound *mwaatypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
