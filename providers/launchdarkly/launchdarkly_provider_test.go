// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestLaunchDarklyProviderInitClearsStateOnMissingToken(t *testing.T) {
	provider := &LaunchDarklyProvider{}
	t.Setenv("LAUNCHDARKLY_ACCESS_TOKEN", "access-token")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.apiKey != "access-token" || provider.client == nil || provider.ctx == nil {
		t.Fatalf("expected provider state to be initialized, got apiKey=%q client=%v ctx=%v", provider.apiKey, provider.client, provider.ctx)
	}

	t.Setenv("LAUNCHDARKLY_ACCESS_TOKEN", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without LAUNCHDARKLY_ACCESS_TOKEN")
	}
	if provider.apiKey != "" || provider.client != nil || provider.ctx != nil {
		t.Fatalf("expected stale provider state to be cleared, got apiKey=%q client=%v ctx=%v", provider.apiKey, provider.client, provider.ctx)
	}
}

func TestLaunchDarklyProviderDataDoesNotExportAccessToken(t *testing.T) {
	provider := &LaunchDarklyProvider{apiKey: "secret-token"}

	data := provider.GetProviderData()
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal provider data: %v", err)
	}
	if strings.Contains(string(encoded), "secret-token") {
		t.Fatalf("provider data exported access token: %s", encoded)
	}
	if len(data) != 0 {
		t.Fatalf("provider data = %#v, want empty map", data)
	}
}

func TestLaunchDarklyProviderSupportedServices(t *testing.T) {
	services := (&LaunchDarklyProvider{}).GetSupportedService()
	got := make([]string, 0, len(services))
	for service := range services {
		got = append(got, service)
	}
	sort.Strings(got)

	want := []string{
		"accessToken",
		"aiConfig",
		"aiConfigVariation",
		"aiTool",
		"auditLogSubscription",
		"customRole",
		"destination",
		"environment",
		"featureFlag",
		"flagTemplates",
		"flagTrigger",
		"metric",
		"modelConfig",
		"project",
		"relayProxyConfiguration",
		"segment",
		"team",
		"teamMember",
		"view",
		"viewLinks",
		"webhook",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supported services = %#v, want %#v", got, want)
	}
}

func TestLaunchDarklyProviderInitServiceSeedsDiscoveryArgs(t *testing.T) {
	t.Setenv("LAUNCHDARKLY_ACCESS_TOKEN", "access-token")

	provider := &LaunchDarklyProvider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := provider.InitService("webhook", true); err != nil {
		t.Fatalf("InitService() error = %v", err)
	}

	if provider.Service.GetName() != "webhook" {
		t.Fatalf("service name = %q, want webhook", provider.Service.GetName())
	}
	if provider.Service.GetProviderName() != "launchdarkly" {
		t.Fatalf("provider name = %q, want launchdarkly", provider.Service.GetProviderName())
	}
	args := provider.Service.GetArgs()
	if args["api_key"] != "access-token" {
		t.Fatalf("api_key arg = %q, want access-token", args["api_key"])
	}
	if args["client"] == nil {
		t.Fatal("client arg was not set")
	}
	if args["ctx"] == nil {
		t.Fatal("ctx arg was not set")
	}
}

func TestLaunchDarklyProviderInitServiceRejectsUnsupportedService(t *testing.T) {
	provider := &LaunchDarklyProvider{}

	err := provider.InitService("missing", false)
	if err == nil {
		t.Fatal("expected unsupported service error")
	}
	if got, want := err.Error(), "launchdarkly: missing not supported service"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
	if provider.Service != nil {
		t.Fatalf("service = %#v, want nil", provider.Service)
	}
}
