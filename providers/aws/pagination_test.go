// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestAWSHasMorePages(t *testing.T) {
	tests := []struct {
		name      string
		nextToken *string
		want      bool
	}{
		{name: "nil token"},
		{name: "empty token", nextToken: aws.String("")},
		{name: "non-empty token", nextToken: aws.String("next"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := awsHasMorePages(tt.nextToken); got != tt.want {
				t.Fatalf("awsHasMorePages() = %t, want %t", got, tt.want)
			}
		})
	}
}
