// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"encoding/base64"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
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
	ctx := context.Background()
	branding, err := m.Branding.Read(ctx)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(branding)
	return nil
}
