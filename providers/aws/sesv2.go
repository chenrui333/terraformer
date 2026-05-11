// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
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
			if resource, ok := newSESV2EmailIdentityResource(identity); ok {
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

func newSESV2EmailIdentityResource(identity sesv2types.IdentityInfo) (terraformutils.Resource, bool) {
	identityName := StringValue(identity.IdentityName)
	if identityName == "" {
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
