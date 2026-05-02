// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type PagesGenerator struct {
	CloudflareService
}

func (g *PagesGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	accountID := account.Identifier

	params := cf.ListPagesProjectsParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		projects, info, err := api.ListPagesProjects(ctx, account, params)
		if err != nil {
			return err
		}
		for _, project := range projects {
			projectResource := terraformutils.NewResource(
				project.Name,
				cloudflareResourceName(accountID, project.Name),
				"cloudflare_pages_project",
				"cloudflare",
				map[string]string{"account_id": accountID, "name": project.Name},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&projectResource, accountID+"/"+project.Name)
			g.Resources = append(g.Resources, projectResource)

			domains, err := api.GetPagesDomains(ctx, cf.PagesDomainsParameters{AccountID: accountID, ProjectName: project.Name})
			if err != nil {
				return err
			}
			for _, domain := range domains {
				domainResource := terraformutils.NewResource(
					domain.Name,
					cloudflareResourceName(accountID, project.Name, domain.Name),
					"cloudflare_pages_domain",
					"cloudflare",
					map[string]string{"account_id": accountID, "project_name": project.Name, "name": domain.Name},
					[]string{},
					map[string]interface{}{},
				)
				setCloudflareImportID(&domainResource, accountID+"/"+project.Name+"/"+domain.Name)
				g.Resources = append(g.Resources, domainResource)
			}
		}
		if !info.HasMorePages() {
			break
		}
		params.Page++
	}
	return nil
}
