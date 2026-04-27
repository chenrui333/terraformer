// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"encoding/base64"

	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
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
	m := g.generateClient()
	Tenant, err := m.Tenant.Read()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(Tenant)
	return nil
}
