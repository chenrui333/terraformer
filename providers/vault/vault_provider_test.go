package vault

import "testing"

func TestProviderInitClearsStateWhenArgsAndEnvOmitted(t *testing.T) {
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")
	provider := Provider{
		address: "https://old-vault.example.com",
		token:   "old-token",
	}

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.address != "" {
		t.Fatalf("address = %q, want empty", provider.address)
	}
	if provider.token != "" {
		t.Fatalf("token = %q, want empty", provider.token)
	}
}
