// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	gitlab "gitlab.com/gitlab-org/api/client-go/v2"
)

type ProjectGenerator struct {
	GitLabService
}

// Generate TerraformResources from gitlab API,
func (g *ProjectGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	group := g.Args["group"].(string)
	resources, err := createProjects(ctx, client, group)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, resources...)

	return nil
}

func createProjects(ctx context.Context, client *gitlab.Client, group string) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	for {
		projects, resp, err := client.Groups.ListGroupProjects(group, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab projects for %s: %w", group, err)
		}

		for _, project := range projects {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d", project.ID),
				getProjectResourceName(project),
				"gitlab_project",
				"gitlab",
				[]string{},
			)

			// NOTE: mirror fields from API doesn't match with the ones from terraform provider
			resource.IgnoreKeys = []string{"mirror_trigger_builds", "only_mirror_protected_branches", "mirror", "mirror_overwrites_diverged_branches"}

			resource.SlowQueryRequired = true
			resources = append(resources, resource)
			variableResources, err := createProjectVariables(ctx, client, project)
			if err != nil {
				return nil, err
			}
			resources = append(resources, variableResources...)
			branchProtectionResources, err := createBranchProtections(ctx, client, project)
			if err != nil {
				return nil, err
			}
			resources = append(resources, branchProtectionResources...)
			tagProtectionResources, err := createTagProtections(ctx, client, project)
			if err != nil {
				return nil, err
			}
			resources = append(resources, tagProtectionResources...)
			membershipResources, err := createProjectMembership(ctx, client, project)
			if err != nil {
				return nil, err
			}
			resources = append(resources, membershipResources...)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources, nil
}

func createProjectVariables(ctx context.Context, client *gitlab.Client, project *gitlab.Project) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListProjectVariablesOptions{}

	for {
		projectVariables, resp, err := client.ProjectVariables.ListVariables(project.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab project variables for %d: %w", project.ID, err)
		}

		for _, projectVariable := range projectVariables {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%s:%s", project.ID, projectVariable.Key, projectVariable.EnvironmentScope),
				fmt.Sprintf("%s___%s___%s", getProjectResourceName(project), projectVariable.Key, projectVariable.EnvironmentScope),
				"gitlab_project_variable",
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

func createBranchProtections(ctx context.Context, client *gitlab.Client, project *gitlab.Project) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListProtectedBranchesOptions{}

	for {
		protectedBranches, resp, err := client.ProtectedBranches.ListProtectedBranches(project.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab branch protections for %d: %w", project.ID, err)
		}

		for _, protectedBranch := range protectedBranches {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%s", project.ID, protectedBranch.Name),
				fmt.Sprintf("%s___%s", getProjectResourceName(project), protectedBranch.Name),
				"gitlab_branch_protection",
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

func createTagProtections(ctx context.Context, client *gitlab.Client, project *gitlab.Project) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListProtectedTagsOptions{}

	for {
		protectedTags, resp, err := client.ProtectedTags.ListProtectedTags(project.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab tag protections for %d: %w", project.ID, err)
		}

		for _, protectedTag := range protectedTags {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%s", project.ID, protectedTag.Name),
				fmt.Sprintf("%s___%s", getProjectResourceName(project), protectedTag.Name),
				"gitlab_tag_protection",
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

func createProjectMembership(ctx context.Context, client *gitlab.Client, project *gitlab.Project) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &gitlab.ListProjectMembersOptions{}

	for {
		projectMembers, resp, err := client.ProjectMembers.ListProjectMembers(project.ID, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("list gitlab project memberships for %d: %w", project.ID, err)
		}

		for _, projectMember := range projectMembers {
			resource := terraformutils.NewSimpleResource(
				fmt.Sprintf("%d:%d", project.ID, projectMember.ID),
				fmt.Sprintf("%s___%s", getProjectResourceName(project), projectMember.Username),
				"gitlab_project_membership",
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

func getProjectResourceName(project *gitlab.Project) string {
	return fmt.Sprintf("%d___%s", project.ID, strings.ReplaceAll(project.PathWithNamespace, "/", "__"))
}
