// SPDX-License-Identifier: Apache-2.0

//nolint:revive // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	identitytypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	idptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var CognitoAllowEmptyValues = []string{"tags."}

var CognitoAdditionalFields = map[string]interface{}{}

type CognitoGenerator struct {
	AWSService
}

type cognitoOptionalResourceLoader struct {
	name string
	load func() error
}

type cognitoIdentityPoolRef struct {
	id   string
	name string
}

type cognitoUserPoolRef struct {
	id           string
	name         string
	domain       string
	customDomain string
}

const (
	CognitoMaxResults = 60 // Required field for Cognito API

	cognitoIdentityPoolResourceType                = "cognito_identity_pool"
	cognitoIdentityPoolRolesAttachmentResourceType = "cognito_identity_pool_roles_attachment"
	cognitoUserPoolResourceType                    = "cognito_user_pool"
	cognitoUserPoolClientResourceType              = "cognito_user_pool_client"
	cognitoUserGroupResourceType                   = "cognito_user_group"
	cognitoIdentityProviderResourceType            = "cognito_identity_provider"
	cognitoResourceServerResourceType              = "cognito_resource_server"
	cognitoUserPoolDomainResourceType              = "cognito_user_pool_domain"
)

var cognitoIdentityPoolChildResourceTypes = []string{
	cognitoIdentityPoolRolesAttachmentResourceType,
}

var cognitoUserPoolChildResourceTypes = []string{
	cognitoUserPoolClientResourceType,
	cognitoUserGroupResourceType,
	cognitoIdentityProviderResourceType,
	cognitoResourceServerResourceType,
	cognitoUserPoolDomainResourceType,
}

var cognitoResourceTypes = append(
	[]string{cognitoIdentityPoolResourceType, cognitoUserPoolResourceType},
	append(cognitoIdentityPoolChildResourceTypes, cognitoUserPoolChildResourceTypes...)...,
)

func (g *CognitoGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}

	if g.shouldLoadIdentityPools() {
		svcCognitoIdentity := cognitoidentity.NewFromConfig(config)
		identityPools, err := g.loadIdentityPools(svcCognitoIdentity)
		if err != nil {
			return err
		}
		g.loadOptionalResources([]cognitoOptionalResourceLoader{
			{name: "identity pool role attachments", load: func() error { return g.loadIdentityPoolRolesAttachments(svcCognitoIdentity, identityPools) }},
		})
	}

	if g.shouldLoadUserPools() {
		svcCognitoIdentityProvider := cognitoidentityprovider.NewFromConfig(config)
		userPools, err := g.loadUserPools(svcCognitoIdentityProvider)
		if err != nil {
			return err
		}
		g.loadOptionalResources([]cognitoOptionalResourceLoader{
			{name: "user pool clients", load: func() error { return g.loadUserPoolClients(svcCognitoIdentityProvider, userPools) }},
			{name: "user groups", load: func() error { return g.loadUserGroups(svcCognitoIdentityProvider, userPools) }},
			{name: "identity providers", load: func() error { return g.loadIdentityProviders(svcCognitoIdentityProvider, userPools) }},
			{name: "resource servers", load: func() error { return g.loadResourceServers(svcCognitoIdentityProvider, userPools) }},
			{name: "user pool domains", load: func() error { return g.loadUserPoolDomains(userPools) }},
		})
	}

	return nil
}

func (g *CognitoGenerator) loadOptionalResources(loaders []cognitoOptionalResourceLoader) {
	for _, loader := range loaders {
		if err := loader.load(); err != nil {
			log.Printf("Skipping Cognito %s: %v", loader.name, err)
		}
	}
}

