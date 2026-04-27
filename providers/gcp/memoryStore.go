// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"log"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/redis/v1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var redisAllowEmptyValues = []string{""}

var redisAdditionalFields = map[string]interface{}{}

type MemoryStoreGenerator struct {
	GCPService
}

// Run on redisInstancesList and create for each TerraformResource
func (g MemoryStoreGenerator) createResources(ctx context.Context, redisInstancesList *redis.ProjectsLocationsInstancesListCall) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	if err := redisInstancesList.Pages(ctx, func(page *redis.ListInstancesResponse) error {
		for _, obj := range page.Instances {
			t := strings.Split(obj.Name, "/")
			name := t[len(t)-1]
			resources = append(resources, terraformutils.NewResource(
				obj.Name,
				name,
				"google_redis_instance",
				g.ProviderName,
				map[string]string{
					"name":    name,
					"project": g.GetArgs()["project"].(string),
					"region":  g.GetArgs()["region"].(compute.Region).Name,
				},
				redisAllowEmptyValues,
				redisAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		log.Println(err)
	}
	return resources
}

// Generate TerraformResources from GCP API,
// from each redis create 1 TerraformResource
// Need Redis name as ID for terraform resource
func (g *MemoryStoreGenerator) InitResources() error {
	ctx := context.Background()
	redisService, err := redis.NewService(ctx)
	if err != nil {
		return err
	}

	redisInstancesList := redisService.Projects.Locations.Instances.List("projects/" + g.GetArgs()["project"].(string) + "/locations/" + g.GetArgs()["region"].(compute.Region).Name)

	g.Resources = g.createResources(ctx, redisInstancesList)
	return nil
}
