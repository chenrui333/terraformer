// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func TestGetAccessTokensPagesAndRequestsShowAll(t *testing.T) {
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("offset") {
		case "0":
			writeTokenPage(t, w, 1, pageSize, pageSize)
		case "20":
			writeTokenPage(t, w, 21, 3, 3)
		default:
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
	}))
	defer server.Close()

	config := ldapi.NewConfiguration()
	config.Servers = ldapi.ServerConfigurations{{URL: server.URL}}
	client := ldapi.NewAPIClient(config)

	tokens, err := getAccessTokens(context.Background(), client)
	if err != nil {
		t.Fatalf("getAccessTokens() error = %v", err)
	}
	if len(tokens) != 23 {
		t.Fatalf("getAccessTokens() returned %d tokens, want 23", len(tokens))
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v, want 2 requests", requests)
	}
	for _, query := range requests {
		values, err := url.ParseQuery(query)
		if err != nil {
			t.Fatalf("failed to parse query %q: %v", query, err)
		}
		if values.Get("showAll") != "true" {
			t.Fatalf("query %q missing showAll=true", query)
		}
	}
}

func writeTokenPage(t *testing.T, w http.ResponseWriter, firstID, count, totalCount int) {
	t.Helper()

	items := make([]string, count)
	for i := range items {
		items[i] = fmt.Sprintf(
			"{\"_id\":\"token-%d\",\"ownerId\":\"owner\",\"memberId\":\"member\",\"creationDate\":1,\"lastModified\":1,\"name\":\"duplicate\",\"_links\":{}}",
			firstID+i,
		)
	}

	_, err := fmt.Fprintf(w, "{\"items\":[%s],\"totalCount\":%d}", strings.Join(items, ","), totalCount)
	if err != nil {
		t.Fatalf("failed to write token page: %v", err)
	}
}

func TestGetAccessTokensFallsBackWhenShowAllForbidden(t *testing.T) {
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("showAll") == "true" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("{}"))
			return
		}
		_, _ = w.Write([]byte("{\"items\":[{\"_id\":\"token-1\",\"ownerId\":\"owner\",\"memberId\":\"member\",\"creationDate\":1,\"lastModified\":1,\"_links\":{}}],\"totalCount\":1}"))
	}))
	defer server.Close()

	config := ldapi.NewConfiguration()
	config.Servers = ldapi.ServerConfigurations{{URL: server.URL}}
	client := ldapi.NewAPIClient(config)

	tokens, err := getAccessTokens(context.Background(), client)
	if err != nil {
		t.Fatalf("getAccessTokens() error = %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("getAccessTokens() returned %d tokens, want 1", len(tokens))
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v, want showAll attempt and fallback", requests)
	}

	first, err := url.ParseQuery(requests[0])
	if err != nil {
		t.Fatalf("failed to parse query %q: %v", requests[0], err)
	}
	second, err := url.ParseQuery(requests[1])
	if err != nil {
		t.Fatalf("failed to parse query %q: %v", requests[1], err)
	}
	if first.Get("showAll") != "true" {
		t.Fatalf("first query %q missing showAll=true", requests[0])
	}
	if second.Get("showAll") != "" {
		t.Fatalf("fallback query %q should omit showAll", requests[1])
	}
}

func TestAccessTokenResourceNameIncludesTokenID(t *testing.T) {
	got := accessTokenResourceName("duplicate", "token-123")
	want := "duplicate-token-123"
	if got != want {
		t.Fatalf("accessTokenResourceName() = %q, want %q", got, want)
	}
}
