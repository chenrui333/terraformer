// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"log"

	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

type Auth0Service struct { //nolint
	terraformutils.Service
}

func (s *Auth0Service) generateClient() *management.Management {
	authenticationOption := management.WithClientCredentials(s.Args["client_id"].(string), s.Args["client_secret"].(string))

	apiClient, err := management.New(s.Args["domain"].(string),
		authenticationOption,
		management.WithDebug(false),
	)
	if err != nil {
		log.Fatalf("%v", err)
	}

	return apiClient
}
