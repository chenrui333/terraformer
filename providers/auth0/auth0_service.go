// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"errors"
	"fmt"

	managementclient "github.com/auth0/go-auth0/v2/management/client"
	managementcore "github.com/auth0/go-auth0/v2/management/core"
	managementoption "github.com/auth0/go-auth0/v2/management/option"
	"github.com/chenrui333/terraformer/terraformutils"
)

type Auth0Service struct { //nolint
	terraformutils.Service
}

const managementClientArg = "management_client"

func newManagementClient(domain, clientID, clientSecret string) (*managementclient.Management, error) {
	authenticationOption := managementoption.WithClientCredentials(context.Background(), clientID, clientSecret)

	apiClient, err := managementclient.New(domain,
		authenticationOption,
		managementoption.WithDebug(false),
	)
	if err != nil {
		return nil, fmt.Errorf("create Auth0 management client: %w", err)
	}

	return apiClient, nil
}

func (s *Auth0Service) generateClient() (*managementclient.Management, error) {
	if apiClient, ok := s.Args[managementClientArg].(*managementclient.Management); ok && apiClient != nil {
		return apiClient, nil
	}

	domain, err := auth0ServiceArgString(s.Args, "domain")
	if err != nil {
		return nil, err
	}
	clientID, err := auth0ServiceArgString(s.Args, "client_id")
	if err != nil {
		return nil, err
	}
	clientSecret, err := auth0ServiceArgString(s.Args, "client_secret")
	if err != nil {
		return nil, err
	}

	return newManagementClient(domain, clientID, clientSecret)
}

func auth0ServiceArgString(args map[string]interface{}, key string) (string, error) {
	value, ok := args[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("auth0: %s arg is missing, empty, or not a string", key)
	}
	return value, nil
}

func auth0PageResults[C comparable, T any, R any](ctx context.Context, page *managementcore.Page[C, T, R]) ([]T, error) {
	var results []T
	for page != nil {
		results = append(results, page.Results...)
		nextPage, err := page.GetNextPage(ctx)
		if errors.Is(err, managementcore.ErrNoPages) {
			return results, nil
		}
		if err != nil {
			return nil, err
		}
		page = nextPage
	}
	return results, nil
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

func auth0RequiredValueString(resourceType, field, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s resource is missing %s", resourceType, field)
	}
	return value, nil
}

func auth0OptionalStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func auth0ResourceName(name *string, fallback string) string {
	if name != nil && *name != "" && *name != fallback {
		return fallback + "_" + *name
	}
	return fallback
}
