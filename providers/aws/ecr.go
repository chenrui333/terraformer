// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"github.com/chenrui333/terraformer/terraformutils"
)

var ecrAllowEmptyValues = []string{"tags."}

// Unsupported account settings are skipped by the InvalidParameterException guard.
var ecrAccountSettingNames = []string{
	"BASIC_SCAN_TYPE_VERSION",
	"BLOB_MOUNTING",
	"REGISTRY_POLICY_SCOPE",
}

type ecrOptionalResourceLoader struct {
	name string
	load func() error
}

type EcrGenerator struct {
	AWSService
}

func (g *EcrGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}

	svc := ecr.NewFromConfig(config)

	// Registry-level resources use separate IAM actions; keep repository import best-effort when those are denied.
	g.getOptionalRegistryResources(
		ecrOptionalResourceLoader{name: "account settings", load: func() error { return g.getAccountSettings(svc) }},
		ecrOptionalResourceLoader{name: "registry policy", load: func() error { return g.getRegistryPolicy(svc) }},
		ecrOptionalResourceLoader{name: "registry scanning configuration", load: func() error { return g.getRegistryScanningConfiguration(svc) }},
		ecrOptionalResourceLoader{name: "replication configuration", load: func() error { return g.getReplicationConfiguration(svc) }},
		ecrOptionalResourceLoader{name: "pull-through cache rules", load: func() error { return g.getPullThroughCacheRules(svc) }},
		ecrOptionalResourceLoader{name: "repository creation templates", load: func() error { return g.getRepositoryCreationTemplates(svc) }},
	)

	p := ecr.NewDescribeRepositoriesPaginator(svc, &ecr.DescribeRepositoriesInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, repository := range page.Repositories {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*repository.RepositoryName,
				*repository.RepositoryName,
				"aws_ecr_repository",
				"aws",
				ecrAllowEmptyValues))

			repositoryPolicy, err := svc.GetRepositoryPolicy(context.TODO(), &ecr.GetRepositoryPolicyInput{
				RepositoryName: repository.RepositoryName,
				RegistryId:     repository.RegistryId,
			})
			if err == nil && StringValue(repositoryPolicy.PolicyText) != "" {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					*repository.RepositoryName,
					*repository.RepositoryName,
					"aws_ecr_repository_policy",
					"aws",
					ecrAllowEmptyValues))
			}

			lifecyclePolicy, err := svc.GetLifecyclePolicy(context.TODO(), &ecr.GetLifecyclePolicyInput{
				RepositoryName: repository.RepositoryName,
				RegistryId:     repository.RegistryId,
			})
			if err == nil && StringValue(lifecyclePolicy.LifecyclePolicyText) != "" {
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					*repository.RepositoryName,
					*repository.RepositoryName,
					"aws_ecr_lifecycle_policy",
					"aws",
					ecrAllowEmptyValues))
			}
		}
	}
	return nil
}

func (g *EcrGenerator) getOptionalRegistryResources(loaders ...ecrOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("failed to discover optional ECR %s: %s", loader.name, err)
		}
	}
}

func (g *EcrGenerator) getAccountSettings(svc *ecr.Client) error {
	for _, settingName := range ecrAccountSettingNames {
		setting, err := svc.GetAccountSetting(context.TODO(), &ecr.GetAccountSettingInput{
			Name: &settingName,
		})
		if ecrInvalidParameter(err) {
			continue
		}
		if err != nil {
			return err
		}
		if StringValue(setting.Value) == "" {
			continue
		}

		name := StringValue(setting.Name)
		if name == "" {
			name = settingName
		}
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			name,
			name,
			"aws_ecr_account_setting",
			"aws",
			ecrAllowEmptyValues))
	}
	return nil
}

func (g *EcrGenerator) getRegistryPolicy(svc *ecr.Client) error {
	policy, err := svc.GetRegistryPolicy(context.TODO(), &ecr.GetRegistryPolicyInput{})
	if ecrRegistryPolicyNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	registryID := StringValue(policy.RegistryId)
	if registryID == "" || StringValue(policy.PolicyText) == "" {
		return nil
	}

	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		registryID,
		"registry-policy",
		"aws_ecr_registry_policy",
		"aws",
		ecrAllowEmptyValues))
	return nil
}

