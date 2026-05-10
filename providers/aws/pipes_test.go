// SPDX-License-Identifier: Apache-2.0

package aws

import "testing"

func TestNewPipesPipeResource(t *testing.T) {
	tests := []struct {
		name     string
		pipeName string
		wantOK   bool
	}{
		{name: "pipe", pipeName: "orders", wantOK: true},
		{name: "empty pipe skipped", pipeName: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := newPipesPipeResource(tt.pipeName)
			if ok != tt.wantOK {
				t.Fatalf("newPipesPipeResource() ok = %t, want %t", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if resource.InstanceState.ID != tt.pipeName {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.pipeName)
			}
			if resource.InstanceInfo.Type != "aws_pipes_pipe" {
				t.Fatalf("resource type = %q, want aws_pipes_pipe", resource.InstanceInfo.Type)
			}
		})
	}
}
