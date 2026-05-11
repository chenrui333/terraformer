// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/customerprofiles"
	customerprofilestypes "github.com/aws/aws-sdk-go-v2/service/customerprofiles/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const customerProfilesDomainResourceType = "aws_customerprofiles_domain"

var customerProfilesAllowEmptyValues = []string{"tags."}

type CustomerProfilesGenerator struct {
	AWSService
}

func (g *CustomerProfilesGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := customerprofiles.NewFromConfig(config)
	input := &customerprofiles.ListDomainsInput{}
	for {
		page, err := svc.ListDomains(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, domain := range page.Items {
			domainName := StringValue(domain.DomainName)
			if domainName == "" {
				continue
			}
			output, err := svc.GetDomain(context.TODO(), &customerprofiles.GetDomainInput{
				DomainName: &domainName,
			})
			if err != nil {
				if customerProfilesNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newCustomerProfilesDomainResource(output); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
		if page.NextToken == nil {
			break
		}
		input.NextToken = page.NextToken
	}
	return nil
}

func newCustomerProfilesDomainResource(output *customerprofiles.GetDomainOutput) (terraformutils.Resource, bool) {
	if output == nil || StringValue(output.DomainName) == "" || output.DefaultExpirationDays == nil {
		return terraformutils.Resource{}, false
	}
	domainName := StringValue(output.DomainName)
	attributes := map[string]string{
		"domain_name":             domainName,
		"default_expiration_days": strconv.Itoa(int(*output.DefaultExpirationDays)),
	}
	if deadLetterQueueURL := StringValue(output.DeadLetterQueueUrl); deadLetterQueueURL != "" {
		attributes["dead_letter_queue_url"] = deadLetterQueueURL
	}
	if defaultEncryptionKey := StringValue(output.DefaultEncryptionKey); defaultEncryptionKey != "" {
		attributes["default_encryption_key"] = defaultEncryptionKey
	}
	return terraformutils.NewResource(
		customerProfilesDomainImportID(domainName),
		customerProfilesResourceName("domain", domainName),
		customerProfilesDomainResourceType,
		"aws",
		attributes,
		customerProfilesAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func customerProfilesDomainImportID(domainName string) string {
	return domainName
}

func customerProfilesResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func customerProfilesNotFound(err error) bool {
	var notFound *customerprofilestypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
