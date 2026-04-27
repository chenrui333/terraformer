//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package myrasec

import (
	"errors"
	"fmt"
	"os"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

// MyrasecService ...
type MyrasecService struct {
	terraformutils.Service
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
	if urlPresent {
		api.BaseURL = apiURL
	}
	api.EnableCaching()
	api.SetCachingTTL(3600)

	return api, err
}
