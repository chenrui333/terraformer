// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/storage/v1"
)

var GcsAllowEmptyValues = []string{"labels.", "created_before"}

var GcsAdditionalFields = map[string]interface{}{}

type GcsGenerator struct {
	GCPService
}

func (g *GcsGenerator) createBucketsResources(ctx context.Context, gcsService *storage.Service) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	bucketList := gcsService.Buckets.List(g.GetArgs()["project"].(string))
	if err := bucketList.Pages(ctx, func(page *storage.Buckets) error {
		for _, bucket := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				bucket.Name,
				bucket.Name,
				"google_storage_bucket",
				g.ProviderName,
				map[string]string{
					"name":          bucket.Name,
					"force_destroy": "false",
				},
				GcsAllowEmptyValues,
				GcsAdditionalFields,
			))
			resources = append(resources, terraformutils.NewResource(
				bucket.Name,
				bucket.Name,
				"google_storage_bucket_acl",
				g.ProviderName,
				map[string]string{
					"bucket":        bucket.Name,
					"role_entity.#": strconv.Itoa(len(bucket.Acl)),
				},
				GcsAllowEmptyValues,
				GcsAdditionalFields,
			))
			resources = append(resources, terraformutils.NewResource(
				bucket.Name,
				bucket.Name,
				"google_storage_default_object_acl",
				g.ProviderName,
				map[string]string{
					"bucket":        bucket.Name,
					"role_entity.#": strconv.Itoa(len(bucket.Acl)),
				},
				GcsAllowEmptyValues,
				GcsAdditionalFields,
			))

			resources = append(resources, terraformutils.NewResource(
				bucket.Name,
				bucket.Name,
				"google_storage_bucket_iam_policy",
				g.ProviderName,
				map[string]string{
					"bucket": bucket.Name,
				},
				GcsAllowEmptyValues,
				GcsAdditionalFields,
			))

			if iam, err := gcsService.Buckets.GetIamPolicy(bucket.Name).Do(); err == nil {
				for _, binding := range iam.Bindings {
					resources = append(resources, terraformutils.NewResource(
						bucket.Name,
						bucket.Name,
						"google_storage_bucket_iam_binding",
						g.ProviderName,
						map[string]string{
							"bucket": bucket.Name,
							"role":   binding.Role,
						},
						GcsAllowEmptyValues,
						GcsAdditionalFields,
					))

					for _, member := range binding.Members {
						resources = append(resources, terraformutils.NewResource(
							bucket.Name,
							bucket.Name,
							"google_storage_bucket_iam_member",
							g.ProviderName,
							map[string]string{
								"bucket": bucket.Name,
								"role":   binding.Role,
								"member": member,
							},
							GcsAllowEmptyValues,
							GcsAdditionalFields,
						))
					}
				}
			}

			notificationResources, err := g.createNotificationResources(gcsService, bucket)
			if err != nil {
				return err
			}
			resources = append(resources, notificationResources...)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list gcs buckets: %w", err)
	}
	return resources, nil
}

func (g *GcsGenerator) createNotificationResources(gcsService *storage.Service, bucket *storage.Bucket) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	notificationList, err := gcsService.Notifications.List(bucket.Name).Do()
	if err != nil {
		return nil, fmt.Errorf("list gcs notifications for %s: %w", bucket.Name, err)
	}
	for _, notification := range notificationList.Items {
		resources = append(resources, terraformutils.NewResource(
			bucket.Name+"/notificationConfigs/"+notification.Id,
			bucket.Name+"/"+notification.Id,
			"google_storage_notification",
			g.ProviderName,
			map[string]string{},
			GcsAllowEmptyValues,
			GcsAdditionalFields,
		))
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each bucket  create 1 TerraformResource
// Need bucket name as ID for terraform resource
func (g *GcsGenerator) InitResources() error {
	ctx := context.Background()
	gcsService, err := storage.NewService(ctx)
	if err != nil {
		return err
	}
	resources, err := g.createBucketsResources(ctx, gcsService)
	if err != nil {
		return err
	}
	g.Resources = resources

	// TODO find bug with storageTransferService.TransferJobs.List().Pages
	// storageTransferService, err := storagetransfer.NewService(ctx)
	// if err != nil {
	// 	log.Print(err)
	// 		return err
	// 	}
	// g.Resources = append(g.Resources, g.createTransferJobsResources(ctx, storageTransferService)...)
	return nil
}

// PostGenerateHook for add bucket policy json as heredoc
// support only bucket with policy
func (g *GcsGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != "google_storage_bucket_iam_policy" {
			continue
		}
		if _, exist := resource.Item["policy_data"]; exist {
			policy := resource.Item["policy_data"].(string)
			g.Resources[i].Item["policy_data"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
		}
	}
	return nil
}
