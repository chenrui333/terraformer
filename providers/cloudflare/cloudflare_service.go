// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
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
		err := errors.New("Either CLOUDFLARE_API_TOKEN or CLOUDFLARE_API_KEY/CLOUDFLARE_EMAIL environment variables must be set")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	if apiToken != "" {
		return cf.NewWithAPIToken(apiToken)
	}

	return cf.New(apiKey, apiEmail)
}

func (s *CloudflareService) accountID() string {
	return os.Getenv("CLOUDFLARE_ACCOUNT_ID")
}

func (s *CloudflareService) accountResourceContainer() (*cf.ResourceContainer, error) {
	accountID := s.accountID()
	if accountID == "" {
		return nil, errors.New("set CLOUDFLARE_ACCOUNT_ID env var")
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

func cloudflareZones(ctx context.Context, api *cf.API) ([]cf.Zone, error) {
	return api.ListZones(ctx)
}
