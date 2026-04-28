// SPDX-License-Identifier: Apache-2.0

package aws

import "testing"

func TestWildcardUnescape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"escaped wildcard", "\\052.example.com", "*.example.com"},
		{"no wildcard", "www.example.com", "www.example.com"},
		{"empty string", "", ""},
		{"only wildcard", "\\052", "*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := wildcardUnescape(tc.input); got != tc.want {
				t.Errorf("wildcardUnescape(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanZoneID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with prefix", "/hostedzone/Z1234ABC", "Z1234ABC"},
		{"without prefix", "Z1234ABC", "Z1234ABC"},
		{"empty string", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanZoneID(tc.input); got != tc.want {
				t.Errorf("cleanZoneID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCleanPrefix(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		prefix string
		want   string
	}{
		{"matching prefix", "/hostedzone/Z123", "/hostedzone/", "Z123"},
		{"no match", "Z123", "/hostedzone/", "Z123"},
		{"empty id", "", "/hostedzone/", ""},
		{"empty prefix", "/hostedzone/Z123", "", "/hostedzone/Z123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanPrefix(tc.id, tc.prefix); got != tc.want {
				t.Errorf("cleanPrefix(%q, %q) = %q, want %q", tc.id, tc.prefix, got, tc.want)
			}
		})
	}
}
