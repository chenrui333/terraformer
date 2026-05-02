// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

type DestinationGenerator struct {
	LaunchDarklyService
}

func (g *DestinationGenerator) loadDestinations(ctx context.Context) error {
	for path := "/destinations"; path != ""; {
		destinations := &ldapi.Destinations{}
		if err := getLaunchDarklyAPI(ctx, g.GetArgs()["api_key"].(string), path, destinations); err != nil {
			return err
		}

		for _, destination := range destinations.GetItems() {
			projectKey, envKey, err := destinationProjectEnv(destination)
			if err != nil {
				return err
			}
			destinationID := destination.GetId()
			resource := terraformutils.NewResource(
				strings.Join([]string{projectKey, envKey, destinationID}, "/"),
				destinationResourceName(projectKey, envKey, destination.GetName(), destinationID),
				"launchdarkly_destination",
				"launchdarkly",
				map[string]string{
					"project_key": projectKey,
					"env_key":     envKey,
				},
				[]string{},
				map[string]interface{}{})
			g.Resources = append(g.Resources, resource)
		}
		path = nextPagePath(destinations.GetLinks())
	}
	return nil
}

func destinationResourceName(projectKey, envKey, name, destinationID string) string {
	return fmt.Sprintf("%s-%s-%s", projectKey, envKey, resourceNameWithID(name, destinationID))
}

func destinationProjectEnv(destination ldapi.Destination) (string, string, error) {
	self, ok := destination.GetLinks()["self"]
	if !ok || self.Href == nil {
		return "", "", fmt.Errorf("destination %q is missing self link", destination.GetId())
	}
	href := *self.Href
	if parsed, err := url.Parse(href); err == nil && parsed.Path != "" {
		href = parsed.Path
	}
	path := strings.TrimPrefix(href, "/api/v2/destinations/")
	if path == href {
		return "", "", fmt.Errorf("unexpected destination self link %q", *self.Href)
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("unexpected destination self link %q", *self.Href)
	}
	projectKey, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", err
	}
	envKey, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", err
	}
	return projectKey, envKey, nil
}

func (g *DestinationGenerator) InitResources() error {
	return g.loadDestinations(g.GetArgs()["ctx"].(context.Context))
}
