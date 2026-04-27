// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"google.golang.org/api/gmail/v1"
)

type FilterGenerator struct {
	GmailfilterService
}

func (g FilterGenerator) createResources(filters []*gmail.Filter) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, f := range filters {
		resources = append(resources, terraformutils.NewResource(
			f.Id,
			f.Id,
			"gmailfilter_filter",
			"gmailfilter",
			map[string]string{},
			[]string{},
			map[string]interface{}{}))
	}
	return resources
}

func (g *FilterGenerator) InitResources() error {
	ctx := context.Background()
	gmailService, err := g.gmailService(ctx)
	if err != nil {
		return err
	}

	filters, err := gmailService.Users.Settings.Filters.List(gmailUser).Do()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(filters.Filter)...)

	return nil
}
