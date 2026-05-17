// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func listSecurityMonitoringRules(auth context.Context, api *datadogV2.SecurityMonitoringApi) ([]datadogV2.SecurityMonitoringRuleResponse, error) {
	rules := []datadogV2.SecurityMonitoringRuleResponse{}
	const pageSize int64 = 1000
	var pageNumber int64

	for {
		optionalParams := datadogV2.NewListSecurityMonitoringRulesOptionalParameters().
			WithPageSize(pageSize).
			WithPageNumber(pageNumber)

		response, httpResp, err := api.ListSecurityMonitoringRules(auth, *optionalParams)
		closeDatadogResponseBody(httpResp)
		if err != nil {
			return nil, err
		}

		pageRules := response.GetData()
		rules = append(rules, pageRules...)
		if !securityMonitoringRulesHasNextPage(response, pageNumber, pageSize, len(pageRules)) {
			break
		}
		pageNumber++
	}

	return rules, nil
}

func securityMonitoringRulesHasNextPage(response datadogV2.SecurityMonitoringListRulesResponse, pageNumber int64, pageSize int64, pageItems int) bool {
	if meta, ok := response.GetMetaOk(); ok {
		page := meta.GetPage()
		if totalCount, ok := page.GetTotalCountOk(); ok {
			return *totalCount > pageSize*(pageNumber+1)
		}
	}
	return pageItems == int(pageSize)
}
