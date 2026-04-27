// SPDX-License-Identifier: Apache-2.0

package gmailfilter

import (
	"context"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"google.golang.org/api/gmail/v1"
)

type LabelGenerator struct {
	GmailfilterService
}

func (g LabelGenerator) createResources(labels []*gmail.Label) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, l := range labels {
		if l.Type == "system" {
			continue // ignore system labels
		}
		resources = append(resources, terraformutils.NewResource(
			l.Id,
			strings.ReplaceAll(l.Name, "/", "_"),
			"gmailfilter_label",
			"gmailfilter",
			map[string]string{},
			[]string{},
			map[string]interface{}{}))
	}
	return resources
}

func (g *LabelGenerator) InitResources() error {
	ctx := context.Background()
	gmailService, err := g.gmailService(ctx)
	if err != nil {
		return err
	}

	labels, err := gmailService.Users.Labels.List(gmailUser).Do()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(labels.Labels)...)

	return nil
}
