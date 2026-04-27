// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// AlertGroupSettingGenerator ...
type AlertGroupSettingGenerator struct {
	MackerelService
}

func (g *AlertGroupSettingGenerator) createResources(alertGroupSettings []*mackerel.AlertGroupSetting) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, alertGroupSetting := range alertGroupSettings {
		resources = append(resources, g.createResource(alertGroupSetting.ID))
	}
	return resources
}

func (g *AlertGroupSettingGenerator) createResource(alertGroupSettingID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		alertGroupSettingID,
		fmt.Sprintf("alert_group_setting_%s", alertGroupSettingID),
		"mackerel_alert_group_setting",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each alert group setting create 1 TerraformResource.
// Need Alert Group Setting ID as ID for terraform resource
func (g *AlertGroupSettingGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	alertGroupSettings, err := client.FindAlertGroupSettings()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(alertGroupSettings)...)
	return nil
}
