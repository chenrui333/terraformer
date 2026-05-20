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
	"AADSTS700082",
	"refresh token has expired",
	"AzureCLICredential",
	"please run 'az login'",
	"DefaultAzureCredential",
	"CredentialUnavailableError",
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
	lower := strings.ToLower(msg)
	for _, pattern := range authPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func isRateLimitError(msg string) bool {
	lower := strings.ToLower(msg)
	for _, pattern := range rateLimitPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
