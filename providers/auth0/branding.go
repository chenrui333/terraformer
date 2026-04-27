// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"encoding/base64"

	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	BrandingAllowEmptyValues = []string{}
)

type BrandingGenerator struct {
	Auth0Service
}

func (g BrandingGenerator) createResources(branding *management.Branding) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	resourceName := base64.StdEncoding.EncodeToString([]byte(branding.String()))
	resources = append(resources, terraformutils.NewSimpleResource(
		resourceName,
		resourceName,
		"auth0_branding",
		"auth0",
		BrandingAllowEmptyValues,
	))
	return resources
}

func (g *BrandingGenerator) InitResources() error {
	m := g.generateClient()
	branding, err := m.Branding.Read()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(branding)
	return nil
}
