// SPDX-License-Identifier: Apache-2.0

package importreport

import (
	"errors"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorCategory
	}{
		{"nil error", nil, CategoryUnknown},
		{"SSO expired", errors.New("failed to refresh cached credentials, the SSO session has expired or is invalid"), CategoryAuth},
		{"no valid creds", errors.New("No valid credential sources found"), CategoryAuth},
		{"expired token", errors.New("ExpiredToken: token has expired"), CategoryAuth},
		{"security token expired", errors.New("the security token included in the request is expired"), CategoryAuth},
		{"rate exceeded", errors.New("Rate exceeded for API call"), CategoryRateLimit},
		{"throttling", errors.New("Throttling: rate limit reached"), CategoryRateLimit},
		{"too many requests", errors.New("TooManyRequestsException: slow down"), CategoryRateLimit},
		{"generic API error", errors.New("DescribeInstances: connection timeout"), CategoryAPI},
		{"unknown error", errors.New("something went wrong"), CategoryAPI},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyError(tc.err)
			if got != tc.want {
				t.Errorf("ClassifyError(%q) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestClassifyErrorMessage(t *testing.T) {
	tests := []struct {
		msg  string
		want ErrorCategory
	}{
		{"InvalidGrantException: device code expired", CategoryAuth},
		{"RequestLimitExceeded: please slow down", CategoryRateLimit},
		{"ProvisionedThroughputExceededException", CategoryRateLimit},
		{"ResourceNotFoundException: table not found", CategoryAPI},
	}

	for _, tc := range tests {
		t.Run(tc.msg[:20], func(t *testing.T) {
			got := ClassifyErrorMessage(tc.msg)
			if got != tc.want {
				t.Errorf("ClassifyErrorMessage(%q) = %q, want %q", tc.msg, got, tc.want)
			}
		})
	}
}
