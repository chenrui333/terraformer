// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import "testing"

func TestDestinationResourceNameIncludesDestinationID(t *testing.T) {
	got := destinationResourceName("proj", "prod", "duplicate", "dest-123")
	want := "proj-prod-duplicate-dest-123"
	if got != want {
		t.Fatalf("destinationResourceName() = %q, want %q", got, want)
	}
}
