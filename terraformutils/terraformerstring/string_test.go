// SPDX-License-Identifier: Apache-2.0

package terraformerstring

import "testing"

func TestContainsString(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		elem  string
		want  bool
	}{
		{"found at start", []string{"a", "b", "c"}, "a", true},
		{"found at end", []string{"a", "b", "c"}, "c", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"nil slice", nil, "a", false},
		{"empty string match", []string{"", "b"}, "", true},
		{"case sensitive", []string{"ABC"}, "abc", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ContainsString(tc.slice, tc.elem); got != tc.want {
				t.Errorf("ContainsString(%v, %q) = %v, want %v", tc.slice, tc.elem, got, tc.want)
			}
		})
	}
}
