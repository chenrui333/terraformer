// SPDX-License-Identifier: Apache-2.0

package aws

import "testing"

func TestArnLastSegment(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want string
	}{
		{"ecs cluster arn", "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster", "/", "my-cluster"},
		{"ecs service arn", "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/my-service", "/", "my-service"},
		{"sns topic arn", "arn:aws:sns:us-east-1:123456789012:my-topic", ":", "my-topic"},
		{"sqs queue url", "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue", "/", "my-queue"},
		{"sqs fifo url", "https://sqs.eu-west-1.amazonaws.com/987654321098/orders.fifo", "/", "orders.fifo"},
		{"no separator", "simple-string", "/", "simple-string"},
		{"empty string", "", "/", ""},
		{"trailing separator", "a/b/c/", "/", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := arnLastSegment(tc.s, tc.sep); got != tc.want {
				t.Errorf("arnLastSegment(%q, %q) = %q, want %q", tc.s, tc.sep, got, tc.want)
			}
		})
	}
}
