// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cloudflare

import (
	"errors"
	"fmt"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

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