func (g *EcrGenerator) getRegistryScanningConfiguration(svc *ecr.Client) error {
	scanningConfiguration, err := svc.GetRegistryScanningConfiguration(context.TODO(), &ecr.GetRegistryScanningConfigurationInput{})
	if err != nil {
		return err
	}
	registryID := StringValue(scanningConfiguration.RegistryId)
	if registryID == "" || scanningConfiguration.ScanningConfiguration == nil || scanningConfiguration.ScanningConfiguration.ScanType == "" {
		return nil
	}

	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		registryID,
		"registry-scanning-configuration",
		"aws_ecr_registry_scanning_configuration",
		"aws",
		ecrAllowEmptyValues))
	return nil
}

func (g *EcrGenerator) getReplicationConfiguration(svc *ecr.Client) error {
	registry, err := svc.DescribeRegistry(context.TODO(), &ecr.DescribeRegistryInput{})
	if err != nil {
		return err
	}
	registryID := StringValue(registry.RegistryId)
	if registryID == "" || registry.ReplicationConfiguration == nil || len(registry.ReplicationConfiguration.Rules) == 0 {
		return nil
	}

	g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
		registryID,
		"replication-configuration",
		"aws_ecr_replication_configuration",
		"aws",
		ecrAllowEmptyValues))
	return nil
}

func (g *EcrGenerator) getPullThroughCacheRules(svc *ecr.Client) error {
	p := ecr.NewDescribePullThroughCacheRulesPaginator(svc, &ecr.DescribePullThroughCacheRulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if ecrPullThroughCacheRuleNotFound(err) {
			// The unfiltered list should only return this before any rules are found.
			return nil
		}
		if err != nil {
			return err
		}
		for _, rule := range page.PullThroughCacheRules {
			prefix := StringValue(rule.EcrRepositoryPrefix)
			if prefix == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				prefix,
				prefix,
				"aws_ecr_pull_through_cache_rule",
				"aws",
				ecrAllowEmptyValues))
		}
	}
	return nil
}

func (g *EcrGenerator) getRepositoryCreationTemplates(svc *ecr.Client) error {
	p := ecr.NewDescribeRepositoryCreationTemplatesPaginator(svc, &ecr.DescribeRepositoryCreationTemplatesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if ecrTemplateNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, template := range page.RepositoryCreationTemplates {
			prefix := StringValue(template.Prefix)
			if prefix == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				prefix,
				prefix,
				"aws_ecr_repository_creation_template",
				"aws",
				ecrAllowEmptyValues))
		}
	}
	return nil
}

func (g *EcrGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		switch resource.InstanceInfo.Type {
		case "aws_ecr_repository_policy", "aws_ecr_lifecycle_policy", "aws_ecr_registry_policy":
			g.wrapEcrPolicy(i, "policy")
		case "aws_ecr_repository_creation_template":
			g.wrapEcrPolicy(i, "repository_policy")
			g.wrapEcrPolicy(i, "lifecycle_policy")
		}
	}
	return nil
}

func (g *EcrGenerator) wrapEcrPolicy(resourceIndex int, field string) {
	policy, ok := g.Resources[resourceIndex].Item[field].(string)
	if !ok || policy == "" {
		return
	}
	g.Resources[resourceIndex].Item[field] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}

func ecrInvalidParameter(err error) bool {
	var invalidParameter *types.InvalidParameterException
	return errors.As(err, &invalidParameter)
}

func ecrRegistryPolicyNotFound(err error) bool {
	var notFound *types.RegistryPolicyNotFoundException
	return errors.As(err, &notFound)
}

func ecrPullThroughCacheRuleNotFound(err error) bool {
	var notFound *types.PullThroughCacheRuleNotFoundException
	return errors.As(err, &notFound)
}

func ecrTemplateNotFound(err error) bool {
	var notFound *types.TemplateNotFoundException
	return errors.As(err, &notFound)
}
