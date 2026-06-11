// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
)

type LogpushGenerator struct {
	CloudflareService
}

func (g *LogpushGenerator) appendLogpushJobResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	jobs, err := listLogpushJobs(ctx, api, rc)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		jobID := strconv.Itoa(job.ID)
		g.Resources = append(g.Resources, terraformutils.NewResource(
			jobID,
			cloudflareResourceName(scopeType, rc.Identifier, job.Name, jobID),
			"cloudflare_logpush_job",
			"cloudflare",
			accessScopeAttributes(scopeType, rc.Identifier),
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func listLogpushJobs(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.LogpushJob, error) {
	var jobs []cf.LogpushJob
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/%s/%s/logpush/jobs?%s", rc.Level, rc.Identifier, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageJobs []cf.LogpushJob
		if err := json.Unmarshal(response.Result, &pageJobs); err != nil {
			return nil, err
		}
		jobs = append(jobs, pageJobs...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return jobs, nil
}

func (g *LogpushGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	if g.accountID() != "" {
		if err := g.appendLogpushJobResources(ctx, api, cf.AccountIdentifier(g.accountID()), "accounts"); err != nil {
			return err
		}
	}
	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendLogpushJobResources(ctx, api, cf.ZoneIdentifier(zone.ID), "zones"); err != nil {
			return err
		}
	}
	return nil
}
