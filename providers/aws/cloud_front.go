// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudFrontAllowEmptyValues = []string{"tags."}

type cloudFrontOptionalResourceLoader struct {
	name string
	load func() error
}

type CloudFrontGenerator struct {
	AWSService
}

func (g *CloudFrontGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloudfront.NewFromConfig(config)

	if err := g.loadDistribution(svc); err != nil {
		return err
	}

	if err := g.loadCachePolicy(svc); err != nil {
		return err
	}

	g.loadOptionalResources([]cloudFrontOptionalResourceLoader{
		{name: "origin access controls", load: func() error { return g.loadOriginAccessControls(svc) }},
		{name: "origin access identities", load: func() error { return g.loadOriginAccessIdentities(svc) }},
		{name: "origin request policies", load: func() error { return g.loadOriginRequestPolicies(svc) }},
		{name: "response headers policies", load: func() error { return g.loadResponseHeadersPolicies(svc) }},
		{name: "realtime log configs", load: func() error { return g.loadRealtimeLogConfigs(svc) }},
		{name: "functions", load: func() error { return g.loadFunctions(svc) }},
		{name: "public keys", load: func() error { return g.loadPublicKeys(svc) }},
		{name: "key groups", load: func() error { return g.loadKeyGroups(svc) }},
		{name: "field-level encryption configs", load: func() error { return g.loadFieldLevelEncryptionConfigs(svc) }},
		{name: "field-level encryption profiles", load: func() error { return g.loadFieldLevelEncryptionProfiles(svc) }},
		{name: "continuous deployment policies", load: func() error { return g.loadContinuousDeploymentPolicies(svc) }},
		{name: "VPC origins", load: func() error { return g.loadVpcOrigins(svc) }},
		{name: "key value stores", load: func() error { return g.loadKeyValueStores(svc) }},
	})

	return nil
}

func (g *CloudFrontGenerator) loadOptionalResources(loaders []cloudFrontOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			if cloudFrontResourceMissing(err) {
				continue
			}
			log.Printf("Skipping CloudFront %s: %v", loader.name, err)
		}
	}
}

