// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type AccessGenerator struct {
	CloudflareService
}

type cloudflareAccessShortLivedCertificate struct {
	ID string `json:"id"`
}

func cloudflareAccessShortLivedCertificateOptionalError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return true
	}
	return cloudflareCertificateOptionalDiscoveryError(err)
}

func accessScopeAttributes(scopeType, scopeID string) map[string]string {
	if scopeType == "accounts" {
		return map[string]string{"account_id": scopeID}
	}
	return map[string]string{"zone_id": scopeID}
}

func (g *AccessGenerator) appendAccessApplicationResources(
	ctx context.Context,
	api *cf.API,
	rc *cf.ResourceContainer,
	scopeType string,
) error {
	applications, err := listAccessApplications(ctx, api, rc)
	if err != nil {
		return err
	}
	for _, app := range applications {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			app.ID,
			cloudflareResourceName(scopeType, rc.Identifier, app.Name, app.ID),
			"cloudflare_zero_trust_access_application",
			"cloudflare",
			accessScopeAttributes(scopeType, rc.Identifier),
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func listAccessApplications(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.AccessApplication, error) {
	var allApplications []cf.AccessApplication
	params := cf.ListAccessApplicationsParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		applications, info, err := api.ListAccessApplications(ctx, rc, params)
		if err != nil {
			return nil, err
		}
		allApplications = append(allApplications, applications...)
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return allApplications, nil
}

func (g *AccessGenerator) appendAccessGroupResources(
	ctx context.Context,
	api *cf.API,
	rc *cf.ResourceContainer,
	scopeType string,
) error {
	params := cf.ListAccessGroupsParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		groups, info, err := api.ListAccessGroups(ctx, rc, params)
		if err != nil {
			return err
		}
		for _, group := range groups {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				group.ID,
				cloudflareResourceName(scopeType, rc.Identifier, group.Name, group.ID),
				"cloudflare_zero_trust_access_group",
				"cloudflare",
				accessScopeAttributes(scopeType, rc.Identifier),
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *AccessGenerator) appendAccessIdentityProviderResources(
	ctx context.Context,
	api *cf.API,
	rc *cf.ResourceContainer,
	scopeType string,
) error {
	params := cf.ListAccessIdentityProvidersParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		providers, info, err := api.ListAccessIdentityProviders(ctx, rc, params)
		if err != nil {
			return err
		}
		for _, provider := range providers {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				provider.ID,
				cloudflareResourceName(scopeType, rc.Identifier, provider.Name, provider.ID),
				"cloudflare_zero_trust_access_identity_provider",
				"cloudflare",
				accessScopeAttributes(scopeType, rc.Identifier),
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *AccessGenerator) appendAccessMTLSCertificateResources(
	ctx context.Context,
	api *cf.API,
	rc *cf.ResourceContainer,
	scopeType string,
) error {
	params := cf.ListAccessMutualTLSCertificatesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		certificates, info, err := api.ListAccessMutualTLSCertificates(ctx, rc, params)
		if err != nil {
			return err
		}
		for _, certificate := range certificates {
			if certificate.Certificate == "" {
				certificate, err = api.GetAccessMutualTLSCertificate(ctx, rc, certificate.ID)
				if err != nil {
					return err
				}
			}
			if certificate.Certificate == "" {
				continue
			}
			attributes := accessScopeAttributes(scopeType, rc.Identifier)
			attributes["certificate"] = certificate.Certificate
			g.Resources = append(g.Resources, terraformutils.NewResource(
				certificate.ID,
				cloudflareResourceName(scopeType, rc.Identifier, certificate.Name, certificate.ID),
				"cloudflare_zero_trust_access_mtls_certificate",
				"cloudflare",
				attributes,
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *AccessGenerator) appendAccessServiceTokenResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	serviceTokens, err := listAccessServiceTokens(ctx, api, rc)
	if err != nil {
		return err
	}
	for _, token := range serviceTokens {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			token.ID,
			cloudflareResourceName(scopeType, rc.Identifier, token.Name, token.ID),
			"cloudflare_zero_trust_access_service_token",
			"cloudflare",
			accessScopeAttributes(scopeType, rc.Identifier),
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func cloudflareAccessShortLivedCertificateResource(
	scopeType string,
	scopeID string,
	appID string,
	certificate cloudflareAccessShortLivedCertificate,
) (terraformutils.Resource, bool) {
	if scopeID == "" || appID == "" || certificate.ID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := accessScopeAttributes(scopeType, scopeID)
	attributes["app_id"] = appID
	resource := terraformutils.NewResource(
		appID,
		cloudflareResourceName(scopeType, scopeID, "short_lived_certificate", appID, certificate.ID),
		"cloudflare_zero_trust_access_short_lived_certificate",
		"cloudflare",
		attributes,
		[]string{},
		map[string]interface{}{},
	)
	setCloudflareImportID(&resource, fmt.Sprintf("%s/%s/%s", scopeType, scopeID, appID))
	return resource, true
}

func (g *AccessGenerator) appendAccessShortLivedCertificateResources(
	ctx context.Context,
	api *cf.API,
	rc *cf.ResourceContainer,
	scopeType string,
) error {
	applications, err := listAccessApplications(ctx, api, rc)
	if err != nil {
		return err
	}
	for _, app := range applications {
		if app.ID == "" {
			continue
		}
		certificate, err := getAccessShortLivedCertificate(ctx, api, rc, app.ID)
		if err != nil {
			if cloudflareAccessShortLivedCertificateOptionalError(err) {
				log.Printf("Skipping Cloudflare Access short-lived certificate discovery for %s/%s app %s: %v", scopeType, rc.Identifier, app.ID, err)
				continue
			}
			return err
		}
		resource, ok := cloudflareAccessShortLivedCertificateResource(scopeType, rc.Identifier, app.ID, certificate)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func getAccessShortLivedCertificate(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, appID string) (cloudflareAccessShortLivedCertificate, error) {
	response, err := api.Raw(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/%s/%s/access/apps/%s/ca", rc.Level, rc.Identifier, appID),
		nil,
		nil,
	)
	if err != nil {
		return cloudflareAccessShortLivedCertificate{}, err
	}
	var certificate cloudflareAccessShortLivedCertificate
	if err := json.Unmarshal(response.Result, &certificate); err != nil {
		return cloudflareAccessShortLivedCertificate{}, err
	}
	return certificate, nil
}

func listAccessServiceTokens(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.AccessServiceToken, error) {
	var serviceTokens []cf.AccessServiceToken
	for page := 1; ; page++ {
		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("per_page", strconv.Itoa(cloudflarePageSize))
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/%s/%s/access/service_tokens?%s", rc.Level, rc.Identifier, values.Encode()),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageTokens []cf.AccessServiceToken
		if err := json.Unmarshal(response.Result, &pageTokens); err != nil {
			return nil, err
		}
		serviceTokens = append(serviceTokens, pageTokens...)
		if response.ResultInfo == nil || !response.ResultInfo.HasMorePages() {
			break
		}
	}
	return serviceTokens, nil
}

func (g *AccessGenerator) appendAccountAccessPolicyResources(ctx context.Context, api *cf.API, accountID string) error {
	params := cf.ListAccessPoliciesParams{ResultInfo: cf.ResultInfo{Page: 1, PerPage: cloudflarePageSize}}
	for {
		policies, info, err := api.ListAccessPolicies(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return err
		}
		for _, policy := range policies {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				policy.ID,
				cloudflareResourceName("accounts", accountID, policy.Name, policy.ID),
				"cloudflare_zero_trust_access_policy",
				"cloudflare",
				map[string]string{"account_id": accountID},
				[]string{},
				map[string]interface{}{},
			))
		}
		if info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return nil
}

func (g *AccessGenerator) appendAccountAccessCustomPageResources(ctx context.Context, api *cf.API, accountID string) error {
	pages, err := listAccessCustomPages(ctx, api, cf.AccountIdentifier(accountID))
	if err != nil {
		return err
	}
	for _, page := range pages {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			page.UID,
			cloudflareResourceName("accounts", accountID, page.Name, page.UID),
			"cloudflare_zero_trust_access_custom_page",
			"cloudflare",
			map[string]string{"account_id": accountID, "uid": page.UID},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func listAccessCustomPages(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.AccessCustomPage, error) {
	var pages []cf.AccessCustomPage
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/%s/%s/access/custom_pages?%s", rc.Level, rc.Identifier, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageCustomPages []cf.AccessCustomPage
		if err := json.Unmarshal(response.Result, &pageCustomPages); err != nil {
			return nil, err
		}
		pages = append(pages, pageCustomPages...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return pages, nil
}

func (g *AccessGenerator) appendAccountAccessTagResources(ctx context.Context, api *cf.API, accountID string) error {
	tags, err := listAccessTags(ctx, api, cf.AccountIdentifier(accountID))
	if err != nil {
		return err
	}
	for _, tag := range tags {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			tag.Name,
			cloudflareResourceName("accounts", accountID, tag.Name),
			"cloudflare_zero_trust_access_tag",
			"cloudflare",
			map[string]string{"account_id": accountID, "name": tag.Name},
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}

func listAccessTags(ctx context.Context, api *cf.API, rc *cf.ResourceContainer) ([]cf.AccessTag, error) {
	var tags []cf.AccessTag
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/%s/%s/access/tags?%s", rc.Level, rc.Identifier, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageTags []cf.AccessTag
		if err := json.Unmarshal(response.Result, &pageTags); err != nil {
			return nil, err
		}
		tags = append(tags, pageTags...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return tags, nil
}

func (g *AccessGenerator) appendScopedAccessResources(ctx context.Context, api *cf.API, rc *cf.ResourceContainer, scopeType string) error {
	for _, f := range []func(context.Context, *cf.API, *cf.ResourceContainer, string) error{
		g.appendAccessApplicationResources,
		g.appendAccessGroupResources,
		g.appendAccessIdentityProviderResources,
		g.appendAccessMTLSCertificateResources,
		g.appendAccessServiceTokenResources,
		g.appendAccessShortLivedCertificateResources,
	} {
		if err := f(ctx, api, rc, scopeType); err != nil {
			return fmt.Errorf("%s/%s: %w", scopeType, rc.Identifier, err)
		}
	}
	return nil
}

func (g *AccessGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}

	accountID := g.accountID()
	if accountID != "" {
		account := cf.AccountIdentifier(accountID)
		if err := g.appendScopedAccessResources(ctx, api, account, "accounts"); err != nil {
			return err
		}
		for _, f := range []func(context.Context, *cf.API, string) error{
			g.appendAccountAccessPolicyResources,
			g.appendAccountAccessCustomPageResources,
			g.appendAccountAccessTagResources,
		} {
			if err := f(ctx, api, accountID); err != nil {
				return err
			}
		}
	}

	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		if err := g.appendScopedAccessResources(ctx, api, cf.ZoneIdentifier(zone.ID), "zones"); err != nil {
			return err
		}
	}
	return nil
}
