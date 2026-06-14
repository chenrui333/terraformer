// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/v2/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	EmailAllowEmptyValues = []string{}
)

type EmailGenerator struct {
	Auth0Service
}

func (g EmailGenerator) createResources(email *management.GetEmailProviderResponseContent) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if email == nil {
		return nil, auth0MissingResource("auth0_email_provider")
	}
	resourceName, err := auth0RequiredString("auth0_email_provider", "name", email.Name)
	if err != nil {
		return nil, err
	}
	resources = append(resources, terraformutils.NewSimpleResource(
		resourceName,
		resourceName,
		"auth0_email_provider",
		"auth0",
		EmailAllowEmptyValues,
	))
	return resources, nil
}

func (g *EmailGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	email, err := m.Emails.Provider.Get(ctx, &management.GetEmailProviderRequestParameters{})
	if err != nil {
		return err
	}
	resources, err := g.createResources(email)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}
