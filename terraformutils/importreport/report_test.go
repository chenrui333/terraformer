// SPDX-License-Identifier: Apache-2.0

package importreport

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

func TestReportAddAndSummary(t *testing.T) {
	r := New()
	r.Add(ResourceEvent{Service: "vpc", Status: StatusSuccess})
	r.Add(ResourceEvent{Service: "iam", Status: StatusSuccess})
	r.Add(ResourceEvent{Service: "backup", Status: StatusFailed, Category: CategoryAPI, Error: "timeout"})
	r.Add(ResourceEvent{Service: "ses", Status: StatusSkipped, Category: CategoryAuth})

	s := r.Summary()

	if s.ServicesSuccess != 2 {
		t.Errorf("ServicesSuccess = %d, want 2", s.ServicesSuccess)
	}
	if s.ServicesFailed != 1 {
		t.Errorf("ServicesFailed = %d, want 1", s.ServicesFailed)
	}
	if s.ServicesSkipped != 1 {
		t.Errorf("ServicesSkipped = %d, want 1", s.ServicesSkipped)
	}
	if len(s.Failures) != 1 {
		t.Errorf("Failures count = %d, want 1", len(s.Failures))
	}
	if len(s.Skipped) != 1 {
		t.Errorf("Skipped count = %d, want 1", len(s.Skipped))
	}
}

func TestReportResourceEvents(t *testing.T) {
	r := New()
	r.Add(ResourceEvent{Service: "backup", ResourceID: "vault-1", Status: StatusSuccess})
	r.Add(ResourceEvent{Service: "backup", ResourceID: "framework-1", Status: StatusPanic, Category: CategoryPanic, Error: "cty panic"})
	r.Add(ResourceEvent{Service: "backup", ResourceID: "plan-1", Status: StatusFailed, Category: CategoryAPI, Error: "api err"})

	s := r.Summary()

	if s.ResourcesImport != 1 {
		t.Errorf("ResourcesImport = %d, want 1", s.ResourcesImport)
	}
	if s.ResourcesPanic != 1 {
		t.Errorf("ResourcesPanic = %d, want 1", s.ResourcesPanic)
	}
	if s.ResourcesFailed != 1 {
		t.Errorf("ResourcesFailed = %d, want 1", s.ResourcesFailed)
	}
}

func TestAuthFailedTracking(t *testing.T) {
	r := New()

	if r.IsAuthFailed("aws-us-east-1") {
		t.Error("should not be auth failed initially")
	}

	r.SetAuthFailed("aws-us-east-1")

	if !r.IsAuthFailed("aws-us-east-1") {
		t.Error("should be auth failed after SetAuthFailed")
	}
	if r.IsAuthFailed("aws-eu-west-1") {
		t.Error("different session should not be auth failed")
	}
}

func TestHasFailures(t *testing.T) {
	r := New()
	r.Add(ResourceEvent{Service: "vpc", Status: StatusSuccess})
	if r.HasFailures() {
		t.Error("should not have failures with only successes")
	}

	r.Add(ResourceEvent{Service: "iam", Status: StatusFailed, Error: "err"})
	if !r.HasFailures() {
		t.Error("should have failures after adding failed event")
	}
}

func TestConcurrentAdd(t *testing.T) {
	r := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.Add(ResourceEvent{Service: "svc", ResourceID: "r", Status: StatusSuccess})
		}(i)
	}
	wg.Wait()

	if len(r.Events) != 100 {
		t.Errorf("Events count = %d, want 100", len(r.Events))
	}
}

func TestWriteJSON(t *testing.T) {
	r := New()
	r.Add(ResourceEvent{Service: "vpc", Status: StatusSuccess})
	r.Add(ResourceEvent{Service: "iam", Status: StatusFailed, Category: CategoryAuth, Error: "expired"})

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if _, ok := result["events"]; !ok {
		t.Error("JSON output missing 'events' field")
	}
	if _, ok := result["summary"]; !ok {
		t.Error("JSON output missing 'summary' field")
	}
}
