// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"errors"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	cf "github.com/cloudflare/cloudflare-go"
)

func TestRunCloudflareStorageChildDiscoveriesContinuesAfterError(t *testing.T) {
	calls := []string{}
	runCloudflareStorageChildDiscoveries([]cloudflareStorageChildDiscovery{
		{
			name:   "fails",
			parent: "parent",
			discover: func() error {
				calls = append(calls, "fails")
				return errors.New("permission denied")
			},
		},
		{
			name:   "succeeds",
			parent: "parent",
			discover: func() error {
				calls = append(calls, "succeeds")
				return nil
			},
		},
		{name: "nil discoverer", parent: "parent"},
	})

	if len(calls) != 2 || calls[0] != "fails" || calls[1] != "succeeds" {
		t.Fatalf("discoveries called in order = %#v, want [fails succeeds]", calls)
	}
}

func TestRunCloudflareStorageFamilyDiscoveriesContinuesAfterError(t *testing.T) {
	calls := []string{}
	resources := []terraformutils.Resource{}
	err := runCloudflareStorageFamilyDiscoveries([]cloudflareStorageFamilyDiscovery{
		{
			name:      "queues",
			account:   "account-id",
			resources: &resources,
			discover: func() error {
				calls = append(calls, "queues")
				resources = append(resources, terraformutils.NewResource("partial-queue", "partial_queue", "cloudflare_queue", "cloudflare", nil, nil, nil))
				return errors.New("permission denied")
			},
		},
		{
			name:      "R2 buckets",
			account:   "account-id",
			resources: &resources,
			discover: func() error {
				calls = append(calls, "r2")
				resources = append(resources, terraformutils.NewResource("complete-r2", "complete_r2", "cloudflare_r2_bucket", "cloudflare", nil, nil, nil))
				return nil
			},
		},
		{name: "nil discoverer", account: "account-id"},
	})
	if err != nil {
		t.Fatalf("runCloudflareStorageFamilyDiscoveries() error = %v, want nil", err)
	}
	if len(calls) != 2 || calls[0] != "queues" || calls[1] != "r2" {
		t.Fatalf("discoveries called in order = %#v, want [queues r2]", calls)
	}
	if len(resources) != 1 || resources[0].InstanceState.ID != "complete-r2" {
		t.Fatalf("resources after rollback = %#v, want only complete-r2", resources)
	}
}

func TestRunCloudflareStorageFamilyDiscoveriesReturnsErrorWhenAllFail(t *testing.T) {
	wantErr := errors.New("permission denied")
	resources := []terraformutils.Resource{}
	err := runCloudflareStorageFamilyDiscoveries([]cloudflareStorageFamilyDiscovery{
		{
			name:      "queues",
			account:   "account-id",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial-queue", "partial_queue", "cloudflare_queue", "cloudflare", nil, nil, nil))
				return wantErr
			},
		},
		{
			name:      "R2 buckets",
			account:   "account-id",
			resources: &resources,
			discover: func() error {
				resources = append(resources, terraformutils.NewResource("partial-r2", "partial_r2", "cloudflare_r2_bucket", "cloudflare", nil, nil, nil))
				return errors.New("forbidden")
			},
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("runCloudflareStorageFamilyDiscoveries() error = %v, want %v", err, wantErr)
	}
	if len(resources) != 0 {
		t.Fatalf("resources after all failed discoveries = %#v, want none", resources)
	}
}

func TestNewCloudflareQueueConsumerResource(t *testing.T) {
	queue := cf.Queue{ID: "queue-id", Name: "orders"}
	consumer := cloudflareQueueConsumer{
		ConsumerID:      "consumer-id",
		DeadLetterQueue: "orders-dlq",
		ScriptName:      "consumer-worker",
		Type:            "worker",
	}

	resource, ok := newCloudflareQueueConsumerResource("account-id", queue, consumer)
	if !ok {
		t.Fatal("expected queue consumer resource")
	}
	if resource.InstanceInfo.Type != "cloudflare_queue_consumer" {
		t.Fatalf("resource type = %q, want cloudflare_queue_consumer", resource.InstanceInfo.Type)
	}
	assertCloudflarePreservesID(t, resource)
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"account_id":        "account-id",
		"consumer_id":       "consumer-id",
		"dead_letter_queue": "orders-dlq",
		"queue_id":          "queue-id",
		"script_name":       "consumer-worker",
		"type":              "worker",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}

func TestNewCloudflareQueueConsumerResourceSkipsMalformedConsumers(t *testing.T) {
	queue := cf.Queue{ID: "queue-id", Name: "orders"}
	for name, consumer := range map[string]cloudflareQueueConsumer{
		"missing consumer id": {Type: "worker"},
		"missing type":        {ConsumerID: "consumer-id"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := newCloudflareQueueConsumerResource("account-id", queue, consumer); ok {
				t.Fatal("expected malformed consumer to be skipped")
			}
		})
	}
	if _, ok := newCloudflareQueueConsumerResource("account-id", cf.Queue{}, cloudflareQueueConsumer{ConsumerID: "consumer-id", Type: "worker"}); ok {
		t.Fatal("expected consumer without parent queue ID to be skipped")
	}
}

