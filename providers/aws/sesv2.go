// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	sesv2ConfigurationSetResourceType = "aws_sesv2_configuration_set"
	sesv2DedicatedIPPoolResourceType  = "aws_sesv2_dedicated_ip_pool"
	sesv2EmailIdentityResourceType    = "aws_sesv2_email_identity"
)

var sesv2AllowEmptyValues = []string{"tags."}

type SesV2Generator struct {
	AWSService
}

func (g *SesV2Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := sesv2.NewFromConfig(config)

	if err := g.loadConfigurationSets(svc); err != nil {
		return err
	}
	if err := g.loadDedicatedIPPools(svc); err != nil {
		return err
	}
	if err := g.loadEmailIdentities(svc); err != nil {
		return err
	}

	return nil
}

func (g *SesV2Generator) loadConfigurationSets(svc *sesv2.Client) error {
	p := sesv2.NewListConfigurationSetsPaginator(svc, &sesv2.ListConfigurationSetsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configurationSetName := range page.ConfigurationSets {
			if resource, ok := newSESV2ConfigurationSetResource(configurationSetName); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SesV2Generator) loadDedicatedIPPools(svc *sesv2.Client) error {
	p := sesv2.NewListDedicatedIpPoolsPaginator(svc, &sesv2.ListDedicatedIpPoolsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, poolName := range page.DedicatedIpPools {
			if resource, ok := newSESV2DedicatedIPPoolResource(poolName); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SesV2Generator) loadEmailIdentities(svc *sesv2.Client) error {
	p := sesv2.NewListEmailIdentitiesPaginator(svc, &sesv2.ListEmailIdentitiesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, identity := range page.EmailIdentities {
			identityName := StringValue(identity.IdentityName)
			if identityName == "" {
				continue
			}
			output, err := svc.GetEmailIdentity(context.TODO(), &sesv2.GetEmailIdentityInput{
				EmailIdentity: &identityName,
			})
			if err != nil {
				if sesv2NotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSESV2EmailIdentityResource(identityName, output); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newSESV2ConfigurationSetResource(configurationSetName string) (terraformutils.Resource, bool) {
	if configurationSetName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2ConfigurationSetImportID(configurationSetName),
		sesv2ResourceName("configuration_set", configurationSetName),
		sesv2ConfigurationSetResourceType,
		"aws",
		map[string]string{
			"configuration_set_name": configurationSetName,
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSESV2DedicatedIPPoolResource(poolName string) (terraformutils.Resource, bool) {
	if poolName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2DedicatedIPPoolImportID(poolName),
		sesv2ResourceName("dedicated_ip_pool", poolName),
		sesv2DedicatedIPPoolResourceType,
		"aws",
		map[string]string{
			"pool_name": poolName,
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSESV2EmailIdentityResource(identityName string, output *sesv2.GetEmailIdentityOutput) (terraformutils.Resource, bool) {
	if identityName == "" || !sesv2EmailIdentityImportable(output) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2EmailIdentityImportID(identityName),
		sesv2ResourceName("email_identity", identityName),
		sesv2EmailIdentityResourceType,
		"aws",
		map[string]string{
			"email_identity": identityName,
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func sesv2EmailIdentityImportable(output *sesv2.GetEmailIdentityOutput) bool {
	if output == nil || output.DkimAttributes == nil {
		return true
	}
	// BYODKIM private keys are sensitive and are not returned by SESv2, so importing
	// external-signing identities would generate Terraform that can reset DKIM mode.
	return output.DkimAttributes.SigningAttributesOrigin != sesv2types.DkimSigningAttributesOriginExternal
}

func sesv2NotFound(err error) bool {
	var notFound *sesv2types.NotFoundException
	return errors.As(err, &notFound)
}

func sesv2ConfigurationSetImportID(configurationSetName string) string {
	return configurationSetName
}

func sesv2DedicatedIPPoolImportID(poolName string) string {
	return poolName
}

func sesv2EmailIdentityImportID(identityName string) string {
	return identityName
}

func sesv2ResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}
