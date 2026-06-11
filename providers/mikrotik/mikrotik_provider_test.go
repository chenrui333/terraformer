// SPDX-License-Identifier: Apache-2.0

package mikrotik

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ddelnano/terraform-provider-mikrotik/client"
)

func TestMikrotikProviderDataDoesNotExportConnectionSettings(t *testing.T) {
	provider := &MikrotikProvider{
		Mikrotik: client.Mikrotik{
			Host:     "router.example.com:8728",
			Username: "admin",
			Password: "secret-password",
		},
	}

	data := provider.GetProviderData()
	encoded, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal provider data: %v", err)
	}
	for _, sensitiveValue := range []string{"router.example.com", "admin", "secret-password"} {
		if strings.Contains(string(encoded), sensitiveValue) {
			t.Fatalf("provider data exported %q: %s", sensitiveValue, encoded)
		}
	}
	if len(data) != 0 {
		t.Fatalf("provider data = %#v, want empty map", data)
	}
}

func TestMikrotikProviderSupportedServices(t *testing.T) {
	services := (&MikrotikProvider{}).GetSupportedService()
	got := make([]string, 0, len(services))
	for service := range services {
		got = append(got, service)
	}
	sort.Strings(got)

	want := []string{"dhcp_lease"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supported services = %#v, want %#v", got, want)
	}
}

func TestMikrotikProviderInitServiceSeedsDiscoveryArgs(t *testing.T) {
	provider := &MikrotikProvider{
		Mikrotik: client.Mikrotik{
			Host:     "router.example.com:8728",
			Username: "admin",
			Password: "secret-password",
			TLS:      true,
			CA:       "ca-data",
			Insecure: true,
		},
	}

	if err := provider.InitService("dhcp_lease", true); err != nil {
		t.Fatalf("InitService() error = %v", err)
	}
	if provider.Service.GetName() != "dhcp_lease" {
		t.Fatalf("service name = %q, want dhcp_lease", provider.Service.GetName())
	}
	if provider.Service.GetProviderName() != "mikrotik" {
		t.Fatalf("provider name = %q, want mikrotik", provider.Service.GetProviderName())
	}

	args := provider.Service.GetArgs()
	wantArgs := map[string]interface{}{
		"host":           "router.example.com:8728",
		"user":           "admin",
		"password":       "secret-password",
		"tls":            true,
		"ca_certificate": "ca-data",
		"insecure":       true,
	}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("service args = %#v, want %#v", args, wantArgs)
	}
}

func TestMikrotikProviderInitServiceRejectsUnsupportedService(t *testing.T) {
	provider := &MikrotikProvider{}

	err := provider.InitService("missing", false)
	if err == nil {
		t.Fatal("expected unsupported service error")
	}
	if got, want := err.Error(), "mikrotik: missing not supported service"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
	if provider.Service != nil {
		t.Fatalf("service = %#v, want nil", provider.Service)
	}
}
