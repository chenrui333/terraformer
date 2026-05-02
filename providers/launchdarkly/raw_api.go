// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

const (
	launchDarklyAPIBasePath     = "https://app.launchdarkly.com/api/v2"
	launchDarklyDefaultPageSize = 20
)

func getLaunchDarklyAPI(ctx context.Context, apiKey, path string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, launchDarklyAPIURL(path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("LD-API-Version", APIVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("launchdarkly GET %s failed: %s", path, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func launchDarklyAPIURL(path string) string {
	switch {
	case strings.HasPrefix(path, "https://"), strings.HasPrefix(path, "http://"):
		return path
	case strings.HasPrefix(path, "/api/v2"):
		return "https://app.launchdarkly.com" + path
	case strings.HasPrefix(path, "/"):
		return launchDarklyAPIBasePath + path
	default:
		return launchDarklyAPIBasePath + "/" + path
	}
}

func nextPagePath(links map[string]ldapi.Link) string {
	next, ok := links["next"]
	if !ok || next.Href == nil {
		return ""
	}
	return *next.Href
}
