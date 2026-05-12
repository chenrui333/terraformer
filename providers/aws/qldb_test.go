// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewQLDBLedgerResource(t *testing.T) {
	resource, ok := newQLDBLedgerResource("ledger")
	assertQLDBResource(t, resource, ok, "ledger", "ledger", qldbLedgerResourceType)

	if _, ok := newQLDBLedgerResource(""); ok {
		t.Fatal("ledger with empty name should be skipped")
	}
}

func TestQLDBImportIDs(t *testing.T) {
	if got, want := qldbLedgerImportID("ledger"), "ledger"; got != want {
		t.Fatalf("QLDB ledger import ID = %q, want %q", got, want)
	}
}

func assertQLDBResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}