func TestNewCloudflareR2BucketConfigResourcePreservesID(t *testing.T) {
	resource := newCloudflareR2BucketConfigResource("account-id", "bucket-name", "eu", "cloudflare_r2_bucket_cors")
	if resource.InstanceInfo.Type != "cloudflare_r2_bucket_cors" {
		t.Fatalf("resource type = %q, want cloudflare_r2_bucket_cors", resource.InstanceInfo.Type)
	}
	assertCloudflarePreservesID(t, resource)
}

func TestNewCloudflareR2BucketEventNotificationResource(t *testing.T) {
	resource, ok := newCloudflareR2BucketEventNotificationResource(
		"account-id",
		"bucket-name",
		"eu",
		cloudflareR2BucketEventNotificationQueue{
			QueueID:   "queue-id",
			QueueName: "orders",
			Rules: []cloudflareR2BucketEventNotificationRule{
				{Description: "ignore empty actions"},
				{
					Actions:     []string{"PutObject", "DeleteObject"},
					Description: "objects",
					Prefix:      "in/",
					Suffix:      ".json",
				},
			},
		},
	)
	if !ok {
		t.Fatal("expected event notification resource")
	}
	assertCloudflarePreservesID(t, resource)
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"account_id":          "account-id",
		"bucket_name":         "bucket-name",
		"jurisdiction":        "eu",
		"queue_id":            "queue-id",
		"queue_name":          "orders",
		"rules.#":             "1",
		"rules.0.actions.#":   "2",
		"rules.0.actions.0":   "PutObject",
		"rules.0.actions.1":   "DeleteObject",
		"rules.0.description": "objects",
		"rules.0.prefix":      "in/",
		"rules.0.suffix":      ".json",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}

func TestNewCloudflareR2BucketEventNotificationResourceSkipsEmptyRules(t *testing.T) {
	queue := cloudflareR2BucketEventNotificationQueue{
		QueueID: "queue-id",
		Rules:   []cloudflareR2BucketEventNotificationRule{{Description: "no actions"}},
	}
	if _, ok := newCloudflareR2BucketEventNotificationResource("account-id", "bucket-name", "default", queue); ok {
		t.Fatal("expected event notification without actionable rules to be skipped")
	}
	if _, ok := newCloudflareR2BucketEventNotificationResource("account-id", "bucket-name", "default", cloudflareR2BucketEventNotificationQueue{}); ok {
		t.Fatal("expected event notification without queue ID to be skipped")
	}
}

func TestNewCloudflareR2CustomDomainResource(t *testing.T) {
	resource, ok := newCloudflareR2CustomDomainResource(
		"account-id",
		"bucket-name",
		"default",
		cloudflareR2CustomDomain{
			Ciphers:  []string{"TLS_AES_128_GCM_SHA256"},
			Domain:   "assets.example.com",
			Enabled:  true,
			MinTLS:   "1.2",
			ZoneID:   "zone-id",
			ZoneName: "example.com",
		},
	)
	if !ok {
		t.Fatal("expected custom domain resource")
	}
	assertCloudflarePreservesID(t, resource)
	attributes := resource.InstanceState.Attributes
	for key, want := range map[string]string{
		"account_id":   "account-id",
		"bucket_name":  "bucket-name",
		"ciphers.#":    "1",
		"ciphers.0":    "TLS_AES_128_GCM_SHA256",
		"domain":       "assets.example.com",
		"enabled":      "true",
		"jurisdiction": "default",
		"min_tls":      "1.2",
		"zone_id":      "zone-id",
		"zone_name":    "example.com",
	} {
		if got := attributes[key]; got != want {
			t.Fatalf("attribute %s = %q, want %q", key, got, want)
		}
	}
}

func TestNewCloudflareR2CustomDomainResourceRequiresZoneID(t *testing.T) {
	if _, ok := newCloudflareR2CustomDomainResource("account-id", "bucket-name", "default", cloudflareR2CustomDomain{Domain: "assets.example.com"}); ok {
		t.Fatal("expected custom domain without zone ID to be skipped")
	}
}

func TestNewCloudflareR2DataCatalogResource(t *testing.T) {
	resource, ok := newCloudflareR2DataCatalogResource(
		"account-id",
		"input-bucket",
		cloudflareR2DataCatalog{Bucket: "catalog-bucket", ID: "catalog-id", Status: "active"},
	)
	if !ok {
		t.Fatal("expected active data catalog resource")
	}
	if resource.InstanceState.ID != "catalog-id" {
		t.Fatalf("state ID = %q, want catalog-id", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["bucket_name"]; got != "catalog-bucket" {
		t.Fatalf("bucket_name = %q, want catalog-bucket", got)
	}
	if got := resource.InstanceState.Meta["import_id"]; got != "account-id/catalog-bucket" {
		t.Fatalf("import_id = %v, want account-id/catalog-bucket", got)
	}
}

func TestNewCloudflareR2DataCatalogResourceSkipsInactiveCatalog(t *testing.T) {
	if _, ok := newCloudflareR2DataCatalogResource("account-id", "bucket-name", cloudflareR2DataCatalog{Status: "inactive"}); ok {
		t.Fatal("expected inactive data catalog to be skipped")
	}
}

func assertCloudflarePreservesID(t *testing.T, resource terraformutils.Resource) {
	t.Helper()

	preserveID, ok := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh].(bool)
	if !ok || !preserveID {
		t.Fatalf("preserve ID metadata = %#v, want true", resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh])
	}
}
