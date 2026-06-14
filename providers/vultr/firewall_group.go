// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type FirewallGroupGenerator struct {
	VultrService
}

func (g *FirewallGroupGenerator) loadFirewallGroups(client *govultr.Client) ([]govultr.FirewallGroup, error) {
	firewallGroups, err := listAllVultrResources(context.Background(), client.FirewallGroup.List)
	if err != nil {
		return nil, fmt.Errorf("list vultr firewall groups: %w", err)
	}
	for _, firewallGroup := range firewallGroups {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			firewallGroup.ID,
			firewallGroup.ID,
			"vultr_firewall_group",
			"vultr",
			[]string{}))
	}
	return firewallGroups, nil
}

func (g *FirewallGroupGenerator) loadFirewallRules(client *govultr.Client, firewallGroupID string) error {
	firewallRules, err := listAllVultrResources(context.Background(), func(ctx context.Context, opt *govultr.ListOptions) ([]govultr.FirewallRule, *govultr.Meta, *http.Response, error) {
		return client.FirewallRule.List(ctx, firewallGroupID, opt)
	})
	if err != nil {
		return fmt.Errorf("list vultr firewall rules for %q: %w", firewallGroupID, err)
	}
	for _, ipType := range []string{"v4", "v6"} {
		for _, firewallRule := range firewallRules {
			if firewallRule.IPType != ipType {
				continue
			}

			g.Resources = append(g.Resources, terraformutils.NewResource(
				strconv.Itoa(firewallRule.ID),
				strconv.Itoa(firewallRule.ID),
				"vultr_firewall_rule",
				"vultr",
				map[string]string{
					"firewall_group_id": firewallGroupID,
					"ip_type":           ipType,
				},
				[]string{},
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *FirewallGroupGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	firewallGroups, err := g.loadFirewallGroups(client)
	if err != nil {
		return err
	}
	for _, firewallGroup := range firewallGroups {
		err := g.loadFirewallRules(client, firewallGroup.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
