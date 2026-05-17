// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	cf "github.com/cloudflare/cloudflare-go"
)

type StorageGenerator struct {
	CloudflareService
}

var r2BucketJurisdictions = []string{"default", "eu", "fedramp"}

type r2BucketListResult struct {
	Buckets []cf.R2Bucket
}

type cloudflareQueueConsumer struct {
	ConsumerID      string `json:"consumer_id"`
	DeadLetterQueue string `json:"dead_letter_queue"`
	ScriptName      string `json:"script_name"`
	Type            string `json:"type"`
}

type cloudflareR2RulesResponse struct {
	Rules []json.RawMessage `json:"rules"`
}

type cloudflareR2BucketEventNotificationList struct {
	Queues []cloudflareR2BucketEventNotificationQueue `json:"queues"`
}

type cloudflareR2BucketEventNotificationQueue struct {
	QueueID   string                                    `json:"queueId"`
	QueueName string                                    `json:"queueName"`
	Rules     []cloudflareR2BucketEventNotificationRule `json:"rules"`
}

type cloudflareR2BucketEventNotificationRule struct {
	Actions     []string `json:"actions"`
	Description string   `json:"description"`
	Prefix      string   `json:"prefix"`
	Suffix      string   `json:"suffix"`
}

type cloudflareR2CustomDomainList struct {
	Domains []cloudflareR2CustomDomain `json:"domains"`
}

type cloudflareR2CustomDomain struct {
	Ciphers  []string `json:"ciphers"`
	Domain   string   `json:"domain"`
	Enabled  bool     `json:"enabled"`
	MinTLS   string   `json:"minTLS"`
	ZoneID   string   `json:"zoneId"`
	ZoneName string   `json:"zoneName"`
}

type cloudflareR2DataCatalog struct {
	Bucket           string `json:"bucket"`
	CredentialStatus string `json:"credential_status"`
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
}

type cloudflareStorageChildDiscovery struct {
	name     string
	parent   string
	discover func() error
}

type cloudflareStorageFamilyDiscovery struct {
	name      string
	account   string
	resources *[]terraformutils.Resource
	discover  func() error
}

func runCloudflareStorageChildDiscoveries(discoveries []cloudflareStorageChildDiscovery) {
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		if err := discovery.discover(); err != nil {
			log.Printf("Skipping Cloudflare storage %s discovery for %s: %v", discovery.name, discovery.parent, err)
		}
	}
}

func runCloudflareStorageFamilyDiscoveries(discoveries []cloudflareStorageFamilyDiscovery) error {
	successes := 0
	var firstErr error
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		resourceCount := 0
		if discovery.resources != nil {
			resourceCount = len(*discovery.resources)
		}
		if err := discovery.discover(); err != nil {
			if discovery.resources != nil && resourceCount <= len(*discovery.resources) {
				*discovery.resources = (*discovery.resources)[:resourceCount]
			}
			if firstErr == nil {
				firstErr = err
			}
			log.Printf("Skipping Cloudflare storage %s discovery for %s: %v", discovery.name, discovery.account, err)
			continue
		}
		successes++
	}
	if successes == 0 && firstErr != nil {
		return firstErr
	}
	return nil
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

func cloudflareR2JurisdictionHeaders(jurisdiction string) http.Header {
	headers := http.Header{}
	if jurisdiction != "" {
		headers.Set("cf-r2-jurisdiction", jurisdiction)
	}
	return headers
}

