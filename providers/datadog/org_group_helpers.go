// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

func listAllOrgGroups(ctx context.Context, api *datadogV2.OrgGroupsApi) ([]datadogV2.OrgGroupData, error) {
	var all []datadogV2.OrgGroupData
	var pageNumber int64
	const pageSize int64 = 100

	for {
		opts := datadogV2.NewListOrgGroupsOptionalParameters().WithPageNumber(pageNumber).WithPageSize(pageSize)
		resp, httpResp, err := api.ListOrgGroups(ctx, *opts)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}

		data := resp.GetData()
		all = append(all, data...)

		if int64(len(data)) < pageSize {
			break
		}
		pageNumber++
	}
	return all, nil
}
