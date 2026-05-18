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
	"sort"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type CertificatesGenerator struct {
	CloudflareService
}

type cloudflareCertificateRawResource map[string]interface{}

type cloudflareCertificateDiscovery struct {
	name      string
	scope     string
	resources *[]terraformutils.Resource
	discover  func() error
}

func cloudflareCertificateString(resource cloudflareCertificateRawResource, keys ...string) string {
	for _, key := range keys {
		value, ok := resource[key].(string)
		if ok && value != "" {
			return value
		}
	}
	return ""
}

func cloudflareCertificateStringSlice(resource cloudflareCertificateRawResource, key string) []string {
	value, ok := resource[key]
	if !ok {
		return nil
	}
	switch values := value.(type) {
	case []string:
		return values
	case []interface{}:
		result := make([]string, 0, len(values))
		for _, item := range values {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func cloudflareCertificateNumberString(resource cloudflareCertificateRawResource, key string) (string, bool) {
	value, ok := resource[key]
	if !ok {
		return "", false
	}
	switch v := value.(type) {
	case float64:
		if v == 0 {
			return "", false
		}
		return strconv.Itoa(int(v)), true
	case int:
		if v == 0 {
			return "", false
		}
		return strconv.Itoa(v), true
	case json.Number:
		if v == "0" || v == "" {
			return "", false
		}
		return v.String(), true
	default:
		return "", false
	}
}

func cloudflareCertificateBool(resource cloudflareCertificateRawResource, key string) (bool, bool) {
	value, ok := resource[key]
	if !ok {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func cloudflareCertificateDeletedStatus(status string) bool {
	switch strings.ToLower(status) {
	case "deleted", "pending_deletion", "deletion_timed_out":
		return true
	default:
		return false
	}
}

func cloudflareCertificateOptionalDiscoveryError(err error) bool {
	var notFoundErr *cf.NotFoundError
	if errors.As(err, &notFoundErr) {
		return cloudflareCertificateOptionalErrorMessage(notFoundErr.Error(), notFoundErr.ErrorMessages())
	}

	var requestErr *cf.RequestError
	if errors.As(err, &requestErr) {
		return cloudflareCertificateOptionalErrorMessage(requestErr.Error(), requestErr.ErrorMessages())
	}

	var authenticationErr *cf.AuthenticationError
	if errors.As(err, &authenticationErr) {
		return cloudflareCertificateOptionalErrorMessage(authenticationErr.Error(), authenticationErr.ErrorMessages())
	}

	var authorizationErr *cf.AuthorizationError
	if errors.As(err, &authorizationErr) {
		return cloudflareCertificateOptionalErrorMessage(authorizationErr.Error(), authorizationErr.ErrorMessages())
	}

	return false
}

func cloudflareCertificateOptionalErrorMessage(message string, errorMessages []string) bool {
	messages := append([]string{message}, errorMessages...)
	for _, msg := range messages {
		normalized := strings.ToLower(msg)
		for _, marker := range []string{
			"access denied",
			"feature is not available",
			"missing permission",
			"not authorized",
			"not configured",
			"not enabled",
			"not entitled",
			"permission denied",
			"requires a paid plan",
			"unauthorized",
			"upgrade your plan",
		} {
			if strings.Contains(normalized, marker) {
				return true
			}
		}
	}
	return false
}

func runCloudflareCertificateDiscoveries(discoveries []cloudflareCertificateDiscovery) error {
	for _, discovery := range discoveries {
		if discovery.discover == nil {
			continue
		}
		resourceCount := 0
		if discovery.resources != nil {
			resourceCount = len(*discovery.resources)
		}
		if err := discovery.discover(); err != nil {
			if discovery.resources != nil && resourceCount <= len(*discovery.resources) {
				*discovery.resources = (*discovery.resources)[:resourceCount]
			}
			if cloudflareCertificateOptionalDiscoveryError(err) {
				log.Printf("Skipping Cloudflare certificate %s discovery for %s: %v", discovery.name, discovery.scope, err)
				continue
			}
			return fmt.Errorf("discover Cloudflare certificate %s for %s: %w", discovery.name, discovery.scope, err)
		}
	}
	return nil
}

func cloudflareCertificatePagedPath(path string, page int, cursor string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%s%s", path, separator, cloudflarePaginationQuery(page, cursor))
}

func listCloudflareCertificateResources(
	ctx context.Context,
	api *cf.API,
	path string,
) ([]cloudflareCertificateRawResource, error) {
	var resources []cloudflareCertificateRawResource
	page, cursor := 1, ""
	for {
		response, err := api.Raw(ctx, http.MethodGet, cloudflareCertificatePagedPath(path, page, cursor), nil, nil)
		if err != nil {
			return nil, err
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return resources, nil
		}

		var pageResources []cloudflareCertificateRawResource
		if err := json.Unmarshal(response.Result, &pageResources); err != nil {
			return nil, err
		}
		resources = append(resources, pageResources...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return resources, nil
}

func listCloudflareMTLSCertificates(ctx context.Context, api *cf.API, accountID string) ([]cf.MTLSCertificate, error) {
	var allCertificates []cf.MTLSCertificate
	params := cf.ListMTLSCertificatesParams{PaginationOptions: cf.PaginationOptions{Page: 1, PerPage: cloudflarePageSize}}
	for {
		certificates, info, err := api.ListMTLSCertificates(ctx, cf.AccountIdentifier(accountID), params)
		if err != nil {
			return nil, err
		}
		allCertificates = append(allCertificates, certificates...)
		if !info.HasMorePages() {
			break
		}
		params.Page++
	}
	return allCertificates, nil
}

func (g *CertificatesGenerator) appendMTLSCertificateResources(ctx context.Context, api *cf.API, accountID string) ([]cf.MTLSCertificate, error) {
	certificates, err := listCloudflareMTLSCertificates(ctx, api, accountID)
	if err != nil {
		return nil, err
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
	return certificates, nil
}

func cloudflareZoneCertificateResource(
	zone cf.Zone,
	id string,
	resourceType string,
	resourceNamePrefix string,
	attributes map[string]string,
	additionalFields map[string]interface{},
	nameParts ...string,
) (terraformutils.Resource, bool) {
	if zone.ID == "" || id == "" {
		return terraformutils.Resource{}, false
	}
	if attributes == nil {
		attributes = map[string]string{}
	}
	attributes["zone_id"] = zone.ID
	if additionalFields == nil {
		additionalFields = map[string]interface{}{}
	}
	parts := append([]string{zone.Name, resourceNamePrefix}, nameParts...)
	parts = append(parts, id)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(parts...),
		resourceType,
		"cloudflare",
		attributes,
		[]string{},
		additionalFields,
	)
	setCloudflareImportID(&resource, zone.ID+"/"+id)
	return resource, true
}

func cloudflareCertificatePackImportable(certificatePack cloudflareCertificateRawResource) bool {
	if cloudflareCertificateString(certificatePack, "id") == "" {
		return false
	}
	if cloudflareCertificateDeletedStatus(cloudflareCertificateString(certificatePack, "status")) {
		return false
	}
	if cloudflareCertificateString(certificatePack, "certificate_authority") == "" ||
		!strings.EqualFold(cloudflareCertificateString(certificatePack, "type"), "advanced") ||
		cloudflareCertificateString(certificatePack, "validation_method") == "" {
		return false
	}
	_, ok := cloudflareCertificateNumberString(certificatePack, "validity_days")
	return ok
}

func cloudflareClientCertificateImportable(certificate cloudflareCertificateRawResource) bool {
	if cloudflareCertificateString(certificate, "id") == "" ||
		cloudflareCertificateString(certificate, "csr") == "" {
		return false
	}
	switch strings.ToLower(cloudflareCertificateString(certificate, "status")) {
	case "revoked", "pending_revocation":
		return false
	}
	_, ok := cloudflareCertificateNumberString(certificate, "validity_days")
	return ok
}

func cloudflareCustomOriginTrustStoreImportable(trustStore cloudflareCertificateRawResource) bool {
	return cloudflareCertificateString(trustStore, "id") != "" &&
		cloudflareCertificateString(trustStore, "certificate") != "" &&
		!cloudflareCertificateDeletedStatus(cloudflareCertificateString(trustStore, "status"))
}

func cloudflareOriginCACertificateImportable(certificate cloudflareCertificateRawResource) bool {
	return cloudflareCertificateString(certificate, "id") != "" &&
		cloudflareCertificateString(certificate, "csr") != "" &&
		cloudflareCertificateString(certificate, "request_type") != "" &&
		cloudflareCertificateString(certificate, "revoked_at") == "" &&
		len(cloudflareCertificateStringSlice(certificate, "hostnames")) > 0
}

func cloudflareCertificateListAttributes(field string, values []string, attributes map[string]string, additionalFields map[string]interface{}) {
	sort.Strings(values)
	attributes[field+".#"] = strconv.Itoa(len(values))
	list := make([]interface{}, 0, len(values))
	for i, value := range values {
		attributes[fmt.Sprintf("%s.%d", field, i)] = value
		list = append(list, value)
	}
	additionalFields[field] = list
}

func cloudflareCertificatePackResource(zone cf.Zone, certificatePack cloudflareCertificateRawResource) (terraformutils.Resource, bool) {
	id := cloudflareCertificateString(certificatePack, "id")
	validityDays, _ := cloudflareCertificateNumberString(certificatePack, "validity_days")
	attributes := map[string]string{
		"certificate_authority": cloudflareCertificateString(certificatePack, "certificate_authority"),
		"type":                  cloudflareCertificateString(certificatePack, "type"),
		"validation_method":     cloudflareCertificateString(certificatePack, "validation_method"),
		"validity_days":         validityDays,
	}
	additionalFields := map[string]interface{}{}
	if cloudflareBranding, ok := cloudflareCertificateBool(certificatePack, "cloudflare_branding"); ok && cloudflareBranding {
		attributes["cloudflare_branding"] = strconv.FormatBool(cloudflareBranding)
		additionalFields["cloudflare_branding"] = cloudflareBranding
	}
	return cloudflareZoneCertificateResource(
		zone,
		id,
		"cloudflare_certificate_pack",
		"certificate_pack",
		attributes,
		additionalFields,
		cloudflareCertificateString(certificatePack, "status"),
	)
}

func cloudflareClientCertificateResource(zone cf.Zone, certificate cloudflareCertificateRawResource) (terraformutils.Resource, bool) {
	id := cloudflareCertificateString(certificate, "id")
	validityDays, _ := cloudflareCertificateNumberString(certificate, "validity_days")
	attributes := map[string]string{
		"csr":           cloudflareCertificateString(certificate, "csr"),
		"validity_days": validityDays,
	}
	return cloudflareZoneCertificateResource(
		zone,
		id,
		"cloudflare_client_certificate",
		"client_certificate",
		attributes,
		map[string]interface{}{},
		cloudflareCertificateString(certificate, "common_name", "fingerprint_sha256"),
	)
}

func cloudflareCustomOriginTrustStoreResource(zone cf.Zone, trustStore cloudflareCertificateRawResource) (terraformutils.Resource, bool) {
	id := cloudflareCertificateString(trustStore, "id")
	attributes := map[string]string{"certificate": cloudflareCertificateString(trustStore, "certificate")}
	return cloudflareZoneCertificateResource(
		zone,
		id,
		"cloudflare_custom_origin_trust_store",
		"custom_origin_trust_store",
		attributes,
		map[string]interface{}{},
		cloudflareCertificateString(trustStore, "issuer", "status"),
	)
}

func cloudflareOriginCACertificateResource(zone cf.Zone, certificate cloudflareCertificateRawResource) (terraformutils.Resource, bool) {
	id := cloudflareCertificateString(certificate, "id")
	attributes := map[string]string{
		"csr":          cloudflareCertificateString(certificate, "csr"),
		"request_type": cloudflareCertificateString(certificate, "request_type"),
	}
	if requestedValidity, ok := cloudflareCertificateNumberString(certificate, "requested_validity"); ok {
		attributes["requested_validity"] = requestedValidity
	}
	hostnames := cloudflareCertificateStringSlice(certificate, "hostnames")
	additionalFields := map[string]interface{}{}
	cloudflareCertificateListAttributes("hostnames", hostnames, attributes, additionalFields)
	resource := terraformutils.NewResource(
		id,
		cloudflareResourceName(zone.Name, "origin_ca_certificate", strings.Join(hostnames, "_"), id),
		"cloudflare_origin_ca_certificate",
		"cloudflare",
		attributes,
		[]string{},
		additionalFields,
	)
	return resource, id != ""
}

func cloudflareCertificateAuthorityHostnameAssociationsResource(
	zone cf.Zone,
	mtlsCertificateID string,
	hostnames []cf.HostnameAssociation,
) (terraformutils.Resource, bool) {
	if zone.ID == "" || len(hostnames) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{"zone_id": zone.ID}
	if mtlsCertificateID != "" {
		attributes["mtls_certificate_id"] = mtlsCertificateID
	}
	additionalFields := map[string]interface{}{}
	values := make([]string, 0, len(hostnames))
	for _, hostname := range hostnames {
		if hostname != "" {
			values = append(values, string(hostname))
		}
	}
	if len(values) == 0 {
		return terraformutils.Resource{}, false
	}
	cloudflareCertificateListAttributes("hostnames", values, attributes, additionalFields)

	importID := zone.ID
	nameParts := []string{zone.Name, "certificate_authorities_hostname_associations"}
	if mtlsCertificateID != "" {
		importID += "/" + mtlsCertificateID
		nameParts = append(nameParts, mtlsCertificateID)
	} else {
		nameParts = append(nameParts, "managed_ca")
	}
	resource := terraformutils.NewResource(
		zone.ID,
		cloudflareResourceName(nameParts...),
		"cloudflare_certificate_authorities_hostname_associations",
		"cloudflare",
		attributes,
		[]string{},
		additionalFields,
	)
	setCloudflareImportID(&resource, importID)
	return resource, true
}

func (g *CertificatesGenerator) appendCertificatePackResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	certificatePacks, err := listCloudflareCertificateResources(ctx, api, fmt.Sprintf("/zones/%s/ssl/certificate_packs?status=all", zone.ID))
	if err != nil {
		return err
	}
	for _, certificatePack := range certificatePacks {
		if !cloudflareCertificatePackImportable(certificatePack) {
			continue
		}
		resource, ok := cloudflareCertificatePackResource(zone, certificatePack)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *CertificatesGenerator) appendClientCertificateResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	certificates, err := listCloudflareCertificateResources(ctx, api, fmt.Sprintf("/zones/%s/client_certificates?status=all", zone.ID))
	if err != nil {
		return err
	}
	for _, certificate := range certificates {
		if !cloudflareClientCertificateImportable(certificate) {
			continue
		}
		resource, ok := cloudflareClientCertificateResource(zone, certificate)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *CertificatesGenerator) appendCustomOriginTrustStoreResources(ctx context.Context, api *cf.API, zone cf.Zone) error {
	trustStores, err := listCloudflareCertificateResources(ctx, api, fmt.Sprintf("/zones/%s/acm/custom_trust_store", zone.ID))
	if err != nil {
		return err
	}
	for _, trustStore := range trustStores {
		if !cloudflareCustomOriginTrustStoreImportable(trustStore) {
			continue
		}
		resource, ok := cloudflareCustomOriginTrustStoreResource(zone, trustStore)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *CertificatesGenerator) appendOriginCACertificateResources(
	ctx context.Context,
	api *cf.API,
	zone cf.Zone,
	seen map[string]struct{},
) error {
	values := url.Values{}
	values.Set("zone_id", zone.ID)
	certificates, err := listCloudflareCertificateResources(ctx, api, "/certificates?"+values.Encode())
	if err != nil {
		return err
	}
	for _, certificate := range certificates {
		if !cloudflareOriginCACertificateImportable(certificate) {
			continue
		}
		id := cloudflareCertificateString(certificate, "id")
		if _, ok := seen[id]; ok {
			continue
		}
		resource, ok := cloudflareOriginCACertificateResource(zone, certificate)
		if ok {
			g.Resources = append(g.Resources, resource)
			seen[id] = struct{}{}
		}
	}
	return nil
}

func (g *CertificatesGenerator) appendCertificateAuthorityHostnameAssociationsResources(
	ctx context.Context,
	api *cf.API,
	zone cf.Zone,
	mtlsCertificates []cf.MTLSCertificate,
) error {
	hostnames, err := api.ListCertificateAuthoritiesHostnameAssociations(
		ctx,
		cf.ZoneIdentifier(zone.ID),
		cf.ListCertificateAuthoritiesHostnameAssociationsParams{},
	)
	if err != nil {
		return err
	}
	if resource, ok := cloudflareCertificateAuthorityHostnameAssociationsResource(zone, "", hostnames); ok {
		g.Resources = append(g.Resources, resource)
	}

	for _, certificate := range mtlsCertificates {
		if certificate.ID == "" {
			continue
		}
		hostnames, err := api.ListCertificateAuthoritiesHostnameAssociations(
			ctx,
			cf.ZoneIdentifier(zone.ID),
			cf.ListCertificateAuthoritiesHostnameAssociationsParams{MTLSCertificateID: certificate.ID},
		)
		if err != nil {
			return err
		}
		if resource, ok := cloudflareCertificateAuthorityHostnameAssociationsResource(zone, certificate.ID, hostnames); ok {
			g.Resources = append(g.Resources, resource)
		}
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

func (g *CertificatesGenerator) appendZoneCertificateResources(
	ctx context.Context,
	api *cf.API,
	zone cf.Zone,
	mtlsCertificates []cf.MTLSCertificate,
	seenOriginCACertificates map[string]struct{},
) error {
	return runCloudflareCertificateDiscoveries([]cloudflareCertificateDiscovery{
		{
			name:      "certificate packs",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendCertificatePackResources(ctx, api, zone)
			},
		},
		{
			name:      "client certificates",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendClientCertificateResources(ctx, api, zone)
			},
		},
		{
			name:      "custom origin trust stores",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendCustomOriginTrustStoreResources(ctx, api, zone)
			},
		},
		{
			name:      "origin CA certificates",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendOriginCACertificateResources(ctx, api, zone, seenOriginCACertificates)
			},
		},
		{
			name:      "certificate authority hostname associations",
			scope:     zone.ID,
			resources: &g.Resources,
			discover: func() error {
				return g.appendCertificateAuthorityHostnameAssociationsResources(ctx, api, zone, mtlsCertificates)
			},
		},
	})
}

func (g *CertificatesGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	var mtlsCertificates []cf.MTLSCertificate
	if g.accountID() != "" {
		mtlsCertificates, err = g.appendMTLSCertificateResources(ctx, api, g.accountID())
		if err != nil {
			return err
		}
	}
	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	seenOriginCACertificates := map[string]struct{}{}
	for _, zone := range zones {
		if err := g.appendCustomHostnameResources(ctx, api, zone); err != nil {
			return err
		}
		if err := g.appendZoneCertificateResources(ctx, api, zone, mtlsCertificates, seenOriginCACertificates); err != nil {
			return err
		}
	}
	return nil
}