func cloudflareRawGetOptional(
	ctx context.Context,
	api *cf.API,
	path string,
	headers http.Header,
) (json.RawMessage, bool, error) {
	response, err := api.Raw(ctx, http.MethodGet, path, nil, headers)
	if err != nil {
		if cloudflareNotFoundError(err) || cloudflareUnsupportedJurisdictionError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(response.Result) == 0 || string(response.Result) == "null" {
		return nil, false, nil
	}
	return response.Result, true, nil
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

func addCloudflareStringListAttributes(attributes map[string]string, name string, values []string) {
	attributes[name+".#"] = strconv.Itoa(len(values))
	for i, value := range values {
		attributes[fmt.Sprintf("%s.%d", name, i)] = value
	}
}

func setCloudflarePreserveIDAfterRefresh(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil {
		return
	}
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
}

func newCloudflareQueueConsumerResource(
	accountID string,
	queue cf.Queue,
	consumer cloudflareQueueConsumer,
) (terraformutils.Resource, bool) {
	if queue.ID == "" || consumer.ConsumerID == "" || consumer.Type == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"account_id":  accountID,
		"consumer_id": consumer.ConsumerID,
		"queue_id":    queue.ID,
		"type":        consumer.Type,
	}
	if consumer.DeadLetterQueue != "" {
		attributes["dead_letter_queue"] = consumer.DeadLetterQueue
	}
	if consumer.ScriptName != "" {
		attributes["script_name"] = consumer.ScriptName
	}
	resource := terraformutils.NewResource(
		cloudflareResourceName(accountID, queue.ID, consumer.ConsumerID),
		cloudflareResourceName(accountID, queue.Name, queue.ID, consumer.ConsumerID),
		"cloudflare_queue_consumer",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newCloudflareR2BucketConfigResource(
	accountID string,
	bucketName string,
	jurisdiction string,
	resourceType string,
) terraformutils.Resource {
	resourceName := strings.TrimPrefix(resourceType, "cloudflare_")
	resource := terraformutils.NewResource(
		cloudflareResourceName(accountID, bucketName, jurisdiction, resourceName),
		cloudflareResourceName(accountID, jurisdiction, bucketName, resourceName),
		resourceType,
		"cloudflare",
		map[string]string{
			"account_id":   accountID,
			"bucket_name":  bucketName,
			"jurisdiction": jurisdiction,
		},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource
}

func newCloudflareR2BucketEventNotificationResource(
	accountID string,
	bucketName string,
	jurisdiction string,
	queue cloudflareR2BucketEventNotificationQueue,
) (terraformutils.Resource, bool) {
	if queue.QueueID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"account_id":   accountID,
		"bucket_name":  bucketName,
		"jurisdiction": jurisdiction,
		"queue_id":     queue.QueueID,
	}
	if queue.QueueName != "" {
		attributes["queue_name"] = queue.QueueName
	}
	validRules := 0
	for _, rule := range queue.Rules {
		if len(rule.Actions) == 0 {
			continue
		}
		prefix := fmt.Sprintf("rules.%d", validRules)
		addCloudflareStringListAttributes(attributes, prefix+".actions", rule.Actions)
		if rule.Description != "" {
			attributes[prefix+".description"] = rule.Description
		}
		if rule.Prefix != "" {
			attributes[prefix+".prefix"] = rule.Prefix
		}
		if rule.Suffix != "" {
			attributes[prefix+".suffix"] = rule.Suffix
		}
		validRules++
	}
	if validRules == 0 {
		return terraformutils.Resource{}, false
	}
	attributes["rules.#"] = strconv.Itoa(validRules)
	resource := terraformutils.NewResource(
		cloudflareResourceName(accountID, bucketName, jurisdiction, queue.QueueID),
		cloudflareResourceName(accountID, jurisdiction, bucketName, queue.QueueName, queue.QueueID),
		"cloudflare_r2_bucket_event_notification",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newCloudflareR2CustomDomainResource(
	accountID string,
	bucketName string,
	jurisdiction string,
	domain cloudflareR2CustomDomain,
) (terraformutils.Resource, bool) {
	if domain.Domain == "" || domain.ZoneID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"account_id":   accountID,
		"bucket_name":  bucketName,
		"domain":       domain.Domain,
		"enabled":      strconv.FormatBool(domain.Enabled),
		"jurisdiction": jurisdiction,
		"zone_id":      domain.ZoneID,
	}
	if domain.MinTLS != "" {
		attributes["min_tls"] = domain.MinTLS
	}
	if len(domain.Ciphers) > 0 {
		addCloudflareStringListAttributes(attributes, "ciphers", domain.Ciphers)
	}
	if domain.ZoneName != "" {
		attributes["zone_name"] = domain.ZoneName
	}
	resource := terraformutils.NewResource(
		cloudflareResourceName(accountID, bucketName, jurisdiction, domain.Domain),
		cloudflareResourceName(accountID, jurisdiction, bucketName, domain.Domain),
		"cloudflare_r2_custom_domain",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflarePreserveIDAfterRefresh(&resource)
	return resource, true
}

func newCloudflareR2DataCatalogResource(
	accountID string,
	bucketName string,
	catalog cloudflareR2DataCatalog,
) (terraformutils.Resource, bool) {
	if catalog.Status != "active" {
		return terraformutils.Resource{}, false
	}
	if catalog.Bucket != "" {
		bucketName = catalog.Bucket
	}
	if bucketName == "" {
		return terraformutils.Resource{}, false
	}
	resourceID := catalog.ID
	if resourceID == "" {
		resourceID = bucketName
	}
	resource := terraformutils.NewResource(
		resourceID,
		cloudflareResourceName(accountID, bucketName, "data_catalog"),
		"cloudflare_r2_data_catalog",
		"cloudflare",
		map[string]string{
			"account_id":  accountID,
			"bucket_name": bucketName,
		},
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, accountID+"/"+bucketName)
	return resource, true
}

func (g *StorageGenerator) appendWorkersKVNamespaceResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListWorkersKVNamespacesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		namespaces, info, err := api.ListWorkersKVNamespaces(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, namespace := range namespaces {
			resource := terraformutils.NewResource(
				namespace.ID,
				cloudflareResourceName(accountID, namespace.Title, namespace.ID),
				"cloudflare_workers_kv_namespace",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, accountID+"/"+namespace.ID)
			g.Resources = append(g.Resources, resource)
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *StorageGenerator) appendQueueConsumerResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	queue cf.Queue,
) error {
	if queue.ID == "" {
		return nil
	}
	path := fmt.Sprintf(
		"/accounts/%s/queues/%s/consumers",
		accountID,
		url.PathEscape(queue.ID),
	)
	result, found, err := cloudflareRawGetOptional(ctx, api, path, nil)
	if err != nil || !found {
		return err
	}
	var consumers []cloudflareQueueConsumer
	if err := json.Unmarshal(result, &consumers); err != nil {
		return err
	}
	for _, consumer := range consumers {
		resource, ok := newCloudflareQueueConsumerResource(accountID, queue, consumer)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
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
			resource := terraformutils.NewResource(
				queue.ID,
				cloudflareResourceName(accountID, queue.Name, queue.ID),
				"cloudflare_queue",
				"cloudflare",
				map[string]string{"account_id": accountID, "queue_id": queue.ID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, accountID+"/"+queue.ID)
			g.Resources = append(g.Resources, resource)
			runCloudflareStorageChildDiscoveries([]cloudflareStorageChildDiscovery{{
				name:   "queue consumers",
				parent: cloudflareResourceName(accountID, queue.ID),
				discover: func() error {
					return g.appendQueueConsumerResources(ctx, api, accountID, queue)
				},
			}})
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *StorageGenerator) appendR2BucketConfigChildResource(
	ctx context.Context,
	api *cf.API,
	accountID string,
	bucketName string,
	jurisdiction string,
	resourceType string,
	pathSuffix string,
) error {
	path := fmt.Sprintf(
		"/accounts/%s/r2/buckets/%s/%s",
		accountID,
		url.PathEscape(bucketName),
		pathSuffix,
	)
	result, found, err := cloudflareRawGetOptional(ctx, api, path, cloudflareR2JurisdictionHeaders(jurisdiction))
	if err != nil || !found {
		return err
	}
	var rules cloudflareR2RulesResponse
	if err := json.Unmarshal(result, &rules); err != nil {
		return err
	}
	if len(rules.Rules) == 0 {
		return nil
	}
	g.Resources = append(g.Resources, newCloudflareR2BucketConfigResource(accountID, bucketName, jurisdiction, resourceType))
	return nil
}

func (g *StorageGenerator) appendR2BucketEventNotificationResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	bucketName string,
	jurisdiction string,
) error {
	path := fmt.Sprintf(
		"/accounts/%s/event_notifications/r2/%s/configuration",
		accountID,
		url.PathEscape(bucketName),
	)
	result, found, err := cloudflareRawGetOptional(ctx, api, path, cloudflareR2JurisdictionHeaders(jurisdiction))
	if err != nil || !found {
		return err
	}
	var notifications cloudflareR2BucketEventNotificationList
	if err := json.Unmarshal(result, &notifications); err != nil {
		return err
	}
	for _, queue := range notifications.Queues {
		resource, ok := newCloudflareR2BucketEventNotificationResource(accountID, bucketName, jurisdiction, queue)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *StorageGenerator) appendR2CustomDomainResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	bucketName string,
	jurisdiction string,
) error {
	path := fmt.Sprintf(
		"/accounts/%s/r2/buckets/%s/domains/custom",
		accountID,
		url.PathEscape(bucketName),
	)
	result, found, err := cloudflareRawGetOptional(ctx, api, path, cloudflareR2JurisdictionHeaders(jurisdiction))
	if err != nil || !found {
		return err
	}
	var domains cloudflareR2CustomDomainList
	if err := json.Unmarshal(result, &domains); err != nil {
		return err
	}
	for _, domain := range domains.Domains {
		resource, ok := newCloudflareR2CustomDomainResource(accountID, bucketName, jurisdiction, domain)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *StorageGenerator) appendR2DataCatalogResource(
	ctx context.Context,
	api *cf.API,
	accountID string,
	bucketName string,
) error {
	path := fmt.Sprintf(
		"/accounts/%s/r2-catalog/%s",
		accountID,
		url.PathEscape(bucketName),
	)
	result, found, err := cloudflareRawGetOptional(ctx, api, path, nil)
	if err != nil || !found {
		return err
	}
	var catalog cloudflareR2DataCatalog
	if err := json.Unmarshal(result, &catalog); err != nil {
		return err
	}
	resource, ok := newCloudflareR2DataCatalogResource(accountID, bucketName, catalog)
	if ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *StorageGenerator) appendR2BucketChildResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
	bucketName string,
	jurisdiction string,
	includeDataCatalog bool,
) {
	parent := cloudflareResourceName(accountID, bucketName, jurisdiction)
	discoveries := []cloudflareStorageChildDiscovery{
		{
			name:   "R2 bucket CORS",
			parent: parent,
			discover: func() error {
				return g.appendR2BucketConfigChildResource(ctx, api, accountID, bucketName, jurisdiction, "cloudflare_r2_bucket_cors", "cors")
			},
		},
		{
			name:   "R2 bucket lifecycle",
			parent: parent,
			discover: func() error {
				return g.appendR2BucketConfigChildResource(ctx, api, accountID, bucketName, jurisdiction, "cloudflare_r2_bucket_lifecycle", "lifecycle")
			},
		},
		{
			name:   "R2 bucket lock",
			parent: parent,
			discover: func() error {
				return g.appendR2BucketConfigChildResource(ctx, api, accountID, bucketName, jurisdiction, "cloudflare_r2_bucket_lock", "lock")
			},
		},
		{
			name:   "R2 bucket event notifications",
			parent: parent,
			discover: func() error {
				return g.appendR2BucketEventNotificationResources(ctx, api, accountID, bucketName, jurisdiction)
			},
		},
		{
			name:   "R2 custom domains",
			parent: parent,
			discover: func() error {
				return g.appendR2CustomDomainResources(ctx, api, accountID, bucketName, jurisdiction)
			},
		},
	}
	if includeDataCatalog {
		discoveries = append(discoveries, cloudflareStorageChildDiscovery{
			name:   "R2 data catalog",
			parent: parent,
			discover: func() error {
				return g.appendR2DataCatalogResource(ctx, api, accountID, bucketName)
			},
		})
	}
	runCloudflareStorageChildDiscoveries(discoveries)
}

func (g *StorageGenerator) appendR2BucketResources(ctx context.Context, api *cf.API, accountID string) error {
	seenDataCatalogBuckets := map[string]bool{}
	for _, jurisdiction := range r2BucketJurisdictions {
		buckets, err := listR2BucketsInJurisdiction(ctx, api, accountID, jurisdiction)
		if err != nil {
			return err
		}
		for _, bucket := range buckets {
			resource := terraformutils.NewResource(
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
			)
			setCloudflareImportID(&resource, accountID+"/"+bucket.Name+"/"+jurisdiction)
			g.Resources = append(g.Resources, resource)

			includeDataCatalog := !seenDataCatalogBuckets[bucket.Name]
			seenDataCatalogBuckets[bucket.Name] = true
			g.appendR2BucketChildResources(ctx, api, accountID, bucket.Name, jurisdiction, includeDataCatalog)
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
			resource := terraformutils.NewResource(
				database.UUID,
				cloudflareResourceName(accountID, database.Name, database.UUID),
				"cloudflare_d1_database",
				"cloudflare",
				map[string]string{"account_id": accountID, "uuid": database.UUID},
				[]string{},
				map[string]interface{}{},
			)
			setCloudflareImportID(&resource, accountID+"/"+database.UUID)
			g.Resources = append(g.Resources, resource)
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
	return runCloudflareStorageFamilyDiscoveries([]cloudflareStorageFamilyDiscovery{
		{
			name:      "Workers KV namespaces",
			account:   account.Identifier,
			resources: &g.Resources,
			discover: func() error {
				return g.appendWorkersKVNamespaceResources(ctx, api, account.Identifier)
			},
		},
		{
			name:      "queues",
			account:   account.Identifier,
			resources: &g.Resources,
			discover: func() error {
				return g.appendQueueResources(ctx, api, account.Identifier)
			},
		},
		{
			name:      "R2 buckets",
			account:   account.Identifier,
			resources: &g.Resources,
			discover: func() error {
				return g.appendR2BucketResources(ctx, api, account.Identifier)
			},
		},
		{
			name:      "D1 databases",
			account:   account.Identifier,
			resources: &g.Resources,
			discover: func() error {
				return g.appendD1DatabaseResources(ctx, api, account.Identifier)
			},
		},
	})
}
