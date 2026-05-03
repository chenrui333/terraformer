// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type CertificatesGenerator struct {
	CloudflareService
}

func (g *CertificatesGenerator) appendMTLSCertificateResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListMTLSCertificatesParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		certificates, info, err := api.ListMTLSCertificates(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, certificate := range certificates {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				certificate.ID,
				cloudflareResourceName(accountID, certificate.Name, certificate.ID),
				"cloudflare_mtls_certificate",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if !info.HasMorePages() {
			break
		}
		params.Page++
	}
	return nil
}

func (g *CertificatesGenerator) appendCustomHostnameResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	for page := 1; ; page++ {
		customHostnames, info, err := api.CustomHostnames(ctx, zone.ID, page, cf.CustomHostname{})
		if err != nil {
			return err
		}
		for _, customHostname := range customHostnames {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				customHostname.ID,
				cloudflareResourceName(zone.Name, customHostname.Hostname, customHostname.ID),
				"cloudflare_custom_hostname",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if !info.HasMorePages() {
			break
		}
	}
	return nil
}

func (g *CertificatesGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	if g.accountID() != "" {
		if err := g.appendMTLSCertificateResources(ctx, api, g.accountID()); err != nil {
			return err
		}
	}
	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendCustomHostnameResources(ctx, api, zone); err != nil {
			return err
		}
	}
	return nil
}
