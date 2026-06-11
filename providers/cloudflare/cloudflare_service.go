// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/cloudflare/cloudflare-go/v7/option"
)

const cloudflarePageSize = 50

type CloudflareService struct { //nolint
	terraformutils.Service
}

func (s *CloudflareService) initializeAPI() (*cf.API, error) {
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	apiEmail := os.Getenv("CLOUDFLARE_EMAIL")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")

	if apiToken == "" && (apiEmail == "" || apiKey == "") {
		err := errors.New("either CLOUDFLARE_API_TOKEN or CLOUDFLARE_API_KEY/CLOUDFLARE_EMAIL environment variables must be set")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	if apiToken != "" {
		return cf.NewWithAPIToken(apiToken)
	}

	return cf.New(apiKey, apiEmail)
}

func (s *CloudflareService) cloudflareV7Options() ([]option.RequestOption, error) {
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")
	apiEmail := os.Getenv("CLOUDFLARE_EMAIL")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")

	if apiToken == "" && (apiEmail == "" || apiKey == "") {
		err := errors.New("either CLOUDFLARE_API_TOKEN or CLOUDFLARE_API_KEY/CLOUDFLARE_EMAIL environment variables must be set")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	if apiToken != "" {
		return []option.RequestOption{option.WithAPIToken(apiToken)}, nil
	}

	return []option.RequestOption{option.WithAPIKey(apiKey), option.WithAPIEmail(apiEmail)}, nil
}

func (s *CloudflareService) accountID() string {
	return os.Getenv("CLOUDFLARE_ACCOUNT_ID")
}

func (s *CloudflareService) accountIDRequired() (string, error) {
	accountID := s.accountID()
	if accountID == "" {
		return "", errors.New("set CLOUDFLARE_ACCOUNT_ID env var")
	}
	return accountID, nil
}

func (s *CloudflareService) accountResourceContainer() (*cf.ResourceContainer, error) {
	accountID, err := s.accountIDRequired()
	if err != nil {
		return nil, err
	}
	return cf.AccountIdentifier(accountID), nil
}

func cloudflareResourceName(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, "_")
}

func setCloudflareImportID(resource *terraformutils.Resource, importID string) {
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta["import_id"] = importID
}

func cloudflarePaginationQuery(page int, cursor string) string {
	values := url.Values{}
	values.Set("per_page", strconv.Itoa(cloudflarePageSize))
	if cursor != "" {
		values.Set("cursor", cursor)
	} else {
		values.Set("page", strconv.Itoa(page))
	}
	return values.Encode()
}

func cloudflareAdvancePagination(info *cf.ResultInfo, page *int, cursor *string) bool {
	if info == nil {
		return false
	}
	if info.Cursors.After != "" {
		if info.Cursors.After == *cursor {
			return false
		}
		*cursor = info.Cursors.After
		return true
	}
	if info.Cursor != "" {
		if info.Cursor == *cursor {
			return false
		}
		*cursor = info.Cursor
		return true
	}
	if info.HasMorePages() {
		*page++
		return true
	}
	return false
}

func cloudflareAdvancePaginationWithItemCount(info *cf.ResultInfo, page *int, cursor *string, itemCount int) bool {
	if cloudflareAdvancePagination(info, page, cursor) {
		return true
	}
	if info == nil || *cursor != "" {
		return false
	}

	pageSize := cloudflarePageSize
	if info.PerPage > 0 {
		pageSize = info.PerPage
	}
	if itemCount < pageSize {
		return false
	}
	if info.Page > 0 {
		nextPage := info.Page + 1
		if nextPage <= *page {
			return false
		}
		*page = nextPage
		return true
	}
	*page++
	return true
}

func cloudflareZones(ctx context.Context, api *cf.API) ([]cf.Zone, error) {
	return api.ListZones(ctx)
}
