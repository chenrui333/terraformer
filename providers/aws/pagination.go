// SPDX-License-Identifier: Apache-2.0

package aws

import "github.com/aws/aws-sdk-go-v2/aws"

func awsHasMorePages(nextToken *string) bool {
	return aws.ToString(nextToken) != ""
}