func (g *CognitoGenerator) loadIdentityPools(svc *cognitoidentity.Client) ([]cognitoIdentityPoolRef, error) {
	p := cognitoidentity.NewListIdentityPoolsPaginator(svc, &cognitoidentity.ListIdentityPoolsInput{
		MaxResults: aws.Int32(CognitoMaxResults),
	})

	var identityPools []cognitoIdentityPoolRef
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, pool := range page.IdentityPools {
			id := StringValue(pool.IdentityPoolId)
			resourceName := StringValue(pool.IdentityPoolName)
			if id == "" || resourceName == "" {
				continue
			}
			ref := cognitoIdentityPoolRef{id: id, name: resourceName}
			identityPools = append(identityPools, ref)

			resource := newCognitoIdentityPoolResource(ref)
			if g.shouldAppendCognitoResource(cognitoIdentityPoolResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return identityPools, nil
}

func (g *CognitoGenerator) loadIdentityPoolRolesAttachments(svc *cognitoidentity.Client, identityPools []cognitoIdentityPoolRef) error {
	for _, identityPool := range identityPools {
		if !g.shouldLoadIdentityPoolChildResourceType(cognitoIdentityPoolRolesAttachmentResourceType, identityPool.id) {
			continue
		}
		output, err := svc.GetIdentityPoolRoles(context.TODO(), &cognitoidentity.GetIdentityPoolRolesInput{
			IdentityPoolId: aws.String(identityPool.id),
		})
		if err != nil {
			if cognitoIdentityResourceMissing(err) {
				continue
			}
			return err
		}
		if !cognitoIdentityPoolRolesAttachmentConfigured(output) {
			continue
		}
		resource := newCognitoIdentityPoolRolesAttachmentResource(identityPool)
		if g.shouldAppendCognitoResource(cognitoIdentityPoolRolesAttachmentResourceType, resource) {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *CognitoGenerator) loadUserPools(svc *cognitoidentityprovider.Client) ([]cognitoUserPoolRef, error) {
	p := cognitoidentityprovider.NewListUserPoolsPaginator(svc, &cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: aws.Int32(CognitoMaxResults),
	})

	var userPools []cognitoUserPoolRef
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, pool := range page.UserPools {
			id := StringValue(pool.Id)
			resourceName := StringValue(pool.Name)
			if id == "" || resourceName == "" {
				continue
			}
			ref := cognitoUserPoolRef{id: id, name: resourceName}
			if output, err := svc.DescribeUserPool(context.TODO(), &cognitoidentityprovider.DescribeUserPoolInput{
				UserPoolId: aws.String(id),
			}); err == nil && output.UserPool != nil {
				ref.domain = StringValue(output.UserPool.Domain)
				ref.customDomain = StringValue(output.UserPool.CustomDomain)
			} else if err != nil && !cognitoIDPResourceMissing(err) {
				log.Printf("Skipping Cognito user pool domain metadata for %s: %v", id, err)
			}
			userPools = append(userPools, ref)

			resource := newCognitoUserPoolResource(ref)
			if g.shouldAppendCognitoResource(cognitoUserPoolResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return userPools, nil
}

func (g *CognitoGenerator) loadUserPoolClients(svc *cognitoidentityprovider.Client, userPools []cognitoUserPoolRef) error {
	for _, userPool := range userPools {
		if !g.shouldLoadUserPoolChildResourceType(cognitoUserPoolClientResourceType, userPool.id) {
			continue
		}
		p := cognitoidentityprovider.NewListUserPoolClientsPaginator(svc, &cognitoidentityprovider.ListUserPoolClientsInput{
			UserPoolId: aws.String(userPool.id),
			MaxResults: aws.Int32(CognitoMaxResults),
		})

		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if cognitoIDPResourceMissing(err) {
					break
				}
				return err
			}
			for _, poolClient := range page.UserPoolClients {
				clientID := StringValue(poolClient.ClientId)
				resourceName := StringValue(poolClient.ClientName)
				if clientID == "" || resourceName == "" {
					continue
				}
				resource := newCognitoUserPoolClientResource(userPool.id, clientID, resourceName)
				if g.shouldAppendCognitoResource(cognitoUserPoolClientResourceType, resource) {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *CognitoGenerator) loadUserGroups(svc *cognitoidentityprovider.Client, userPools []cognitoUserPoolRef) error {
	for _, userPool := range userPools {
		if !g.shouldLoadUserPoolChildResourceType(cognitoUserGroupResourceType, userPool.id) {
			continue
		}
		p := cognitoidentityprovider.NewListGroupsPaginator(svc, &cognitoidentityprovider.ListGroupsInput{
			UserPoolId: aws.String(userPool.id),
		})

		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if cognitoIDPResourceMissing(err) {
					break
				}
				return err
			}
			for _, group := range page.Groups {
				groupName := StringValue(group.GroupName)
				if groupName == "" {
					continue
				}
				resource := newCognitoUserGroupResource(userPool.id, groupName)
				if g.shouldAppendCognitoResource(cognitoUserGroupResourceType, resource) {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *CognitoGenerator) loadIdentityProviders(svc *cognitoidentityprovider.Client, userPools []cognitoUserPoolRef) error {
	for _, userPool := range userPools {
		if !g.shouldLoadUserPoolChildResourceType(cognitoIdentityProviderResourceType, userPool.id) {
			continue
		}
		p := cognitoidentityprovider.NewListIdentityProvidersPaginator(svc, &cognitoidentityprovider.ListIdentityProvidersInput{
			UserPoolId: aws.String(userPool.id),
		})

		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if cognitoIDPResourceMissing(err) {
					break
				}
				return err
			}
			for _, provider := range page.Providers {
				providerName := StringValue(provider.ProviderName)
				if providerName == "" {
					continue
				}
				resource := newCognitoIdentityProviderResource(userPool.id, providerName)
				if g.shouldAppendCognitoResource(cognitoIdentityProviderResourceType, resource) {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *CognitoGenerator) loadResourceServers(svc *cognitoidentityprovider.Client, userPools []cognitoUserPoolRef) error {
	for _, userPool := range userPools {
		if !g.shouldLoadUserPoolChildResourceType(cognitoResourceServerResourceType, userPool.id) {
			continue
		}
		p := cognitoidentityprovider.NewListResourceServersPaginator(svc, &cognitoidentityprovider.ListResourceServersInput{
			UserPoolId: aws.String(userPool.id),
		})

		for p.HasMorePages() {
			page, err := p.NextPage(context.TODO())
			if err != nil {
				if cognitoIDPResourceMissing(err) {
					break
				}
				return err
			}
			for _, server := range page.ResourceServers {
				identifier := StringValue(server.Identifier)
				if identifier == "" {
					continue
				}
				resource := newCognitoResourceServerResource(userPool.id, identifier)
				if g.shouldAppendCognitoResource(cognitoResourceServerResourceType, resource) {
					g.Resources = append(g.Resources, resource)
				}
			}
		}
	}
	return nil
}

func (g *CognitoGenerator) loadUserPoolDomains(userPools []cognitoUserPoolRef) error {
	for _, userPool := range userPools {
		for _, domain := range []string{userPool.domain, userPool.customDomain} {
			if domain == "" {
				continue
			}
			resource := newCognitoUserPoolDomainResource(userPool.id, domain)
			if g.shouldAppendCognitoResource(cognitoUserPoolDomainResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newCognitoIdentityPoolResource(identityPool cognitoIdentityPoolRef) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		identityPool.id,
		cognitoResourceName(identityPool.name, identityPool.id),
		"aws_cognito_identity_pool",
		"aws",
		[]string{})
}

func newCognitoIdentityPoolRolesAttachmentResource(identityPool cognitoIdentityPoolRef) terraformutils.Resource {
	return terraformutils.NewResource(
		identityPool.id,
		cognitoResourceName(identityPool.name, "roles", identityPool.id),
		"aws_cognito_identity_pool_roles_attachment",
		"aws",
		map[string]string{
			"identity_pool_id": identityPool.id,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func newCognitoUserPoolResource(userPool cognitoUserPoolRef) terraformutils.Resource {
	return terraformutils.NewSimpleResource(
		userPool.id,
		cognitoResourceName(userPool.name, userPool.id),
		"aws_cognito_user_pool",
		"aws",
		[]string{})
}

func newCognitoUserPoolClientResource(userPoolID, clientID, clientName string) terraformutils.Resource {
	// The AWS provider importer accepts user_pool_id/client_id, but refresh reads
	// framework state with id=client_id and user_pool_id as a separate attribute.
	return terraformutils.NewResource(
		clientID,
		cognitoResourceName(userPoolID, "client", clientName, clientID),
		"aws_cognito_user_pool_client",
		"aws",
		map[string]string{
			"user_pool_id": userPoolID,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func newCognitoUserGroupResource(userPoolID, groupName string) terraformutils.Resource {
	return terraformutils.NewResource(
		cognitoUserGroupResourceID(userPoolID, groupName),
		cognitoResourceName(userPoolID, "group", groupName),
		"aws_cognito_user_group",
		"aws",
		map[string]string{
			"name":         groupName,
			"user_pool_id": userPoolID,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func newCognitoIdentityProviderResource(userPoolID, providerName string) terraformutils.Resource {
	return terraformutils.NewResource(
		cognitoIdentityProviderResourceID(userPoolID, providerName),
		cognitoResourceName(userPoolID, "idp", providerName),
		"aws_cognito_identity_provider",
		"aws",
		map[string]string{
			"provider_name": providerName,
			"user_pool_id":  userPoolID,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func newCognitoResourceServerResource(userPoolID, identifier string) terraformutils.Resource {
	return terraformutils.NewResource(
		cognitoResourceServerResourceID(userPoolID, identifier),
		cognitoResourceName(userPoolID, "resource-server", identifier),
		"aws_cognito_resource_server",
		"aws",
		map[string]string{
			"identifier":   identifier,
			"user_pool_id": userPoolID,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func newCognitoUserPoolDomainResource(userPoolID, domain string) terraformutils.Resource {
	return terraformutils.NewResource(
		domain,
		cognitoResourceName(userPoolID, "domain", domain),
		"aws_cognito_user_pool_domain",
		"aws",
		map[string]string{
			"domain":       domain,
			"user_pool_id": userPoolID,
		},
		CognitoAllowEmptyValues,
		CognitoAdditionalFields)
}

func cognitoUserGroupResourceID(userPoolID, groupName string) string {
	return strings.Join([]string{userPoolID, groupName}, "/")
}

func cognitoIdentityProviderResourceID(userPoolID, providerName string) string {
	return strings.Join([]string{userPoolID, providerName}, ":")
}

func cognitoResourceServerResourceID(userPoolID, identifier string) string {
	return strings.Join([]string{userPoolID, identifier}, "|")
}

func cognitoUserPoolClientImportID(userPoolID, clientID string) string {
	return strings.Join([]string{userPoolID, clientID}, "/")
}

func cognitoResourceName(parts ...string) string {
	var nonempty []string
	for _, part := range parts {
		if part != "" {
			nonempty = append(nonempty, part)
		}
	}
	return strings.Join(nonempty, ":")
}

func (g *CognitoGenerator) shouldLoadIdentityPools() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedCognitoFilter() {
		return g.hasTypedFilterFor(cognitoIdentityPoolResourceType) || g.hasTypedIdentityPoolChildFilter()
	}
	return true
}

func (g *CognitoGenerator) shouldLoadUserPools() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedCognitoFilter() {
		return g.hasTypedFilterFor(cognitoUserPoolResourceType) || g.hasTypedUserPoolChildFilter()
	}
	return true
}

func (g *CognitoGenerator) shouldLoadIdentityPoolChildResourceType(serviceName, identityPoolID string) bool {
	hasTypedChildFilter := g.hasTypedFilterFor(serviceName)
	if g.hasTypedIdentityPoolChildFilter() && !hasTypedChildFilter {
		return false
	}
	if g.hasTypedCognitoFilter() && !hasTypedChildFilter && !g.hasTypedFilterFor(cognitoIdentityPoolResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.initialIDFiltersCanMatchIdentityPoolChild(serviceName, identityPoolID) {
		return false
	}
	if !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return g.identityPoolMatchesPreDiscoveryFilters(identityPoolID)
	}
	if hasTypedChildFilter && !g.hasIDFilterFor(serviceName) && !g.hasUntypedIDFilter() && g.hasTypedFilterFor(cognitoIdentityPoolResourceType) {
		return g.identityPoolMatchesPreDiscoveryFilters(identityPoolID)
	}
	return true
}

func (g *CognitoGenerator) shouldLoadUserPoolChildResourceType(serviceName, userPoolID string) bool {
	hasTypedChildFilter := g.hasTypedFilterFor(serviceName)
	if g.hasTypedUserPoolChildFilter() && !hasTypedChildFilter {
		return false
	}
	if g.hasTypedCognitoFilter() && !hasTypedChildFilter && !g.hasTypedFilterFor(cognitoUserPoolResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.initialIDFiltersCanMatchUserPoolChild(serviceName, userPoolID) {
		return false
	}
	if !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return g.userPoolMatchesPreDiscoveryFilters(userPoolID)
	}
	if hasTypedChildFilter && !g.hasIDFilterFor(serviceName) && !g.hasUntypedIDFilter() && g.hasTypedFilterFor(cognitoUserPoolResourceType) {
		return g.userPoolMatchesPreDiscoveryFilters(userPoolID)
	}
	return true
}

func (g *CognitoGenerator) shouldAppendCognitoResource(serviceName string, resource terraformutils.Resource) bool {
	if !g.resourceMatchesInitialIDFilters(serviceName, resource) {
		return false
	}
	if g.hasTypedCognitoFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *CognitoGenerator) identityPoolMatchesPreDiscoveryFilters(identityPoolID string) bool {
	resource := terraformutils.NewSimpleResource(identityPoolID, identityPoolID, "aws_cognito_identity_pool", "aws", CognitoAllowEmptyValues)
	if !g.resourceMatchesInitialIDFilters(cognitoIdentityPoolResourceType, resource) {
		return false
	}
	return !g.hasTypedNonIDFilterFor(cognitoIdentityPoolResourceType)
}

func (g *CognitoGenerator) userPoolMatchesPreDiscoveryFilters(userPoolID string) bool {
	resource := terraformutils.NewSimpleResource(userPoolID, userPoolID, "aws_cognito_user_pool", "aws", CognitoAllowEmptyValues)
	if !g.resourceMatchesInitialIDFilters(cognitoUserPoolResourceType, resource) {
		return false
	}
	return !g.hasTypedNonIDFilterFor(cognitoUserPoolResourceType)
}

func (g *CognitoGenerator) resourceMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !filter.Filter(resource) && !cognitoResourceMatchesAlternateIDFilter(serviceName, resource, filter.AcceptableValues) {
			return false
		}
	}
	return true
}

func cognitoResourceMatchesAlternateIDFilter(serviceName string, resource terraformutils.Resource, values []string) bool {
	if serviceName != cognitoUserPoolClientResourceType {
		return false
	}
	userPoolID := resource.InstanceState.Attributes["user_pool_id"]
	clientID := resource.InstanceState.ID
	if userPoolID == "" || clientID == "" {
		return false
	}
	return cognitoAnyAcceptableIDMatches(values, func(value string) bool {
		return value == cognitoUserPoolClientImportID(userPoolID, clientID)
	})
}

func (g *CognitoGenerator) initialIDFiltersCanMatchIdentityPoolChild(serviceName, identityPoolID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !cognitoAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return cognitoIdentityPoolChildIDMayBelongToIdentityPool(serviceName, identityPoolID, value)
		}) {
			return false
		}
	}
	return true
}

func (g *CognitoGenerator) initialIDFiltersCanMatchUserPoolChild(serviceName, userPoolID string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !cognitoAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return cognitoUserPoolChildIDMayBelongToUserPool(serviceName, userPoolID, value)
		}) {
			return false
		}
	}
	return true
}

func cognitoAnyAcceptableIDMatches(values []string, match func(string) bool) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

func cognitoIdentityPoolChildIDMayBelongToIdentityPool(serviceName, identityPoolID, value string) bool {
	switch serviceName {
	case cognitoIdentityPoolRolesAttachmentResourceType:
		return value == identityPoolID
	default:
		return true
	}
}

func cognitoUserPoolChildIDMayBelongToUserPool(serviceName, userPoolID, value string) bool {
	switch serviceName {
	case cognitoUserPoolClientResourceType:
		return strings.HasPrefix(value, userPoolID+"/") || !strings.Contains(value, "/")
	case cognitoUserGroupResourceType:
		return strings.HasPrefix(value, userPoolID+"/")
	case cognitoIdentityProviderResourceType:
		return strings.HasPrefix(value, userPoolID+":")
	case cognitoResourceServerResourceType:
		return strings.HasPrefix(value, userPoolID+"|")
	case cognitoUserPoolDomainResourceType:
		return true
	default:
		return true
	}
}

func (g *CognitoGenerator) hasTypedIdentityPoolChildFilter() bool {
	for _, serviceName := range cognitoIdentityPoolChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasTypedUserPoolChildFilter() bool {
	for _, serviceName := range cognitoUserPoolChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasTypedCognitoFilter() bool {
	for _, serviceName := range cognitoResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasTypedNonIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func (g *CognitoGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}

func cognitoIdentityPoolRolesAttachmentConfigured(output *cognitoidentity.GetIdentityPoolRolesOutput) bool {
	if output == nil {
		return false
	}
	return len(output.Roles) > 0 || len(output.RoleMappings) > 0
}

func cognitoIdentityResourceMissing(err error) bool {
	var notFound *identitytypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}

func cognitoIDPResourceMissing(err error) bool {
	var notFound *idptypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}

func (g *CognitoGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_cognito_user_pool" {
			continue
		}
		if _, ok := r.InstanceState.Attributes["admin_create_user_config.0.unused_account_validity_days"]; ok {
			if _, okpp := r.InstanceState.Attributes["admin_create_user_config.0.unused_account_validity_days"]; okpp {
				delete(r.Item["admin_create_user_config"].([]interface{})[0].(map[string]interface{}), "unused_account_validity_days")
			}
		}
		if _, ok := r.InstanceState.Attributes["sms_verification_message"]; ok {
			if _, oktmp := r.InstanceState.Attributes["verification_message_template.0.sms_message"]; oktmp {
				delete(r.Item, "sms_verification_message")
			}
		}
		if _, ok := r.InstanceState.Attributes["email_verification_message"]; ok {
			if _, oktmp := r.InstanceState.Attributes["verification_message_template.0.email_message"]; oktmp {
				delete(r.Item, "email_verification_message")
			}
		}
		if _, ok := r.InstanceState.Attributes["email_verification_subject"]; ok {
			if _, oktmp := r.InstanceState.Attributes["verification_message_template.0.email_subject"]; oktmp {
				delete(r.Item, "email_verification_subject")
			}
		}
	}
	return nil
}
