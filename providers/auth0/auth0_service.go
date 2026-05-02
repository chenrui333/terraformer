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

func (s *Auth0Service) generateClient() (*management.Management, error) {
	authenticationOption := management.WithClientCredentials(context.Background(), s.Args["client_id"].(string), s.Args["client_secret"].(string))

	apiClient, err := management.New(s.Args["domain"].(string),
		authenticationOption,
		management.WithDebug(false),
	)
	if err != nil {
		return nil, fmt.Errorf("create Auth0 management client: %w", err)
	}

	return apiClient, nil
}
