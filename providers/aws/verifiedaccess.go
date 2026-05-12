// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var verifiedAccessAllowEmptyValues = []string{"tags."}

const (
	verifiedAccessEndpointResourceType      = "aws_verifiedaccess_endpoint"
	verifiedAccessGroupResourceType         = "aws_verifiedaccess_group"
	verifiedAccessInstanceResourceType      = "aws_verifiedaccess_instance"
	verifiedAccessTrustProviderResourceType = "aws_verifiedaccess_trust_provider"
)

type VerifiedAccessGenerator struct {
	AWSService
}

func (g *VerifiedAccessGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := ec2.NewFromConfig(config)

	if g.shouldLoadVerifiedAccessResource(verifiedAccessInstanceResourceType) {
		if err := g.loadVerifiedAccessInstances(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadVerifiedAccessResource(verifiedAccessGroupResourceType) {
		if err := g.loadVerifiedAccessGroups(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadVerifiedAccessResource(verifiedAccessEndpointResourceType) {
		if err := g.loadVerifiedAccessEndpoints(svc); err != nil {
			return err
		}
	}
	if g.shouldLoadVerifiedAccessResource(verifiedAccessTrustProviderResourceType) {
		if err := g.loadVerifiedAccessTrustProviders(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *VerifiedAccessGenerator) shouldLoadVerifiedAccessResource(serviceNames ...string) bool {
	return shouldLoadAWSResourceForTypedFilters(g.Filter, serviceNames...)
}

func (g *VerifiedAccessGenerator) loadVerifiedAccessInstances(svc *ec2.Client) error {
	p := ec2.NewDescribeVerifiedAccessInstancesPaginator(svc, &ec2.DescribeVerifiedAccessInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, instance := range page.VerifiedAccessInstances {
			if resource, ok := newVerifiedAccessInstanceResource(instance); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *VerifiedAccessGenerator) loadVerifiedAccessGroups(svc *ec2.Client) error {
	p := ec2.NewDescribeVerifiedAccessGroupsPaginator(svc, &ec2.DescribeVerifiedAccessGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.VerifiedAccessGroups {
			if resource, ok := newVerifiedAccessGroupResource(group); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *VerifiedAccessGenerator) loadVerifiedAccessEndpoints(svc *ec2.Client) error {
	p := ec2.NewDescribeVerifiedAccessEndpointsPaginator(svc, &ec2.DescribeVerifiedAccessEndpointsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, endpoint := range page.VerifiedAccessEndpoints {
			if resource, ok := newVerifiedAccessEndpointResource(endpoint); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *VerifiedAccessGenerator) loadVerifiedAccessTrustProviders(svc *ec2.Client) error {
	p := ec2.NewDescribeVerifiedAccessTrustProvidersPaginator(svc, &ec2.DescribeVerifiedAccessTrustProvidersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, trustProvider := range page.VerifiedAccessTrustProviders {
			if resource, ok := newVerifiedAccessTrustProviderResource(trustProvider); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newVerifiedAccessInstanceResource(instance ec2types.VerifiedAccessInstance) (terraformutils.Resource, bool) {
	id := StringValue(instance.VerifiedAccessInstanceId)
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		verifiedAccessResourceName("instance", id),
		verifiedAccessInstanceResourceType,
		"aws",
		verifiedAccessAllowEmptyValues), true
}

func newVerifiedAccessGroupResource(group ec2types.VerifiedAccessGroup) (terraformutils.Resource, bool) {
	id := StringValue(group.VerifiedAccessGroupId)
	instanceID := StringValue(group.VerifiedAccessInstanceId)
	if id == "" || instanceID == "" || StringValue(group.DeletionTime) != "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		verifiedAccessResourceName("group", id, instanceID),
		verifiedAccessGroupResourceType,
		"aws",
		map[string]string{
			"verifiedaccess_instance_id": instanceID,
		},
		verifiedAccessAllowEmptyValues,
		map[string]interface{}{}), true
}

func newVerifiedAccessEndpointResource(endpoint ec2types.VerifiedAccessEndpoint) (terraformutils.Resource, bool) {
	id := StringValue(endpoint.VerifiedAccessEndpointId)
	groupID := StringValue(endpoint.VerifiedAccessGroupId)
	if id == "" || groupID == "" || StringValue(endpoint.DeletionTime) != "" || !verifiedAccessEndpointStatusImportable(endpoint.Status) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		verifiedAccessResourceName("endpoint", id, groupID),
		verifiedAccessEndpointResourceType,
		"aws",
		map[string]string{
			"verifiedaccess_group_id": groupID,
		},
		verifiedAccessAllowEmptyValues,
		map[string]interface{}{}), true
}

func verifiedAccessEndpointStatusImportable(status *ec2types.VerifiedAccessEndpointStatus) bool {
	return status != nil && status.Code == ec2types.VerifiedAccessEndpointStatusCodeActive
}

func newVerifiedAccessTrustProviderResource(trustProvider ec2types.VerifiedAccessTrustProvider) (terraformutils.Resource, bool) {
	id := StringValue(trustProvider.VerifiedAccessTrustProviderId)
	if id == "" || verifiedAccessTrustProviderRequiresSecret(trustProvider) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		id,
		verifiedAccessResourceName("trust_provider", id, StringValue(trustProvider.PolicyReferenceName)),
		verifiedAccessTrustProviderResourceType,
		"aws",
		verifiedAccessAllowEmptyValues), true
}

func verifiedAccessTrustProviderRequiresSecret(trustProvider ec2types.VerifiedAccessTrustProvider) bool {
	return trustProvider.OidcOptions != nil || trustProvider.NativeApplicationOidcOptions != nil
}

func verifiedAccessResourceName(parts ...string) string {
	return awsResourceNameWithLengths(parts...)
}
