// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type GroupGenerator struct {
	GitLabService
}

// Generate TerraformResources from gitlab API,
func (g *GroupGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	group := g.Args["group"].(string)
	resources, err := createGroups(ctx, client, group)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}

func createGroups(ctx context.Context, client *gitlab.Client, groupID string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	group, _, err := client.Groups.GetGroup(groupID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("get gitlab group %s: %w", groupID, err)
	}

	resource := terraformutils.NewSimpleResource(
		fmt.Sprintf("%d", group.ID),
		getGroupResourceName(group),
		"gitlab_group",
		"gitlab",
		[]string{},
	)

	// NOTE: mirror fields from API doesn't match with the ones from terraform provider
	resource.IgnoreKeys = []string{"mirror_trigger_builds", "only_mirror_protected_branches", "mirror", "mirror_overwrites_diverged_branches"}

	resource.SlowQueryRequired = true
	resources = append(resources, resource)
	variableResources, err := createGroupVariables(ctx, client, group)
	if err != nil {
		return nil, err
	}
	resources = append(resources, variableResources...)
	membershipResources, err := createGroupMembership(ctx, client, group)
	if err != nil {
		return nil, err
	}
	resources = append(resources, membershipResources...)

	return resources, nil
}

func createGroupVariables(ctx context.Context, client *gitlab.Client, group *gitlab.Group) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListGroupVariablesOptions{}

	for {
		groupVariables, resp, err := client.GroupVariables.ListVariables(group.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab group variables for %d: %w", group.ID, err)
		}

		for _, groupVariable := range groupVariables {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%s:%s", group.ID, groupVariable.Key, groupVariable.EnvironmentScope),
				fmt.Sprintf("%s___%s___%s", getGroupResourceName(group), groupVariable.Key, groupVariable.EnvironmentScope),
				"gitlab_group_variable",
				"gitlab",
				[]string{},
			)
			resource.SlowQueryRequired = true
			resources = append(resources, resource)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources, nil
}

func createGroupMembership(ctx context.Context, client *gitlab.Client, group *gitlab.Group) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListGroupMembersOptions{}

	for {
		groupMembers, resp, err := client.Groups.ListGroupMembers(group.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab group memberships for %d: %w", group.ID, err)
		}

		for _, groupMember := range groupMembers {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%d", group.ID, groupMember.ID),
				fmt.Sprintf("%s___%s", getGroupResourceName(group), groupMember.Username),
				"gitlab_group_membership",
				"gitlab",
				[]string{},
			)
			resource.SlowQueryRequired = true
			resources = append(resources, resource)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources, nil
}

func getGroupResourceName(group *gitlab.Group) string {
	return fmt.Sprintf("%d___%s", group.ID, strings.ReplaceAll(group.FullPath, "/", "__"))
}
