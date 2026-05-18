// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	docdbtypes "github.com/aws/aws-sdk-go-v2/service/docdb/types"
)

func TestNewDocDBEventSubscriptionResource(t *testing.T) {
	resource, ok := newDocDBEventSubscriptionResource(docdbtypes.EventSubscription{
		CustSubscriptionId: aws.String("events-prod"),
		Status:             aws.String("active"),
	})
	if !ok {
		t.Fatal("newDocDBEventSubscriptionResource() ok = false, want true")
	}
	if got, want := resource.InstanceInfo.Type, "aws_docdb_event_subscription"; got != want {
		t.Fatalf("resource type = %q, want %q", got, want)
	}
	if got, want := resource.InstanceState.ID, "events-prod"; got != want {
		t.Fatalf("resource ID = %q, want %q", got, want)
	}
	if got, want := resource.ResourceName, "tfer--event_subscription_events-prod"; got != want {
		t.Fatalf("resource name = %q, want %q", got, want)
	}
	if !computeDBTestStringSliceContains(resource.IgnoreKeys, "^name_prefix$") {
		t.Fatalf("event subscription IgnoreKeys = %v, want ^name_prefix$", resource.IgnoreKeys)
	}

	if _, ok := newDocDBEventSubscriptionResource(docdbtypes.EventSubscription{
		CustSubscriptionId: aws.String("events-prod"),
		Status:             aws.String("creating"),
	}); ok {
		t.Fatal("creating event subscription should be skipped")
	}
	if _, ok := newDocDBEventSubscriptionResource(docdbtypes.EventSubscription{
		Status: aws.String("active"),
	}); ok {
		t.Fatal("event subscription with empty name should be skipped")
	}
}

func TestDocDBStatusPredicates(t *testing.T) {
	if !docDBEventSubscriptionStatusImportable("ACTIVE") {
		t.Fatal("ACTIVE event subscription should be importable")
	}
	if docDBEventSubscriptionStatusImportable("modifying") {
		t.Fatal("modifying event subscription should be skipped")
	}
}

func TestDocDBLoadOptionalResourcesPropagatesUnexpectedError(t *testing.T) {
	boom := errors.New("boom")
	g := &DocDBGenerator{}
	called := false

	err := g.loadOptionalResources([]docDBOptionalResourceLoader{
		{name: "denied", load: func() error { return boom }},
		{name: "next", load: func() error {
			called = true
			return nil
		}},
	})

	if !errors.Is(err, boom) {
		t.Fatalf("loadOptionalResources() error = %v, want wrapped boom", err)
	}
	if called {
		t.Fatal("loadOptionalResources() should stop after unexpected loader error")
	}
}

func TestDocDBLoadEventSubscriptionsPaginates(t *testing.T) {
	g := &DocDBGenerator{}
	client := &fakeDocDBDescribeEventSubscriptionsClient{
		t: t,
		pages: []*docdb.DescribeEventSubscriptionsOutput{
			{
				EventSubscriptionsList: []docdbtypes.EventSubscription{
					{CustSubscriptionId: aws.String("events-a"), Status: aws.String("active")},
				},
				Marker: aws.String("page-2"),
			},
			{
				EventSubscriptionsList: []docdbtypes.EventSubscription{
					{CustSubscriptionId: aws.String("events-b"), Status: aws.String("active")},
					{CustSubscriptionId: aws.String("events-c"), Status: aws.String("creating")},
				},
			},
		},
	}

	if err := g.getEventSubscriptions(client); err != nil {
		t.Fatalf("getEventSubscriptions() error = %v", err)
	}
	if got, want := client.calls, 2; got != want {
		t.Fatalf("DescribeEventSubscriptions calls = %d, want %d", got, want)
	}
	if got, want := client.markers, []string{"", "page-2"}; !stringSlicesEqual(got, want) {
		t.Fatalf("DescribeEventSubscriptions markers = %#v, want %#v", got, want)
	}
	if got, want := len(g.Resources), 2; got != want {
		t.Fatalf("len(Resources) = %d, want %d", got, want)
	}
}

type fakeDocDBDescribeEventSubscriptionsClient struct {
	t       *testing.T
	pages   []*docdb.DescribeEventSubscriptionsOutput
	calls   int
	markers []string
}

func (c *fakeDocDBDescribeEventSubscriptionsClient) DescribeEventSubscriptions(_ context.Context, input *docdb.DescribeEventSubscriptionsInput, _ ...func(*docdb.Options)) (*docdb.DescribeEventSubscriptionsOutput, error) {
	c.t.Helper()
	if input.Marker == nil {
		c.markers = append(c.markers, "")
	} else {
		c.markers = append(c.markers, *input.Marker)
	}
	if c.calls >= len(c.pages) {
		c.t.Fatalf("unexpected DescribeEventSubscriptions call %d", c.calls+1)
	}
	page := c.pages[c.calls]
	c.calls++
	return page, nil
}
