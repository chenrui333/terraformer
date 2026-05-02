// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type StorageGenerator struct {
	CloudflareService
}

var r2BucketJurisdictions = []string{"default", "eu", "fedramp"}

type r2BucketListResult struct {
	Buckets []cf.R2Bucket
}

func cloudflareUnsupportedJurisdictionError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return cloudflareErrorIndicatesUnsupportedJurisdiction(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}
	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return cloudflareErrorIndicatesUnsupportedJurisdiction(requestErr.Error(), requestErr.ErrorMessages())
	}
	return false
}

func cloudflareErrorIndicatesUnsupportedJurisdiction(message string, errorMessages []string) bool {
	messages := append([]string{message}, errorMessages...)
	for _, msg := range messages {
		normalized := strings.ToLower(msg)
		if !strings.Contains(normalized, "jurisdiction") {
			continue
		}
		for _, marker := range []string{"not enabled", "not found", "not supported", "unsupported", "invalid", "unknown"} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func listR2BucketsInJurisdiction(
	ctx context.Context,
	api *cf.API,
	accountID string,
	jurisdiction string,
) ([]cf.R2Bucket, error) {
	var buckets []cf.R2Bucket
	cursor := ""
	for {
		values := url.Values{}
		values.Set("per_page", strconv.Itoa(cloudflarePageSize))
		if cursor != "" {
			values.Set("cursor", cursor)
		}
		headers := http.Header{}
		headers.Set("cf-r2-jurisdiction", jurisdiction)
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/r2/buckets?%s", accountID, values.Encode()),
			nil,
			headers,
		)
		if err != nil {
			if jurisdiction != "default" && cloudflareUnsupportedJurisdictionError(err) {
				return buckets, nil
			}
			return nil, err
		}

		var result r2BucketListResult
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return nil, err
		}
		buckets = append(buckets, result.Buckets...)

		if len(result.Buckets) < cloudflarePageSize || response.ResultInfo == nil || response.ResultInfo.Cursor == "" {
			break
		}
		cursor = response.ResultInfo.Cursor
	}
	return buckets, nil
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
	for _, jurisdiction := range r2BucketJurisdictions {
		buckets, err := listR2BucketsInJurisdiction(ctx, api, accountID, jurisdiction)
		if err != nil {
			return err
		}
		for _, bucket := range buckets {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				bucket.Name,
				cloudflareResourceName(accountID, jurisdiction, bucket.Name),
				"cloudflare_r2_bucket",
				"cloudflare",
				map[string]string{
					"account_id":   accountID,
					"name":         bucket.Name,
					"jurisdiction": jurisdiction,
				},
				[]string{},
				map[string]interface{}{},
			))
		}
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
