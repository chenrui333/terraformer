// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var eksAllowEmptyValues = []string{"tags."}

var eksClusterScopedResourceTypes = map[string]struct{}{
	"aws_eks_access_entry":              {},
	"aws_eks_access_policy_association": {},
	"aws_eks_addon":                     {},
	"aws_eks_fargate_profile":           {},
	"aws_eks_identity_provider_config":  {},
	"aws_eks_node_group":                {},
	"aws_eks_pod_identity_association":  {},
}

type EksGenerator struct {
	AWSService
}

func eksResourceName(parts ...string) string {
	nonEmptyParts := []string{}
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, "-")
}

func eksArnName(arn string) string {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 {
		name := arnLastSegment(arn, "/")
		return arnLastSegment(name, ":")
	}
	return eksResourceName(parts[4], strings.ReplaceAll(parts[5], "/", "-"))
}

func eksAccessEntriesUnsupported(err error) bool {
	var invalidRequest *types.InvalidRequestException
	return errors.As(err, &invalidRequest)
}

func (g *EksGenerator) getNodeGroups(clusterName string, svc *eks.Client) error {
	p := eks.NewListNodegroupsPaginator(svc, &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, nodeGroupName := range page.Nodegroups {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", clusterName, nodeGroupName),
				nodeGroupName,
				"aws_eks_node_group",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) getAddons(clusterName string, svc *eks.Client) error {
	p := eks.NewListAddonsPaginator(svc, &eks.ListAddonsInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, addonName := range page.Addons {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", clusterName, addonName),
				eksResourceName(clusterName, addonName),
				"aws_eks_addon",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) getFargateProfiles(clusterName string, svc *eks.Client) error {
	p := eks.NewListFargateProfilesPaginator(svc, &eks.ListFargateProfilesInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, profileName := range page.FargateProfileNames {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", clusterName, profileName),
				eksResourceName(clusterName, profileName),
				"aws_eks_fargate_profile",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) getIdentityProviderConfigs(clusterName string, svc *eks.Client) error {
	p := eks.NewListIdentityProviderConfigsPaginator(svc, &eks.ListIdentityProviderConfigsInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, config := range page.IdentityProviderConfigs {
			configName := StringValue(config.Name)
			if configName == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", clusterName, configName),
				eksResourceName(clusterName, configName),
				"aws_eks_identity_provider_config",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) getPodIdentityAssociations(clusterName string, svc *eks.Client) error {
	p := eks.NewListPodIdentityAssociationsPaginator(svc, &eks.ListPodIdentityAssociationsInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.Associations {
			associationID := StringValue(association.AssociationId)
			if associationID == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s,%s", clusterName, associationID),
				eksResourceName(clusterName, StringValue(association.Namespace), StringValue(association.ServiceAccount), associationID),
				"aws_eks_pod_identity_association",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) getAccessEntries(clusterName string, svc *eks.Client) error {
	p := eks.NewListAccessEntriesPaginator(svc, &eks.ListAccessEntriesInput{
		ClusterName: &clusterName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			// CONFIG_MAP authentication clusters reject this API but can still import
			// clusters, node groups, add-ons, and other EKS resources.
			if eksAccessEntriesUnsupported(err) {
				return nil
			}
			return err
		}
		for _, principalArn := range page.AccessEntries {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s:%s", clusterName, principalArn),
				eksResourceName(clusterName, eksArnName(principalArn)),
				"aws_eks_access_entry",
				"aws",
				eksAllowEmptyValues,
			))
			if err := g.getAccessPolicyAssociations(clusterName, principalArn, svc); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *EksGenerator) getAccessPolicyAssociations(clusterName, principalArn string, svc *eks.Client) error {
	var nextToken *string
	for {
		page, err := svc.ListAssociatedAccessPolicies(context.TODO(), &eks.ListAssociatedAccessPoliciesInput{
			ClusterName:  &clusterName,
			PrincipalArn: &principalArn,
			NextToken:    nextToken,
		})
		if err != nil {
			return err
		}
		for _, policy := range page.AssociatedAccessPolicies {
			policyArn := StringValue(policy.PolicyArn)
			if policyArn == "" {
				continue
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				fmt.Sprintf("%s#%s#%s", clusterName, principalArn, policyArn),
				eksResourceName(clusterName, eksArnName(principalArn), eksArnName(policyArn)),
				"aws_eks_access_policy_association",
				"aws",
				eksAllowEmptyValues,
			))
		}
		nextToken = page.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *EksGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := eks.NewFromConfig(config)
	p := eks.NewListClustersPaginator(svc, &eks.ListClustersInput{})
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, clusterName := range page.Clusters {
			err := g.getNodeGroups(clusterName, svc)
			if err != nil {
				return err
			}
			err = g.getAddons(clusterName, svc)
			if err != nil {
				return err
			}
			err = g.getFargateProfiles(clusterName, svc)
			if err != nil {
				return err
			}
			err = g.getIdentityProviderConfigs(clusterName, svc)
			if err != nil {
				return err
			}
			err = g.getPodIdentityAssociations(clusterName, svc)
			if err != nil {
				return err
			}
			err = g.getAccessEntries(clusterName, svc)
			if err != nil {
				return err
			}
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				clusterName,
				clusterName,
				"aws_eks_cluster",
				"aws",
				eksAllowEmptyValues,
			))
		}
	}
	return nil
}

func (g *EksGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_eks_node_group" {
			if _, ok := resource.Item["launch_template"]; ok {
				delete(resource.Item["launch_template"].([]interface{})[0].(map[string]interface{}), "id")
			}
			if _, ok := resource.Item["update_config"]; ok {
				delete(resource.Item["update_config"].([]interface{})[0].(map[string]interface{}), "max_unavailable_percentage")
			}
		}
		if _, ok := eksClusterScopedResourceTypes[resource.InstanceInfo.Type]; !ok {
			continue
		}
		if _, ok := resource.Item["cluster_name"]; !ok {
			continue
		}
		for cluster := range g.Resources {
			if g.Resources[cluster].InstanceInfo.Type == "aws_eks_cluster" {
				if g.Resources[cluster].Item["name"] == resource.Item["cluster_name"] {
					resource.Item["cluster_name"] = "${aws_eks_cluster." + g.Resources[cluster].InstanceInfo.ResourceAddress().Name + ".name}"
				}
			}
		}
	}
	return nil
}
