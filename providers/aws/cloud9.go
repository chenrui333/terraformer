// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloud9"
	"github.com/aws/aws-sdk-go-v2/service/cloud9/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	cloud9EnvironmentEC2ResourceType        = "aws_cloud9_environment_ec2"
	cloud9EnvironmentMembershipResourceType = "aws_cloud9_environment_membership"
)

var cloud9AllowEmptyValues = []string{"tags."}

type Cloud9Generator struct {
	AWSService
}

func (g *Cloud9Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloud9.NewFromConfig(config)
	environmentIDFilter := cloud9EnvironmentIDFilter(g.Filter)
	p := cloud9.NewListEnvironmentsPaginator(svc, &cloud9.ListEnvironmentsInput{})
	for p.HasMorePages() {
		output, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, environmentID := range output.EnvironmentIds {
			if !awsIDFilterAllows(environmentIDFilter, environmentID) {
				continue
			}
			if err := g.addEnvironment(svc, environmentID, cloud9ShouldEmitEnvironment(g.Filter, environmentID)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Cloud9Generator) addEnvironment(svc *cloud9.Client, environmentID string, emitEnvironment bool) error {
	if environmentID == "" {
		return nil
	}
	if importable, err := cloud9EnvironmentImportable(svc, environmentID); err != nil {
		return err
	} else if !importable {
		return nil
	}
	if emitEnvironment {
		g.Resources = append(g.Resources, newCloud9EnvironmentEC2Resource(environmentID))
	}
	if cloud9ShouldLoadEnvironmentMemberships(g.Filter) {
		if err := g.loadEnvironmentMemberships(svc, environmentID); err != nil {
			log.Printf("[WARN] Skipping Cloud9 environment memberships for %s: %v", environmentID, err)
		}
	}

	return nil
}

func cloud9EnvironmentImportable(svc *cloud9.Client, environmentID string) (bool, error) {
	details, err := svc.DescribeEnvironmentStatus(context.TODO(), &cloud9.DescribeEnvironmentStatusInput{
		EnvironmentId: &environmentID,
	})
	if cloud9EnvironmentNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("describe Cloud9 environment status for %s: %w", environmentID, err)
	}
	if details.Status == types.EnvironmentStatusError ||
		details.Status == types.EnvironmentStatusDeleting {
		return false, nil
	}
	return true, nil
}

func cloud9EnvironmentNotFound(err error) bool {
	var notFound *types.NotFoundException
	return errors.As(err, &notFound)
}

func newCloud9EnvironmentEC2Resource(environmentID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		environmentID,
		environmentID,
		cloud9EnvironmentEC2ResourceType,
		"aws",
		cloud9AllowEmptyValues)
}

func (g *Cloud9Generator) loadEnvironmentMemberships(svc *cloud9.Client, environmentID string) error {
	p := cloud9.NewDescribeEnvironmentMembershipsPaginator(svc, &cloud9.DescribeEnvironmentMembershipsInput{
		EnvironmentId: &environmentID,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, membership := range page.Memberships {
			if resource, ok := newCloud9EnvironmentMembershipResource(membership); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func newCloud9EnvironmentMembershipResource(membership types.EnvironmentMember) (terraformutils.Resource, bool) {
	environmentID := StringValue(membership.EnvironmentId)
	userArn := StringValue(membership.UserArn)
	if environmentID == "" || userArn == "" || !cloud9EnvironmentMembershipImportable(membership.Permissions) {
		return terraformutils.Resource{}, false
	}
	permissions := string(membership.Permissions)
	return terraformutils.NewResource(
		cloud9EnvironmentMembershipImportID(environmentID, userArn),
		cloud9ResourceName("membership", environmentID, userArn),
		cloud9EnvironmentMembershipResourceType,
		"aws",
		map[string]string{
			"environment_id": environmentID,
			"permissions":    permissions,
			"user_arn":       userArn,
		},
		cloud9AllowEmptyValues,
		map[string]interface{}{}), true
}

func cloud9EnvironmentMembershipImportable(permissions types.Permissions) bool {
	// Owner memberships are created by AWS for environment creators and are not independently managed by Terraform.
	return permissions == types.PermissionsReadOnly || permissions == types.PermissionsReadWrite
}

func cloud9ShouldEmitEnvironment(filters []terraformutils.ResourceFilter, environmentID string) bool {
	if !cloud9HasTypedFilter(filters) {
		return true
	}
	if !awsHasApplicableFilter(filters, cloud9EnvironmentEC2ResourceType) {
		return false
	}
	return awsIDFilterAllows(awsTypedIDFilterValues(filters, cloud9EnvironmentEC2ResourceType), environmentID)
}

func cloud9HasTypedFilter(filters []terraformutils.ResourceFilter) bool {
	return awsHasTypedFilter(filters, cloud9EnvironmentEC2ResourceType) ||
		awsHasTypedFilter(filters, cloud9EnvironmentMembershipResourceType)
}

func cloud9ShouldLoadEnvironmentMemberships(filters []terraformutils.ResourceFilter) bool {
	return !cloud9HasTypedFilter(filters) || awsHasTypedFilter(filters, cloud9EnvironmentMembershipResourceType)
}

func cloud9EnvironmentIDFilter(filters []terraformutils.ResourceFilter) map[string]bool {
	if cloud9HasEnvironmentScanAttributeFilter(filters) {
		return nil
	}
	membershipEnvironmentIDs, ok := cloud9EnvironmentIDsFromMembershipFilterValues(awsTypedIDFilterValues(filters, cloud9EnvironmentMembershipResourceType))
	if !ok {
		return nil
	}
	return awsMergeIDFilterValues(awsTypedIDFilterValues(filters, cloud9EnvironmentEC2ResourceType), membershipEnvironmentIDs)
}

func cloud9HasEnvironmentScanAttributeFilter(filters []terraformutils.ResourceFilter) bool {
	return awsHasApplicableNonIDFilter(filters, cloud9EnvironmentEC2ResourceType) ||
		awsHasApplicableNonIDFilter(filters, cloud9EnvironmentMembershipResourceType)
}

func cloud9EnvironmentIDsFromMembershipFilterValues(membershipIDs map[string]bool) (map[string]bool, bool) {
	if len(membershipIDs) == 0 {
		return nil, true
	}
	environmentIDs := map[string]bool{}
	for membershipID := range membershipIDs {
		environmentID, _, ok := strings.Cut(membershipID, "#")
		if !ok || environmentID == "" {
			return nil, false
		}
		environmentIDs[environmentID] = true
	}
	return environmentIDs, true
}

func cloud9EnvironmentMembershipImportID(environmentID, userArn string) string {
	return fmt.Sprintf("%s#%s", environmentID, userArn)
}

func cloud9ResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "cloud9-resource"
	}
	return strings.Join(cleanParts, "/")
}
