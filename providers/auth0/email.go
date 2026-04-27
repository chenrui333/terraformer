// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	EmailAllowEmptyValues = []string{}
)

type EmailGenerator struct {
	Auth0Service
}

func (g EmailGenerator) createResources(email *management.Email) []terraformutils.Resource {
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
	Email, err := m.Email.Read()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(Email)
	return nil
}
