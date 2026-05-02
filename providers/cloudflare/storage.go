// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type StorageGenerator struct {
	CloudflareService
}

func (g *StorageGenerator) appendWorkersKVNamespaceResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListWorkersKVNamespacesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		namespaces, info, err := api.ListWorkersKVNamespaces(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, namespace := range namespaces {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				namespace.ID,
				cloudflareResourceName(accountID, namespace.Title, namespace.ID),
				"cloudflare_workers_kv_namespace",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *StorageGenerator) appendQueueResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListQueuesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		queues, info, err := api.ListQueues(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, queue := range queues {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				queue.ID,
				cloudflareResourceName(accountID, queue.Name, queue.ID),
				"cloudflare_queue",
				"cloudflare",
				map[string]string{"account_id": accountID, "queue_id": queue.ID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *StorageGenerator) appendR2BucketResources(ctx context.Context, api *cf.API, accountID string) error {
	buckets, err := api.ListR2Buckets(ctx, cf.AccountIdentifier(accountID), cf.ListR2BucketsParams{PerPage: cloudflarePageSize})
	if err != nil {
		return err
	}
	for _, bucket := range buckets {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			bucket.Name,
			cloudflareResourceName(accountID, bucket.Name),
			"cloudflare_r2_bucket",
			"cloudflare",
			map[string]string{
				"account_id":   accountID,
				"name":         bucket.Name,
				"jurisdiction": "default",
			},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func (g *StorageGenerator) appendD1DatabaseResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListD1DatabasesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		databases, info, err := api.ListD1Databases(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, database := range databases {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				database.UUID,
				cloudflareResourceName(accountID, database.Name, database.UUID),
				"cloudflare_d1_database",
				"cloudflare",
				map[string]string{"account_id": accountID, "uuid": database.UUID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *StorageGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	for _, f := range []func(context.Context, *cf.API, string) error{
		g.appendWorkersKVNamespaceResources,
		g.appendQueueResources,
		g.appendR2BucketResources,
		g.appendD1DatabaseResources,
	} {
		if err := f(ctx, api, account.Identifier); err != nil {
			return err
		}
	}
	return nil
}
