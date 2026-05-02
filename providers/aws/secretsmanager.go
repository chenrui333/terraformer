// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var secretsmanagerAllowEmptyValues = []string{"tags."}

type SecretsManagerGenerator struct {
	AWSService
}

type secretsManagerChildResource struct {
	serviceName string
}

var secretsManagerChildResources = []secretsManagerChildResource{
	{serviceName: "secretsmanager_secret_policy"},
	{serviceName: "secretsmanager_secret_rotation"},
}

func (g *SecretsManagerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := secretsmanager.NewFromConfig(config)
	p := secretsmanager.NewListSecretsPaginator(svc, &secretsmanager.ListSecretsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, secret := range page.SecretList {
			secretArn := StringValue(secret.ARN)
			secretName := StringValue(secret.Name)
			if secretArn == "" || secretName == "" {
				continue
			}
			secretResource := newSecretsManagerSecretResource(secretArn, secretName)
			if g.shouldAppendSecretResource(secretResource) {
				g.Resources = append(g.Resources, secretResource)
			}

			if !g.shouldLoadSecretChildren(secretResource) {
				continue
			}
			policyResource := newSecretsManagerSecretPolicyResource(secretArn, secretName, "")
			if g.shouldLoadSecretChildResource("secretsmanager_secret_policy", policyResource) {
				if err := g.addSecretPolicy(svc, secretArn, secretName); err != nil {
					if !secretsManagerResourceMissing(err) {
						log.Printf("Error adding Secrets Manager policy for %s (%s): %v", secretName, secretArn, err)
					}
				}
			}
			rotationResource := newSecretsManagerSecretRotationResource(secretArn, secretName)
			if g.shouldLoadSecretChildResource("secretsmanager_secret_rotation", rotationResource) {
				g.addSecretRotation(secret)
			}
		}
	}
	return nil
}

func newSecretsManagerSecretResource(secretArn, secretName string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		secretArn,
		secretName,
		"aws_secretsmanager_secret",
		"aws",
		secretsmanagerAllowEmptyValues)
}

func newSecretsManagerSecretPolicyResource(secretArn, secretName, policy string) terraformutils.Resource {
	return terraformutils.NewResource(
		secretArn,
		secretName,
		"aws_secretsmanager_secret_policy",
		"aws",
		map[string]string{
			"secret_arn": secretArn,
			"policy":     policy,
		},
		secretsmanagerAllowEmptyValues,
		map[string]interface{}{})
}

func newSecretsManagerSecretRotationResource(secretArn, secretName string) terraformutils.Resource {
	return terraformutils.NewResource(
		secretArn,
		secretName,
		"aws_secretsmanager_secret_rotation",
		"aws",
		map[string]string{"secret_id": secretArn},
		secretsmanagerAllowEmptyValues,
		map[string]interface{}{})
}

func (g *SecretsManagerGenerator) addSecretPolicy(svc *secretsmanager.Client, secretArn, secretName string) error {
	output, err := svc.GetResourcePolicy(context.TODO(), &secretsmanager.GetResourcePolicyInput{
		SecretId: aws.String(secretArn),
	})
	if err != nil {
		return err
	}
	if output == nil {
		return nil
	}
	policy := StringValue(output.ResourcePolicy)
	if policy == "" {
		return nil
	}

	resource := newSecretsManagerSecretPolicyResource(secretArn, secretName, policy)
	if g.shouldAppendSecretChildResource("secretsmanager_secret_policy", resource) {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *SecretsManagerGenerator) addSecretRotation(secret secretstypes.SecretListEntry) {
	if !secretsManagerSecretRotationConfigured(secret) {
		return
	}
	secretArn := StringValue(secret.ARN)
	secretName := StringValue(secret.Name)

	resource := newSecretsManagerSecretRotationResource(secretArn, secretName)
	if g.shouldAppendSecretChildResource("secretsmanager_secret_rotation", resource) {
		g.Resources = append(g.Resources, resource)
	}
}

func (g *SecretsManagerGenerator) shouldAppendSecretResource(secretResource terraformutils.Resource) bool {
	if !g.secretMatchesInitialIDFilters(secretResource) {
		return false
	}
	if g.hasTypedSecretsManagerChildFilter() && !g.hasTypedFilterFor("secretsmanager_secret") && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *SecretsManagerGenerator) shouldLoadSecretChildren(secretResource terraformutils.Resource) bool {
	if g.hasTypedSecretsManagerChildFilter() {
		return g.secretMatchesAnyChildInitialFilter(secretResource)
	}
	if !g.secretMatchesInitialIDFilters(secretResource) {
		return false
	}
	return !g.hasTypedNonIDSecretFilter()
}

func (g *SecretsManagerGenerator) shouldAppendSecretChildResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedSecretsManagerChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	return g.secretChildMatchesInitialIDFilters(serviceName, resource)
}

