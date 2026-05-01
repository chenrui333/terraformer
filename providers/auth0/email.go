// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	EmailAllowEmptyValues = []string{}
)

type EmailGenerator struct {
	Auth0Service
}

func (g EmailGenerator) createResources(email *management.EmailProvider) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	resourceName := *email.Name
	resources = append(resources, terraformutils.NewSimpleResource(
		resourceName,
		resourceName,
		"auth0_email",
		"auth0",
		EmailAllowEmptyValues,
	))
	return resources
}

func (g *EmailGenerator) InitResources() error {
	m := g.generateClient()
	ctx := context.Background()
	email, err := m.EmailProvider.Read(ctx)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(email)
	return nil
}
