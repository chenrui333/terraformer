// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/pagination"
)

type NetworkingGenerator struct {
	OpenStackService
}

// createResources iterate on all openstack_networking_secgroup_v2
func (g *NetworkingGenerator) createSecgroupResources(list *pagination.Pager) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	err := list.EachPage(func(page pagination.Page) (bool, error) {
		groups, err := groups.ExtractGroups(page)
		if err != nil {
			return false, err
		}

		for _, grp := range groups {
			resource := terraformutils.NewSimpleResource(
				grp.ID,
				grp.Name,
				"openstack_networking_secgroup_v2",
				"openstack",
				[]string{},
			)
			resources = append(resources, resource)
			resources = append(resources, g.createSecgroupRuleResources(grp.Rules)...)
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return resources, nil
}

// createResources iterate on all openstack_networking_secgroup_v2
func (g *NetworkingGenerator) createSecgroupRuleResources(rules []rules.SecGroupRule) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, r := range rules {
		resource := terraformutils.NewSimpleResource(
			r.ID,
			r.ID,
			"openstack_networking_secgroup_rule_v2",
			"openstack",
			[]string{},
		)
		resources = append(resources, resource)
	}
	return resources
}

// Generate TerraformResources from OpenStack API,
func (g *NetworkingGenerator) InitResources() error {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return err
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region: g.GetArgs()["region"].(string),
	})
	if err != nil {
		return err
	}

	list := groups.List(client, groups.ListOpts{})

	resources, err := g.createSecgroupResources(&list)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil
}

func (g *NetworkingGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "openstack_networking_secgroup_rule_v2" {
			continue
		}
		for _, sg := range g.Resources {
			if sg.InstanceInfo.Type != "openstack_networking_secgroup_v2" {
				continue
			}
			if r.InstanceState.Attributes["security_group_id"] == sg.InstanceState.Attributes["id"] {
				g.Resources[i].Item["security_group_id"] = "${openstack_networking_secgroup_v2." + sg.ResourceName + ".id}"
			}
		}
	}

	return nil
}
