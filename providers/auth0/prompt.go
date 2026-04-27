// SPDX-License-Identifier: Apache-2.0

package auth0

import (
	"encoding/base64"

	"github.com/chenrui333/terraformer/terraformutils"
	"gopkg.in/auth0.v5/management"
)

var (
	PromptAllowEmptyValues = []string{}
)

type PromptGenerator struct {
	Auth0Service
}

func (g PromptGenerator) createResources(prompt *management.Prompt) []terraformutils.Resource {
	resources := []terraformutils.Resource{}
	resourceName := base64.StdEncoding.EncodeToString([]byte(prompt.String()))
	resources = append(resources, terraformutils.NewSimpleResource(
		resourceName,
		resourceName,
		"auth0_prompt",
		"auth0",
		PromptAllowEmptyValues,
	))
	return resources
}

func (g *PromptGenerator) InitResources() error {
	m := g.generateClient()
	prompt, err := m.Prompt.Read()
	if err != nil {
		return err
	}
	g.Resources = g.createResources(prompt)
	return nil
}
