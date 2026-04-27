// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"log"
	"strings"

	"google.golang.org/api/pubsub/v1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var pubsubAllowEmptyValues = []string{""}

var pubsubAdditionalFields = map[string]interface{}{}

type PubsubGenerator struct {
	GCPService
}

// Run on subscriptionsList and create for each TerraformResource
func (g PubsubGenerator) createSubscriptionsResources(ctx context.Context, subscriptionsList *pubsub.ProjectsSubscriptionsListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := subscriptionsList.Pages(ctx, func(page *pubsub.ListSubscriptionsResponse) error {
		for _, obj := range page.Subscriptions {
			t := strings.Split(obj.Name, "/")
			name := t[len(t)-1]
			resources = append(resources, terraformutils.NewResource(
				name,
				obj.Name,
				"google_pubsub_subscription",
				g.ProviderName,
				map[string]string{
					"name":    name,
					"project": g.GetArgs()["project"].(string),
				},
				pubsubAllowEmptyValues,
				pubsubAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Run on topicsList and create for each TerraformResource
func (g PubsubGenerator) createTopicsListResources(ctx context.Context, topicsList *pubsub.ProjectsTopicsListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := topicsList.Pages(ctx, func(page *pubsub.ListTopicsResponse) error {
		for _, obj := range page.Topics {
			t := strings.Split(obj.Name, "/")
			name := t[len(t)-1]
			resources = append(resources, terraformutils.NewResource(
				g.GetArgs()["project"].(string)+"/"+name,
				obj.Name,
				"google_pubsub_topic",
				g.ProviderName,
				map[string]string{
					"name":    name,
					"project": g.GetArgs()["project"].(string),
				},
				pubsubAllowEmptyValues,
				pubsubAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
func (g *PubsubGenerator) InitResources() error {
	ctx := context.Background()
	pubsubService, err := pubsub.NewService(ctx)
	if err != nil {
		return err
	}

	subscriptionsList := pubsubService.Projects.Subscriptions.List("projects/" + g.GetArgs()["project"].(string))
	subscriptionsResources := g.createSubscriptionsResources(ctx, subscriptionsList)

	topicsList := pubsubService.Projects.Topics.List("projects/" + g.GetArgs()["project"].(string))
	topicsResources := g.createTopicsListResources(ctx, topicsList)

	g.Resources = append(g.Resources, subscriptionsResources...)
	g.Resources = append(g.Resources, topicsResources...)

	return nil
}

func (g *PubsubGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		for _, topic := range g.Resources {
			if r.InstanceState.Attributes["topic"] == "projects/"+g.GetArgs()["project"].(string)+"/topics/"+topic.InstanceState.Attributes["name"] {
				g.Resources[i].Item["topic"] = "${google_pubsub_topic." + topic.ResourceName + ".name}"
			}
		}
	}
	return nil
}
