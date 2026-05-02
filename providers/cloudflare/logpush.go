// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type LogpushGenerator struct {
	CloudflareService
}

func (g *LogpushGenerator) appendLogpushJobResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	jobs, err := api.ListLogpushJobs(ctx, rc, cf.ListLogpushJobsParams{})
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
