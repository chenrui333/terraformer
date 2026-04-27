//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package octopusdeploy

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/OctopusDeploy/go-octopusdeploy/octopusdeploy"
	"github.com/chenrui333/terraformer/terraformutils"
)

type OctopusDeployService struct { //nolint
	terraformutils.Service
}

func (s *OctopusDeployService) Client() (*octopusdeploy.Client, error) {
	octopusURL := s.Args["address"].(string)
	octopusAPIKey := s.Args["api_key"].(string)

	if octopusURL == "" || octopusAPIKey == "" {
		err := errors.New("Please make sure to set the env variables 'OCTOPUS_CLI_SERVER' and 'OCTOPUS_CLI_API_KEY'")
		return nil, err
	}

	apiURL, err := url.Parse(octopusURL)
	if err != nil {
		return nil, err
	}

	httpClient := http.Client{}
	client, err := octopusdeploy.NewClient(&httpClient, apiURL, octopusAPIKey, "")
	if err != nil {
		return nil, err
	}

	return client, nil
}
