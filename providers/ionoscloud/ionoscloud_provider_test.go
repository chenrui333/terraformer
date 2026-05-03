// SPDX-License-Identifier: Apache-2.0

package ionoscloud

import (
	"testing"

	sdk "github.com/ionos-cloud/sdk-go/v6"
)

func TestIonosCloudProviderInitClearsStaleTokenWhenSwitchingAuth(t *testing.T) {
	provider := &IonosCloudProvider{}
	t.Setenv(sdk.IonosTokenEnvVar, "token")
	t.Setenv(sdk.IonosUsernameEnvVar, "")
	t.Setenv(sdk.IonosPasswordEnvVar, "")
	t.Setenv(sdk.IonosApiUrlEnvVar, "https://api.ionos.example.com")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected token init to succeed: %v", err)
	}
	if provider.token != "token" || provider.url != "https://api.ionos.example.com" {
		t.Fatalf("expected token state to be initialized, got token=%q url=%q", provider.token, provider.url)
	}

	t.Setenv(sdk.IonosTokenEnvVar, "")
	t.Setenv(sdk.IonosUsernameEnvVar, "user")
	t.Setenv(sdk.IonosPasswordEnvVar, "password")
	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected username/password init to succeed: %v", err)
	}
	if provider.username != "user" || provider.password != "password" || provider.token != "" {
		t.Fatalf("expected token to be cleared after auth switch, got username=%q password=%q token=%q", provider.username, provider.password, provider.token)
	}
}

func TestIonosCloudProviderInitClearsStateOnMissingCredentials(t *testing.T) {
	provider := &IonosCloudProvider{}
	t.Setenv(sdk.IonosTokenEnvVar, "token")
	t.Setenv(sdk.IonosUsernameEnvVar, "")
	t.Setenv(sdk.IonosPasswordEnvVar, "")
	t.Setenv(sdk.IonosApiUrlEnvVar, "")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.token != "token" {
		t.Fatalf("expected token to be initialized, got %q", provider.token)
	}

	t.Setenv(sdk.IonosTokenEnvVar, "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without credentials")
	}
	if provider.username != "" || provider.password != "" || provider.token != "" || provider.url != "" {
		t.Fatalf("expected stale provider state to be cleared, got username=%q password=%q token=%q url=%q", provider.username, provider.password, provider.token, provider.url)
	}
}
