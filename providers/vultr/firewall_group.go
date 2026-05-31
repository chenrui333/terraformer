// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type FirewallGroupGenerator struct {
	VultrService
}

func (g *FirewallGroupGenerator) loadFirewallGroups(client *govultr.Client) ([]govultr.FirewallGroup, error) {
	firewallGroups, _, _, err := client.FirewallGroup.List(context.Background(), nil)
	if err != nil {
		return nil, err
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

func (g *FirewallGroupGenerator) loadFirewallRulesByIPType(client *govultr.Client, firewallGroupID string, ipType string) error {
	firewallRules, _, _, err := client.FirewallRule.List(context.Background(), firewallGroupID, nil)
	if err != nil {
		return err
	}
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
	return nil
}

func (g *FirewallGroupGenerator) InitResources() error {
	client := g.generateClient()
	firewallGroups, err := g.loadFirewallGroups(client)
	if err != nil {
		return err
	}
	for _, firewallGroup := range firewallGroups {
		err := g.loadFirewallRulesByIPType(client, firewallGroup.ID, "v4")
		if err != nil {
			return err
		}
		err = g.loadFirewallRulesByIPType(client, firewallGroup.ID, "v6")
		if err != nil {
			return err
		}
	}
	return nil
}