func (g *SecretsManagerGenerator) shouldLoadSecretChildResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedSecretsManagerChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	return g.secretChildMatchesInitialIDFilters(serviceName, resource)
}

func (g *SecretsManagerGenerator) secretMatchesInitialIDFilters(secretResource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("secretsmanager_secret") {
			continue
		}
		if !filter.Filter(secretResource) {
			return false
		}
	}
	return true
}

func (g *SecretsManagerGenerator) secretMatchesAnyChildInitialFilter(secretResource terraformutils.Resource) bool {
	for _, child := range secretsManagerChildResources {
		var childResource terraformutils.Resource
		switch child.serviceName {
		case "secretsmanager_secret_policy":
			childResource = newSecretsManagerSecretPolicyResource(secretResource.InstanceState.ID, secretResource.ResourceName, "")
		case "secretsmanager_secret_rotation":
			childResource = newSecretsManagerSecretRotationResource(secretResource.InstanceState.ID, secretResource.ResourceName)
		default:
			continue
		}

		if g.shouldLoadSecretChildResource(child.serviceName, childResource) {
			return true
		}
	}
	return false
}

func (g *SecretsManagerGenerator) secretChildMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !filter.Filter(resource) {
			return false
		}
	}
	return true
}

func (g *SecretsManagerGenerator) hasTypedSecretsManagerChildFilter() bool {
	for _, child := range secretsManagerChildResources {
		if g.hasTypedFilterFor(child.serviceName) {
			return true
		}
	}
	return false
}

func (g *SecretsManagerGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *SecretsManagerGenerator) hasTypedNonIDSecretFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "secretsmanager_secret" && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *SecretsManagerGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}

func (g *SecretsManagerGenerator) PostConvertHook() error {
	policyResourcesBySecretARN := map[string]struct{}{}
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_secretsmanager_secret_policy" {
			policyResourcesBySecretARN[resource.InstanceState.ID] = struct{}{}
		}
	}

	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_secretsmanager_secret":
			if _, ok := policyResourcesBySecretARN[resource.InstanceState.ID]; ok {
				delete(g.Resources[i].Item, "policy")
			} else {
				g.wrapSecretsManagerPolicy(i, "policy")
			}
		case "aws_secretsmanager_secret_policy":
			g.wrapSecretsManagerPolicy(i, "policy")
		}
	}
	return nil
}

func (g *SecretsManagerGenerator) wrapSecretsManagerPolicy(resourceIndex int, field string) {
	policy, ok := g.Resources[resourceIndex].Item[field].(string)
	if !ok || policy == "" {
		return
	}
	g.Resources[resourceIndex].Item[field] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}

func secretsManagerSecretRotationConfigured(secret secretstypes.SecretListEntry) bool {
	if !aws.ToBool(secret.RotationEnabled) || secret.RotationRules == nil {
		return false
	}
	return secretsManagerRotationRulesConfigured(secret.RotationRules)
}

func secretsManagerRotationRulesConfigured(rules *secretstypes.RotationRulesType) bool {
	if rules == nil {
		return false
	}
	return rules.AutomaticallyAfterDays != nil || StringValue(rules.Duration) != "" || StringValue(rules.ScheduleExpression) != ""
}

func secretsManagerResourceMissing(err error) bool {
	var notFound *secretstypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}

	var invalidRequest *secretstypes.InvalidRequestException
	return errors.As(err, &invalidRequest) && strings.Contains(invalidRequest.ErrorMessage(), "marked for deletion")
}
