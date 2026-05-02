// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"context"
	"encoding/base64"

	"github.com/auth0/go-auth0/management"
	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	TenantAllowEmptyValues = []string{}
)

type TenantGenerator struct {
	Auth0Service
}

func (g TenantGenerator) createResources(tenant *management.Tenant) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	resourceName := base64.StdEncoding.EncodeToString([]byte(tenant.String()))
	resources = append(resources, terraformutils.NewSimpleResource(
		resourceName,
		resourceName,
		"auth0_tenant",
		"auth0",
		TenantAllowEmptyValues,
	))
	return resources
}

func (g *TenantGenerator) InitResources() error {
	m, err := g.generateClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	Tenant, err := m.Tenant.Read(ctx)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(Tenant)
	return nil
}
