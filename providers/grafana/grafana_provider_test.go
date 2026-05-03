// SPDX-License-Identifier: Apache-2.0

package grafana

import "testing"

func TestGrafanaProviderInitClearsStateOnMissingURL(t *testing.T) {
	provider := &GrafanaProvider{}
	t.Setenv("GRAFANA_AUTH", "auth")
	t.Setenv("GRAFANA_URL", "https://grafana.example.com")
	t.Setenv("GRAFANA_ORG_ID", "42")
	t.Setenv("HTTPS_TLS_KEY", "tls-key")
	t.Setenv("HTTPS_TLS_CERT", "tls-cert")
	t.Setenv("HTTPS_CA_CERT", "ca-cert")
	t.Setenv("HTTPS_INSECURE_SKIP_VERIFY", "1")

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if provider.auth != "auth" || provider.url != "https://grafana.example.com" || provider.orgID != 42 || !provider.insecureSkipVerify {
		t.Fatalf("expected provider state to be initialized, got auth=%q url=%q orgID=%d insecure=%t", provider.auth, provider.url, provider.orgID, provider.insecureSkipVerify)
	}

	t.Setenv("GRAFANA_URL", "")
	if err := provider.Init(nil); err == nil {
		t.Fatal("expected init to fail without GRAFANA_URL")
	}
	if provider.auth != "" || provider.url != "" || provider.orgID != 0 || provider.tlsKey != "" || provider.tlsCert != "" || provider.caCert != "" || provider.insecureSkipVerify {
		t.Fatalf("expected stale provider state to be cleared, got auth=%q url=%q orgID=%d tlsKey=%q tlsCert=%q caCert=%q insecure=%t", provider.auth, provider.url, provider.orgID, provider.tlsKey, provider.tlsCert, provider.caCert, provider.insecureSkipVerify)
	}
}
