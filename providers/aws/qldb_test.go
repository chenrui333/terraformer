// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	qldbtypes "github.com/aws/aws-sdk-go-v2/service/qldb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestNewQLDBLedgerResource(t *testing.T) {
	resource, ok := newQLDBLedgerResource("ledger")
	assertQLDBResource(t, resource, ok, "ledger", "ledger", qldbLedgerResourceType)

	if _, ok := newQLDBLedgerResource(""); ok {
		t.Fatal("ledger with empty name should be skipped")
	}
}

func TestNewQLDBStreamResource(t *testing.T) {
	resource, ok := newQLDBStreamResource("ledger", qldbtypes.JournalKinesisStreamDescription{
		StreamId:   aws.String("stream-id"),
		StreamName: aws.String("journal-stream"),
		Status:     qldbtypes.StreamStatusActive,
	})
	assertQLDBResource(t, resource, ok, "stream-id", qldbResourceName("stream", "ledger", "journal-stream", "stream-id"), qldbStreamResourceType)
	assertQLDBAttribute(t, resource, "ledger_name", "ledger")
	assertQLDBAttribute(t, resource, "stream_name", "journal-stream")
}

func TestQLDBStreamSkipsEmptyIdentifiers(t *testing.T) {
	if _, ok := newQLDBStreamResource("", qldbtypes.JournalKinesisStreamDescription{
		StreamId:   aws.String("stream-id"),
		StreamName: aws.String("journal-stream"),
		Status:     qldbtypes.StreamStatusActive,
	}); ok {
		t.Fatal("stream with empty ledger name should be skipped")
	}

	if _, ok := newQLDBStreamResource("ledger", qldbtypes.JournalKinesisStreamDescription{
		StreamName: aws.String("journal-stream"),
		Status:     qldbtypes.StreamStatusActive,
	}); ok {
		t.Fatal("stream with empty ID should be skipped")
	}
}

func TestQLDBStreamImportable(t *testing.T) {
	tests := []struct {
		name   string
		status qldbtypes.StreamStatus
		want   bool
	}{
		{name: "active", status: qldbtypes.StreamStatusActive, want: true},
		{name: "impaired", status: qldbtypes.StreamStatusImpaired, want: true},
		{name: "empty", want: true},
		{name: "completed", status: qldbtypes.StreamStatusCompleted, want: false},
		{name: "canceled", status: qldbtypes.StreamStatusCanceled, want: false},
		{name: "failed", status: qldbtypes.StreamStatusFailed, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := qldbStreamImportable(tt.status); got != tt.want {
				t.Fatalf("qldbStreamImportable(%q) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

func TestQLDBImportIDs(t *testing.T) {
	if got, want := qldbLedgerImportID("ledger"), "ledger"; got != want {
		t.Fatalf("QLDB ledger import ID = %q, want %q", got, want)
	}
	if got, want := qldbStreamImportID("stream-id"), "stream-id"; got != want {
		t.Fatalf("QLDB stream import ID = %q, want %q", got, want)
	}
}

func TestQLDBResourceNamesPreserveSegmentBoundaries(t *testing.T) {
	left := terraformutils.TfSanitize(qldbResourceName("stream", "ledger/a_b", "stream-id"))
	right := terraformutils.TfSanitize(qldbResourceName("stream", "ledger", "a_b/stream-id"))
	if left == right {
		t.Fatalf("QLDB resource names collide: %q", left)
	}
}

func TestQLDBLedgerIDFilterAllowsAllWhenStreamIDsAreRequested(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"qldb_ledger=ledger-a",
		"qldb_stream=stream-b",
	})

	filter := qldbLedgerIDFilter(service.Filter)
	if !awsIDFilterAllows(filter, "ledger-b") {
		t.Fatalf("QLDB stream ID filter should disable ledger prefilter: %#v", filter)
	}
}

func TestQLDBLedgerIDFilterRestrictsLedgerOnlyFilters(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{"qldb_ledger=ledger-a"})

	filter := qldbLedgerIDFilter(service.Filter)
	if !awsIDFilterAllows(filter, "ledger-a") {
		t.Fatalf("QLDB ledger filter should allow ledger-a: %#v", filter)
	}
	if awsIDFilterAllows(filter, "ledger-b") {
		t.Fatalf("QLDB ledger filter allowed unrelated ledger: %#v", filter)
	}
}

func TestQLDBStreamNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &qldbtypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &qldbtypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := qldbStreamNotFound(tt.err); got != tt.want {
				t.Fatalf("qldbStreamNotFound(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
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

func assertQLDBAttribute(t *testing.T, resource terraformutils.Resource, key, want string) {
	t.Helper()
	if got := resource.InstanceState.Attributes[key]; got != want {
		t.Fatalf("resource attribute %q = %q, want %q", key, got, want)
	}
}
