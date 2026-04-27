// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v5/okta"
)

type AuthenticatorGenerator struct {
	OktaService
}

func (g AuthenticatorGenerator) createResources(authenticators []okta.ListAuthenticators200ResponseInner) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, authenticator := range authenticators {
		instance := authenticator.GetActualInstance()
		if instance == nil {
			continue
		}

		var resourceID, resourceName string
		switch inst := instance.(type) {
		case *okta.AuthenticatorKeyPassword:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		case *okta.AuthenticatorKeyEmail:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		case *okta.AuthenticatorKeyPhone:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		case *okta.AuthenticatorKeyGoogleOtp:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		case *okta.AuthenticatorKeyOktaVerify:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		case *okta.AuthenticatorKeyWebauthn:
			resourceID = inst.GetId()
			resourceName = normalizeResourceNameWithRandom(inst.GetName())
		default:
			continue
		}

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			resourceName,
			"okta_authenticator",
			"okta",
			[]string{},
		))
	}
	return resources
}

func (g *AuthenticatorGenerator) InitResources() error {
	ctx, client, err := g.ClientV5()
	if err != nil {
		return err
	}

	authenticators, _, err := client.AuthenticatorAPI.ListAuthenticators(ctx).Execute()
	if err != nil {
		return err
	}

	g.Resources = g.createResources(authenticators)
	return nil
}