func (g *CloudFrontGenerator) loadDistribution(svc *cloudfront.Client) error {
	p := cloudfront.NewListDistributionsPaginator(svc, &cloudfront.ListDistributionsInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		if page.DistributionList == nil {
			continue
		}
		for _, distribution := range page.DistributionList.Items {
			distributionID := StringValue(distribution.Id)
			if distributionID == "" {
				continue
			}
			r := terraformutils.NewResource(
				distributionID,
				distributionID,
				"aws_cloudfront_distribution",
				"aws",
				map[string]string{
					"retain_on_delete": "false",
				},
				cloudFrontAllowEmptyValues,
				map[string]interface{}{},
			)
			r.IgnoreKeys = append(r.IgnoreKeys, "^active_trusted_signers.(.*)")
			g.Resources = append(g.Resources, r)

			if err := g.loadMonitoringSubscription(svc, distributionID); err != nil {
				if !cloudFrontResourceMissing(err) {
					log.Printf("Skipping CloudFront monitoring subscription for %s: %v", distributionID, err)
				}
			}
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadMonitoringSubscription(svc *cloudfront.Client, distributionID string) error {
	out, err := svc.GetMonitoringSubscription(context.TODO(), &cloudfront.GetMonitoringSubscriptionInput{
		DistributionId: aws.String(distributionID),
	})
	if err != nil {
		return err
	}
	if out.MonitoringSubscription == nil || out.MonitoringSubscription.RealtimeMetricsSubscriptionConfig == nil {
		return nil
	}
	status := string(out.MonitoringSubscription.RealtimeMetricsSubscriptionConfig.RealtimeMetricsSubscriptionStatus)
	if status == "" {
		return nil
	}
	g.Resources = append(g.Resources, terraformutils.NewResource(
		distributionID,
		distributionID,
		"aws_cloudfront_monitoring_subscription",
		"aws",
		map[string]string{
			"distribution_id": distributionID,
		},
		cloudFrontAllowEmptyValues,
		map[string]interface{}{},
	))
	return nil
}

func (g *CloudFrontGenerator) loadCachePolicy(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListCachePolicies(context.TODO(), &cloudfront.ListCachePoliciesInput{
			Marker: marker,
		})
		if err != nil {
			return err
		}
		if out.CachePolicyList == nil {
			return nil
		}
		for _, cachePolicy := range out.CachePolicyList.Items {
			if cachePolicy.CachePolicy == nil {
				continue
			}
			id := StringValue(cachePolicy.CachePolicy.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_cache_policy",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.CachePolicyList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadOriginAccessControls(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListOriginAccessControls(context.TODO(), &cloudfront.ListOriginAccessControlsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.OriginAccessControlList == nil {
			return nil
		}
		for _, control := range out.OriginAccessControlList.Items {
			id := StringValue(control.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_origin_access_control",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.OriginAccessControlList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadOriginAccessIdentities(svc *cloudfront.Client) error {
	p := cloudfront.NewListCloudFrontOriginAccessIdentitiesPaginator(svc, &cloudfront.ListCloudFrontOriginAccessIdentitiesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		if page.CloudFrontOriginAccessIdentityList == nil {
			continue
		}
		for _, identity := range page.CloudFrontOriginAccessIdentityList.Items {
			id := StringValue(identity.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_origin_access_identity",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadOriginRequestPolicies(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListOriginRequestPolicies(context.TODO(), &cloudfront.ListOriginRequestPoliciesInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.OriginRequestPolicyList == nil {
			return nil
		}
		for _, policy := range out.OriginRequestPolicyList.Items {
			if policy.OriginRequestPolicy == nil {
				continue
			}
			id := StringValue(policy.OriginRequestPolicy.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_origin_request_policy",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.OriginRequestPolicyList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadResponseHeadersPolicies(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListResponseHeadersPolicies(context.TODO(), &cloudfront.ListResponseHeadersPoliciesInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.ResponseHeadersPolicyList == nil {
			return nil
		}
		for _, policy := range out.ResponseHeadersPolicyList.Items {
			if policy.ResponseHeadersPolicy == nil {
				continue
			}
			id := StringValue(policy.ResponseHeadersPolicy.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_response_headers_policy",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.ResponseHeadersPolicyList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadRealtimeLogConfigs(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListRealtimeLogConfigs(context.TODO(), &cloudfront.ListRealtimeLogConfigsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.RealtimeLogConfigs == nil {
			return nil
		}
		for _, config := range out.RealtimeLogConfigs.Items {
			arn := StringValue(config.ARN)
			if arn == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				arn,
				cloudFrontResourceName(arn, StringValue(config.Name)),
				"aws_cloudfront_realtime_log_config",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.RealtimeLogConfigs.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadFunctions(svc *cloudfront.Client) error {
	var marker *string
	seen := map[string]struct{}{}
	for {
		out, err := svc.ListFunctions(context.TODO(), &cloudfront.ListFunctionsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.FunctionList == nil {
			return nil
		}
		for _, function := range out.FunctionList.Items {
			name := StringValue(function.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				name,
				name,
				"aws_cloudfront_function",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.FunctionList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadPublicKeys(svc *cloudfront.Client) error {
	p := cloudfront.NewListPublicKeysPaginator(svc, &cloudfront.ListPublicKeysInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		if page.PublicKeyList == nil {
			continue
		}
		for _, publicKey := range page.PublicKeyList.Items {
			id := StringValue(publicKey.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_public_key",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadKeyGroups(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListKeyGroups(context.TODO(), &cloudfront.ListKeyGroupsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.KeyGroupList == nil {
			return nil
		}
		for _, keyGroup := range out.KeyGroupList.Items {
			if keyGroup.KeyGroup == nil {
				continue
			}
			id := StringValue(keyGroup.KeyGroup.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_key_group",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.KeyGroupList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadFieldLevelEncryptionConfigs(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListFieldLevelEncryptionConfigs(context.TODO(), &cloudfront.ListFieldLevelEncryptionConfigsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.FieldLevelEncryptionList == nil {
			return nil
		}
		for _, config := range out.FieldLevelEncryptionList.Items {
			id := StringValue(config.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_field_level_encryption_config",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.FieldLevelEncryptionList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadFieldLevelEncryptionProfiles(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListFieldLevelEncryptionProfiles(context.TODO(), &cloudfront.ListFieldLevelEncryptionProfilesInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.FieldLevelEncryptionProfileList == nil {
			return nil
		}
		for _, profile := range out.FieldLevelEncryptionProfileList.Items {
			id := StringValue(profile.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				cloudFrontResourceName(id, StringValue(profile.Name)),
				"aws_cloudfront_field_level_encryption_profile",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.FieldLevelEncryptionProfileList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadContinuousDeploymentPolicies(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListContinuousDeploymentPolicies(context.TODO(), &cloudfront.ListContinuousDeploymentPoliciesInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.ContinuousDeploymentPolicyList == nil {
			return nil
		}
		for _, policy := range out.ContinuousDeploymentPolicyList.Items {
			if policy.ContinuousDeploymentPolicy == nil {
				continue
			}
			id := StringValue(policy.ContinuousDeploymentPolicy.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				id,
				"aws_cloudfront_continuous_deployment_policy",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.ContinuousDeploymentPolicyList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadVpcOrigins(svc *cloudfront.Client) error {
	var marker *string
	for {
		out, err := svc.ListVpcOrigins(context.TODO(), &cloudfront.ListVpcOriginsInput{Marker: marker})
		if err != nil {
			return err
		}
		if out.VpcOriginList == nil {
			return nil
		}
		for _, origin := range out.VpcOriginList.Items {
			id := StringValue(origin.Id)
			if id == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				id,
				cloudFrontResourceName(id, StringValue(origin.Name)),
				"aws_cloudfront_vpc_origin",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
		marker = out.VpcOriginList.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *CloudFrontGenerator) loadKeyValueStores(svc *cloudfront.Client) error {
	p := cloudfront.NewListKeyValueStoresPaginator(svc, &cloudfront.ListKeyValueStoresInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		if page.KeyValueStoreList == nil {
			continue
		}
		for _, store := range page.KeyValueStoreList.Items {
			name := StringValue(store.Name)
			if name == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				name,
				name,
				"aws_cloudfront_key_value_store",
				"aws",
				cloudFrontAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *CloudFrontGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_cloudfront_distribution" {
			continue
		}

		g.linkDistributionReferences(i)
	}
	return nil
}

func (g *CloudFrontGenerator) linkDistributionReferences(distributionIndex int) {
	for _, referencedResource := range g.Resources {
		switch referencedResource.InstanceInfo.Type {
		case "aws_cloudfront_cache_policy":
			g.replaceCacheBehaviorID(distributionIndex, "cache_policy_id", referencedResource, "id")
		case "aws_cloudfront_origin_request_policy":
			g.replaceCacheBehaviorID(distributionIndex, "origin_request_policy_id", referencedResource, "id")
		case "aws_cloudfront_response_headers_policy":
			g.replaceCacheBehaviorID(distributionIndex, "response_headers_policy_id", referencedResource, "id")
		case "aws_cloudfront_origin_access_control":
			g.replaceOriginID(distributionIndex, "origin_access_control_id", referencedResource, "id")
		}
	}
}

func (g *CloudFrontGenerator) replaceCacheBehaviorID(distributionIndex int, field string, referencedResource terraformutils.Resource, referencedAttribute string) {
	distribution := g.Resources[distributionIndex]
	refID := cloudFrontResourceID(referencedResource)
	if refID == "" {
		return
	}
	ref := cloudFrontReference(referencedResource, referencedAttribute)

	if defaultCacheBehaviors, ok := distribution.Item["default_cache_behavior"].([]interface{}); ok && len(defaultCacheBehaviors) > 0 {
		if behavior, ok := defaultCacheBehaviors[0].(map[string]interface{}); ok {
			cloudFrontReplaceFieldValue(behavior, field, refID, ref)
		}
	}

	if orderedCacheBehaviors, ok := distribution.Item["ordered_cache_behavior"].([]interface{}); ok {
		for _, orderedCacheBehavior := range orderedCacheBehaviors {
			if behavior, ok := orderedCacheBehavior.(map[string]interface{}); ok {
				cloudFrontReplaceFieldValue(behavior, field, refID, ref)
			}
		}
	}
}

func (g *CloudFrontGenerator) replaceOriginID(distributionIndex int, field string, referencedResource terraformutils.Resource, referencedAttribute string) {
	distribution := g.Resources[distributionIndex]
	refID := cloudFrontResourceID(referencedResource)
	if refID == "" {
		return
	}
	ref := cloudFrontReference(referencedResource, referencedAttribute)

	if origins, ok := distribution.Item["origin"].([]interface{}); ok {
		for _, origin := range origins {
			if origin, ok := origin.(map[string]interface{}); ok {
				cloudFrontReplaceFieldValue(origin, field, refID, ref)
			}
		}
	}
}

func cloudFrontReplaceFieldValue(item map[string]interface{}, field string, oldValue string, newValue string) {
	value, ok := item[field].(string)
	if ok && value == oldValue {
		item[field] = newValue
	}
}

func cloudFrontResourceID(resource terraformutils.Resource) string {
	if id := resource.InstanceState.Attributes["id"]; id != "" {
		return id
	}
	return resource.InstanceState.ID
}

func cloudFrontReference(resource terraformutils.Resource, attribute string) string {
	return "$" + "{" + resource.InstanceInfo.Type + "." + resource.ResourceName + "." + attribute + "}"
}

func cloudFrontResourceName(id string, name string) string {
	if name != "" {
		return name
	}
	return id
}

func cloudFrontResourceMissing(err error) bool {
	var entityNotFound *types.EntityNotFound
	var noSuchCachePolicy *types.NoSuchCachePolicy
	var noSuchOriginAccessIdentity *types.NoSuchCloudFrontOriginAccessIdentity
	var noSuchContinuousDeploymentPolicy *types.NoSuchContinuousDeploymentPolicy
	var noSuchDistribution *types.NoSuchDistribution
	var noSuchFieldLevelEncryptionConfig *types.NoSuchFieldLevelEncryptionConfig
	var noSuchFieldLevelEncryptionProfile *types.NoSuchFieldLevelEncryptionProfile
	var noSuchFunction *types.NoSuchFunctionExists
	var noSuchMonitoringSubscription *types.NoSuchMonitoringSubscription
	var noSuchOriginAccessControl *types.NoSuchOriginAccessControl
	var noSuchOriginRequestPolicy *types.NoSuchOriginRequestPolicy
	var noSuchPublicKey *types.NoSuchPublicKey
	var noSuchRealtimeLogConfig *types.NoSuchRealtimeLogConfig
	var noSuchResource *types.NoSuchResource
	var noSuchResponseHeadersPolicy *types.NoSuchResponseHeadersPolicy
	return errors.As(err, &entityNotFound) ||
		errors.As(err, &noSuchCachePolicy) ||
		errors.As(err, &noSuchOriginAccessIdentity) ||
		errors.As(err, &noSuchContinuousDeploymentPolicy) ||
		errors.As(err, &noSuchDistribution) ||
		errors.As(err, &noSuchFieldLevelEncryptionConfig) ||
		errors.As(err, &noSuchFieldLevelEncryptionProfile) ||
		errors.As(err, &noSuchFunction) ||
		errors.As(err, &noSuchMonitoringSubscription) ||
		errors.As(err, &noSuchOriginAccessControl) ||
		errors.As(err, &noSuchOriginRequestPolicy) ||
		errors.As(err, &noSuchPublicKey) ||
		errors.As(err, &noSuchRealtimeLogConfig) ||
		errors.As(err, &noSuchResource) ||
		errors.As(err, &noSuchResponseHeadersPolicy)
}
