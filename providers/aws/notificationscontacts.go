// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/notificationscontacts"
	notificationscontactstypes "github.com/aws/aws-sdk-go-v2/service/notificationscontacts/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const notificationsContactsEmailContactResourceType = "aws_notificationscontacts_email_contact"

var notificationsContactsAllowEmptyValues = []string{"tags."}

type NotificationsContactsGenerator struct {
	AWSService
}

func (g *NotificationsContactsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	config = notificationsEastOnlyConfig(config)
	svc := notificationscontacts.NewFromConfig(config)
	p := notificationscontacts.NewListEmailContactsPaginator(svc, &notificationscontacts.ListEmailContactsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if notificationsContactsNotFound(err) {
				return nil
			}
			return err
		}
		for _, contact := range page.EmailContacts {
			if resource, ok := newNotificationsContactsEmailContactResource(contact); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newNotificationsContactsEmailContactResource(contact notificationscontactstypes.EmailContact) (terraformutils.Resource, bool) {
	arn := StringValue(contact.Arn)
	name := StringValue(contact.Name)
	address := StringValue(contact.Address)
	if arn == "" || name == "" || address == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		notificationsContactsEmailContactImportID(arn),
		notificationsContactsResourceName("email_contact", name, arn),
		notificationsContactsEmailContactResourceType,
		"aws",
		map[string]string{
			"arn":           arn,
			"name":          name,
			"email_address": address,
		},
		notificationsContactsAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func notificationsContactsEmailContactImportID(arn string) string {
	return arn
}

func notificationsContactsResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}

func notificationsContactsNotFound(err error) bool {
	var notFound *notificationscontactstypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
