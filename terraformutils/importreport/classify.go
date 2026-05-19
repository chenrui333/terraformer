// SPDX-License-Identifier: Apache-2.0

package importreport

import "strings"

var authPatterns = []string{
	"SSO session has expired",
	"No valid credential sources",
	"ExpiredToken",
	"security token included in the request is expired",
	"InvalidGrantException",
	"failed to refresh cached credentials",
	"UnauthorizedAccess",
	"InvalidClientTokenId",
	"AuthFailure",
	"ExpiredTokenException",
	"RequestExpired",
}

var rateLimitPatterns = []string{
	"Rate exceeded",
	"Throttling",
	"TooManyRequests",
	"TooManyRequestsException",
	"RequestLimitExceeded",
	"ThrottledException",
	"ProvisionedThroughputExceededException",
}

func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}
	return ClassifyErrorMessage(err.Error())
}

func ClassifyErrorMessage(msg string) ErrorCategory {
	if isAuthError(msg) {
		return CategoryAuth
	}
	if isRateLimitError(msg) {
		return CategoryRateLimit
	}
	return CategoryAPI
}

func isAuthError(msg string) bool {
	for _, pattern := range authPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

func isRateLimitError(msg string) bool {
	for _, pattern := range rateLimitPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}
