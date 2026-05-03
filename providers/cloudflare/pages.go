// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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

			domains, err := listPagesDomains(ctx, api, accountID, project.Name)
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

func listPagesDomains(ctx context.Context, api *cf.API, accountID, projectName string) ([]cf.PagesDomain, error) {
	var domains []cf.PagesDomain
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/pages/projects/%s/domains?%s", accountID, projectName, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageDomains []cf.PagesDomain
		if err := json.Unmarshal(response.Result, &pageDomains); err != nil {
			return nil, err
		}
		domains = append(domains, pageDomains...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return domains, nil
}
