// SPDX-License-Identifier: Apache-2.0

package heroku

import "testing"

func TestHerokuProviderInitClearsStateWhenArgsOmitted(t *testing.T) {
	provider := HerokuProvider{
		apiKey: "old-key",
		team:   "old-team",
	}

	if err := provider.Init(nil); err != nil {
		t.Fatalf("expected Init to succeed: %v", err)
	}
	if provider.apiKey != "" {
		t.Fatalf("apiKey = %q, want empty", provider.apiKey)
	}
	if provider.team != "" {
		t.Fatalf("team = %q, want empty", provider.team)
	}
}
