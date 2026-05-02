// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/pubsub/v1"
)

func TestCreateSubscriptionsResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	pubsubService := newTestPubsubService(ctx, t, server.URL+"/")
	_, err := (PubsubGenerator{}).createSubscriptionsResources(ctx, pubsubService.Projects.Subscriptions.List("projects/test-project"))
	if err == nil {
		t.Fatal("expected pubsub subscription list error")
	}
	if !strings.Contains(err.Error(), "list pubsub subscriptions") {
		t.Fatalf("expected wrapped pubsub subscription list error, got %q", err)
	}
}

func TestCreateTopicsListResourcesReturnsListError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "{\"error\":{\"message\":\"service unavailable\"}}", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	pubsubService := newTestPubsubService(ctx, t, server.URL+"/")
	_, err := (PubsubGenerator{}).createTopicsListResources(ctx, pubsubService.Projects.Topics.List("projects/test-project"))
	if err == nil {
		t.Fatal("expected pubsub topic list error")
	}
	if !strings.Contains(err.Error(), "list pubsub topics") {
		t.Fatalf("expected wrapped pubsub topic list error, got %q", err)
	}
}

func newTestPubsubService(ctx context.Context, t *testing.T, endpoint string) *pubsub.Service {
	t.Helper()

	pubsubService, err := pubsub.NewService(ctx, option.WithEndpoint(endpoint), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	return pubsubService
}
