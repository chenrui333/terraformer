// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"fmt"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

type Auth0Service struct { //nolint
	terraformutils.Service
}

const managementClientArg = "management_client"

func newManagementClient(domain, clientID, clientSecret string) (*management.Management, error) {
	authenticationOption := management.WithClientCredentials(context.Background(), clientID, clientSecret)

	apiClient, err := management.New(domain,
		authenticationOption,
		management.WithDebug(false),
	)
	if err != nil {
		return nil, fmt.Errorf("create Auth0 management client: %w", err)
	}

	return apiClient, nil
}

func (s *Auth0Service) generateClient() (*management.Management, error) {
	if apiClient, ok := s.Args[managementClientArg].(*management.Management); ok && apiClient != nil {
		return apiClient, nil
	}

	return newManagementClient(s.Args["domain"].(string), s.Args["client_id"].(string), s.Args["client_secret"].(string))
}

func auth0MissingResource(resourceType string) error {
	return fmt.Errorf("%s resource is nil", resourceType)
}

func auth0RequiredString(resourceType, field string, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", fmt.Errorf("%s resource is missing %s", resourceType, field)
	}
	return *value, nil
}

func auth0ResourceName(name *string, fallback string) string {
	if name != nil && *name != "" && *name != fallback {
		return fallback + "_" + *name
	}
	return fallback
}
