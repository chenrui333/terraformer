// SPDX-License-Identifier: Apache-2.0

package mackerel

import (
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/mackerelio/mackerel-client-go"
)

// NotificationGroupGenerator ...
type NotificationGroupGenerator struct {
	MackerelService
}

func (g *NotificationGroupGenerator) createResources(notificationGroups []*mackerel.NotificationGroup) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	for _, notificationGroup := range notificationGroups {
		resources = append(resources, g.createResource(notificationGroup.ID))
	}
	return resources
}

func (g *NotificationGroupGenerator) createResource(notificationGroupID string) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		notificationGroupID,
		fmt.Sprintf("notification_group_%s", notificationGroupID),
		"mackerel_notification_group",
		"mackerel",
		[]string{},
	)
}

// InitResources Generate TerraformResources from Mackerel API,
// from each notification group create 1 TerraformResource.
// Need Notification Group ID as ID for terraform resource
func (g *NotificationGroupGenerator) InitResources() error {
	client := g.Args["mackerelClient"].(*mackerel.Client)
	notificationGroups, err := client.FindNotificationGroups()
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, g.createResources(notificationGroups)...)
	return nil
}
