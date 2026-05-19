// SPDX-License-Identifier: Apache-2.0

package importreport

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type ErrorCategory string

const (
	CategoryAuth      ErrorCategory = "auth"
	CategoryAPI       ErrorCategory = "api"
	CategoryPanic     ErrorCategory = "panic"
	CategoryRateLimit ErrorCategory = "rate_limit"
	CategoryEmpty     ErrorCategory = "empty"
	CategoryConvert   ErrorCategory = "convert"
	CategoryUnknown   ErrorCategory = "unknown"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
	StatusEmpty   Status = "empty"
	StatusPanic   Status = "panic"
)

type ResourceEvent struct {
	Service      string        `json:"service"`
	ResourceType string        `json:"resource_type,omitempty"`
	ResourceID   string        `json:"resource_id,omitempty"`
	Status       Status        `json:"status"`
	Error        string        `json:"error,omitempty"`
	Category     ErrorCategory `json:"category,omitempty"`
	DurationMs   int64         `json:"duration_ms,omitempty"`
}

type Summary struct {
	DurationMs      int64           `json:"duration_ms"`
	ServicesSuccess int             `json:"services_success"`
	ServicesFailed  int             `json:"services_failed"`
	ServicesSkipped int             `json:"services_skipped"`
	ResourcesImport int             `json:"resources_imported"`
	ResourcesFailed int             `json:"resources_failed"`
	ResourcesPanic  int             `json:"resources_panic"`
	Failures        []ResourceEvent `json:"failures,omitempty"`
	Skipped         []ResourceEvent `json:"skipped,omitempty"`
}

type Report struct {
	mu         sync.Mutex
	StartTime  time.Time       `json:"start_time"`
	Events     []ResourceEvent `json:"events"`
	authFailed map[string]bool
}

func New() *Report {
	return &Report{
		StartTime:  time.Now(),
		Events:     []ResourceEvent{},
		authFailed: make(map[string]bool),
	}
}

func (r *Report) Add(event ResourceEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Events = append(r.Events, event)
}

func (r *Report) IsAuthFailed(sessionKey string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.authFailed[sessionKey]
}

func (r *Report) SetAuthFailed(sessionKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.authFailed[sessionKey] = true
}

func (r *Report) HasFailures() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.Events {
		if e.Status == StatusFailed || e.Status == StatusPanic {
			return true
		}
	}
	return false
}

func (r *Report) FailedResourceIDs() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make(map[string]bool)
	for _, e := range r.Events {
		if e.ResourceID != "" && (e.Status == StatusFailed || e.Status == StatusPanic) {
			ids[e.ResourceID] = true
		}
	}
	return ids
}

func (r *Report) Summary() Summary {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.summaryLocked()
}

func (r *Report) WriteJSON(w io.Writer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		StartTime time.Time       `json:"start_time"`
		Duration  string          `json:"duration"`
		Events    []ResourceEvent `json:"events"`
		Summary   Summary         `json:"summary"`
	}{
		StartTime: r.StartTime,
		Duration:  time.Since(r.StartTime).Round(time.Millisecond).String(),
		Events:    r.Events,
		Summary:   r.summaryLocked(),
	})
}

func (r *Report) summaryLocked() Summary {
	s := Summary{
		DurationMs: time.Since(r.StartTime).Milliseconds(),
	}
	for _, e := range r.Events {
		switch e.Status {
		case StatusSuccess:
			if e.ResourceID != "" {
				s.ResourcesImport++
			} else {
				s.ServicesSuccess++
			}
		case StatusFailed:
			if e.ResourceID != "" {
				s.ResourcesFailed++
				s.Failures = append(s.Failures, e)
			} else {
				s.ServicesFailed++
				s.Failures = append(s.Failures, e)
			}
		case StatusPanic:
			s.ResourcesPanic++
			s.Failures = append(s.Failures, e)
		case StatusSkipped:
			s.ServicesSkipped++
			s.Skipped = append(s.Skipped, e)
		case StatusEmpty:
			s.ServicesSuccess++
		}
	}
	return s
}

func (r *Report) Print() {
	s := r.Summary()
	w := os.Stderr
	duration := time.Duration(s.DurationMs) * time.Millisecond

	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "═══════════════════════════════════════════\n")
	fmt.Fprintf(w, " Import Summary\n")
	fmt.Fprintf(w, "═══════════════════════════════════════════\n")
	fmt.Fprintf(w, " Duration:    %s\n", duration.Round(time.Second))
	fmt.Fprintf(w, " Services:    %d success, %d failed, %d skipped\n",
		s.ServicesSuccess, s.ServicesFailed, s.ServicesSkipped)
	fmt.Fprintf(w, " Resources:   %d imported, %d failed, %d panic\n",
		s.ResourcesImport, s.ResourcesFailed, s.ResourcesPanic)
	fmt.Fprintf(w, "═══════════════════════════════════════════\n")

	if len(s.Failures) > 0 {
		fmt.Fprintf(w, " Failures:\n")
		for _, f := range s.Failures {
			label := f.Service
			if f.ResourceType != "" {
				label = f.Service + "/" + f.ResourceType
			}
			errMsg := f.Error
			if len(errMsg) > 80 {
				errMsg = errMsg[:80] + "..."
			}
			fmt.Fprintf(w, "   %s: %s (%s)\n", label, strings.ToUpper(string(f.Category)), errMsg)
		}
	}

	if len(s.Skipped) > 0 {
		fmt.Fprintf(w, " Skipped (auth):\n")
		names := make([]string, 0, len(s.Skipped))
		for _, sk := range s.Skipped {
			names = append(names, sk.Service)
		}
		fmt.Fprintf(w, "   %s\n", strings.Join(names, ", "))
	}

	if len(s.Failures) > 0 || len(s.Skipped) > 0 {
		fmt.Fprintf(w, "═══════════════════════════════════════════\n")
	}
}

func (r *Report) WriteJSONFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating report file: %w", err)
	}
	defer f.Close()
	return r.WriteJSON(f)
}
