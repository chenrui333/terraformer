//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// MyrasecService ...
type MyrasecService struct {
	terraformutils.Service
	resourcesMu sync.Mutex
}

func (s *MyrasecService) appendResource(resource terraformutils.Resource) {
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()

	s.Resources = append(s.Resources, resource)
}

// initializeAPI ...
func (s *MyrasecService) initializeAPI() (*mgo.API, error) {
	apiKey := os.Getenv("MYRASEC_API_KEY")
	apiSecret := os.Getenv("MYRASEC_API_SECRET")
	apiURL, urlPresent := os.LookupEnv("MYRASEC_API_BASE_URL")

	if apiKey == "" || apiSecret == "" {
		err := errors.New("missing API credentials")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	api, err := mgo.New(apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	if urlPresent && apiURL != "" {
		api.BaseURL = normalizeMyrasecAPIBaseURL(apiURL)
	}
	api.EnableCaching()
	api.SetCachingTTL(3600)

	return api, err
}

func normalizeMyrasecAPIBaseURL(apiURL string) string {
	if apiURL == "" || strings.Contains(apiURL, "%s") {
		return apiURL
	}
	return strings.TrimRight(apiURL, "/") + "/%s"
}
