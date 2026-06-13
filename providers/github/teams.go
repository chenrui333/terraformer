// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	githubAPI "github.com/google/go-github/v88/github"
)

type TeamsGenerator struct {
	GithubService
}

func (g *TeamsGenerator) createTeamsResources(ctx context.Context, teams []*githubAPI.Team, client *githubAPI.Client) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, team := range teams {
		resource := terraformutils.NewSimpleResource(
			strconv.FormatInt(team.GetID(), 10),
			team.GetName(),
			"github_team",
			"github",
			[]string{},
		)
		resource.SlowQueryRequired = true
		resources = append(resources, resource)
		memberResources, err := g.createTeamMembersResources(ctx, team, client)
		if err != nil {
			return nil, err
		}
		resources = append(resources, memberResources...)
		repositoryResources, err := g.createTeamRepositoriesResources(ctx, team, client)
		if err != nil {
			return nil, err
		}
		resources = append(resources, repositoryResources...)
	}
	return resources, nil
}

func (g *TeamsGenerator) createTeamMembersResources(ctx context.Context, team *githubAPI.Team, client *githubAPI.Client) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &githubAPI.TeamListTeamMembersOptions{
		ListOptions: githubAPI.ListOptions{PerPage: 100},
	}
	for {
		members, resp, err := client.Teams.ListTeamMembersBySlug(ctx, g.Args["owner"].(string), team.GetSlug(), opt)
		if err != nil {
			return nil, fmt.Errorf("list github team members for %s: %w", team.GetSlug(), err)
		}
		for _, member := range members {
			resources = append(resources, terraformutils.NewSimpleResource(
				strconv.FormatInt(team.GetID(), 10)+":"+member.GetLogin(),
				team.GetName()+"_"+member.GetLogin(),
				"github_team_membership",
				"github",
				[]string{},
			))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources, nil
}

func (g *TeamsGenerator) createTeamRepositoriesResources(ctx context.Context, team *githubAPI.Team, client *githubAPI.Client) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	opt := &githubAPI.ListOptions{PerPage: 100}
	for {
		repos, resp, err := client.Teams.ListTeamReposBySlug(ctx, g.Args["owner"].(string), team.GetSlug(), opt)
		if err != nil {
			return nil, fmt.Errorf("list github team repositories for %s: %w", team.GetSlug(), err)
		}
		for _, repo := range repos {
			resources = append(resources, terraformutils.NewSimpleResource(
				strconv.FormatInt(team.GetID(), 10)+":"+repo.GetName(),
				team.GetName()+"_"+repo.GetName(),
				"github_team_repository",
				"github",
				[]string{},
			))
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return resources, nil
}

// InitResources generates TerraformResources from Github API,
func (g *TeamsGenerator) InitResources() error {
	ctx := context.Background()
	client, err := g.createClient()
	if err != nil {
		return err
	}

	opt := &githubAPI.ListOptions{PerPage: 1}

	for {
		teams, resp, err := client.Teams.ListTeams(ctx, g.Args["owner"].(string), opt)
		if err != nil {
			return fmt.Errorf("list github teams for %s: %w", g.Args["owner"].(string), err)
		}

		resources, err := g.createTeamsResources(ctx, teams, client)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return nil
}

// PostConvertHook for connect between team and members
func (g *TeamsGenerator) PostConvertHook() error {
	for _, team := range g.Resources {
		if team.InstanceInfo.Type != "github_team" {
			continue
		}
		for i, member := range g.Resources {
			if member.InstanceInfo.Type != "github_team_membership" {
				continue
			}
			if member.InstanceState.Attributes["team_id"] == team.InstanceState.Attributes["id"] {
				g.Resources[i].Item["team_id"] = "${github_team." + team.ResourceName + ".id}"
			}
		}
		for i, repo := range g.Resources {
			if repo.InstanceInfo.Type != "github_team_repository" {
				continue
			}
			if repo.InstanceState.Attributes["team_id"] == team.InstanceState.Attributes["id"] {
				g.Resources[i].Item["team_id"] = "${github_team." + team.ResourceName + ".id}"
			}
		}
	}
	return nil
}
