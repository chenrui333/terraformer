// SPDX-License-Identifier: Apache-2.0

package unsupportedresources

import (
	"slices"
	"testing"
)

func TestStatuses(t *testing.T) {
	statuses := Statuses()
	if !slices.IsSorted(statuses) {
		t.Fatalf("Statuses() = %v, want sorted statuses", statuses)
	}
	for index, status := range statuses {
		if !IsValidStatus(status) {
			t.Fatalf("IsValidStatus(%q) = false, want true", status)
		}
		if index > 0 && statuses[index-1] == status {
			t.Fatalf("Statuses() contains duplicate status %q", status)
		}
	}

	statuses[0] = "mutated"
	if IsValidStatus("mutated") || Statuses()[0] == "mutated" {
		t.Fatal("Statuses() exposed mutable canonical status storage")
	}
}

func TestIsValidStatusRejectsNonCanonicalValues(t *testing.T) {
	for _, status := range []string{"", "future-status", "needs-research", "unsafe-discovery", " action-style "} {
		if IsValidStatus(status) {
			t.Fatalf("IsValidStatus(%q) = true, want false", status)
		}
	}
}
