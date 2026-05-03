// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

const datadogListUsersMaxPageSize = int64(100)

type onCallUserChildImportID struct {
	userID  string
	childID string
}

func parseOnCallUserChildImportIDs(importIDs []string, childName string) ([]onCallUserChildImportID, error) {
	filterIDs := []onCallUserChildImportID{}
	for _, importID := range importIDs {
		userID, childID, err := parseOnCallUserChildImportID(importID, childName)
		if err != nil {
			return nil, err
		}
		filterIDs = append(filterIDs, onCallUserChildImportID{userID: userID, childID: childID})
	}
	return filterIDs, nil
}

func parseOnCallUserChildImportID(importID string, childName string) (string, string, error) {
	parts := strings.SplitN(importID, ":", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(importID, ",", 2)
	}
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("On-Call user %s import ID %q must be formatted as user_id:%s_id or user_id,%s_id", childName, importID, childName, childName)
	}
	return parts[0], parts[1], nil
}

func onCallUserChildIDs(filterIDs []onCallUserChildImportID) []string {
	ids := []string{}
	for _, filterID := range filterIDs {
		ids = append(ids, filterID.childID)
	}
	return ids
}

func listDatadogUserIDs(auth context.Context, api *datadogV2.UsersApi) ([]string, error) {
	pageSize := datadogListUsersMaxPageSize
	pageNumber := int64(0)
	remaining := int64(1)
	optionalParams := datadogV2.NewListUsersOptionalParameters()

	userIDs := []string{}
	for remaining > 0 {
		resp, httpResp, err := api.ListUsers(auth, *optionalParams.
			WithPageSize(pageSize).
			WithPageNumber(pageNumber))
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		for _, user := range resp.GetData() {
			if userID := user.GetId(); userID != "" {
				userIDs = append(userIDs, userID)
			}
		}

		remaining = resp.Meta.Page.GetTotalCount() - pageSize*(pageNumber+1)
		pageNumber++
	}

	return userIDs, nil
}
