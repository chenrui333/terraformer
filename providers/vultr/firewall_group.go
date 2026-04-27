// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr"
)

type FirewallGroupGenerator struct {
	VultrService
}

func (g *FirewallGroupGenerator) loadFirewallGroups(client *govultr.Client) ([]govultr.FirewallGroup, error) {
	firewallGroups, err := client.FirewallGroup.List(context.Background())
	if err != nil {
		return nil, err
	}
	for _, firewallGroup := range firewallGroups {
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			firewallGroup.FirewallGroupID,
			firewallGroup.FirewallGroupID,
			"vultr_firewall_group",
			"vultr",
			[]string{}))
	}
	return firewallGroups, nil
}

func (g *FirewallGroupGenerator) loadFirewallRulesByIPType(client *govultr.Client, firewallGroupID string, ipType string) error {
	firewallRules, err := client.FirewallRule.ListByIPType(context.Background(), firewallGroupID, ipType)
	if err != nil {
		return err
	}
	for _, firewallRule := range firewallRules {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			strconv.Itoa(firewallRule.RuleNumber),
			strconv.Itoa(firewallRule.RuleNumber),
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
		err := g.loadFirewallRulesByIPType(client, firewallGroup.FirewallGroupID, "v4")
		if err != nil {
			return err
		}
		err = g.loadFirewallRulesByIPType(client, firewallGroup.FirewallGroupID, "v6")
		if err != nil {
			return err
		}
	}
	return nil
}
