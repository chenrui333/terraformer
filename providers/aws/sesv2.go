// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesv2types "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	sesv2ConfigurationSetResourceType                 = "aws_sesv2_configuration_set"
	sesv2ConfigurationSetEventDestinationResourceType = "aws_sesv2_configuration_set_event_destination"
	sesv2ContactListResourceType                      = "aws_sesv2_contact_list"
	sesv2DedicatedIPPoolResourceType                  = "aws_sesv2_dedicated_ip_pool"
	sesv2EmailIdentityResourceType                    = "aws_sesv2_email_identity"
	sesv2EmailIdentityFeedbackAttributesResourceType  = "aws_sesv2_email_identity_feedback_attributes"
	sesv2EmailIdentityMailFromAttributesResourceType  = "aws_sesv2_email_identity_mail_from_attributes"
	sesv2EmailIdentityPolicyResourceType              = "aws_sesv2_email_identity_policy"
	sesv2ResourceIDSeparator                          = "|"
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
	if err := g.loadContactLists(svc); err != nil {
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

func (g *SesV2Generator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type != sesv2EmailIdentityPolicyResourceType {
			continue
		}
		policy, ok := resource.Item["policy"].(string)
		if !ok || policy == "" {
			continue
		}
		g.Resources[i].Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
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
			if configurationSetName == "" {
				continue
			}
			if resource, ok := newSESV2ConfigurationSetResource(configurationSetName); ok {
				g.Resources = append(g.Resources, resource)
			}
			if err := g.loadConfigurationSetEventDestinations(svc, configurationSetName); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SesV2Generator) loadConfigurationSetEventDestinations(svc *sesv2.Client, configurationSetName string) error {
	if configurationSetName == "" {
		return nil
	}
	output, err := svc.GetConfigurationSetEventDestinations(context.TODO(), &sesv2.GetConfigurationSetEventDestinationsInput{
		ConfigurationSetName: &configurationSetName,
	})
	if err != nil {
		if sesv2NotFound(err) {
			return nil
		}
		return err
	}
	if output == nil {
		return nil
	}
	for _, destination := range output.EventDestinations {
		if resource, ok := newSESV2ConfigurationSetEventDestinationResource(configurationSetName, destination); ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *SesV2Generator) loadContactLists(svc *sesv2.Client) error {
	p := sesv2.NewListContactListsPaginator(svc, &sesv2.ListContactListsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, contactList := range page.ContactLists {
			contactListName := StringValue(contactList.ContactListName)
			if contactListName == "" {
				continue
			}
			output, err := svc.GetContactList(context.TODO(), &sesv2.GetContactListInput{
				ContactListName: &contactListName,
			})
			if err != nil {
				if sesv2NotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newSESV2ContactListResource(output); ok {
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
			if resource, ok := newSESV2EmailIdentityFeedbackAttributesResource(identityName, output); ok {
				g.Resources = append(g.Resources, resource)
			}
			if resource, ok := newSESV2EmailIdentityMailFromAttributesResource(identityName, output); ok {
				g.Resources = append(g.Resources, resource)
			}
			if err := g.loadEmailIdentityPolicies(svc, identityName); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *SesV2Generator) loadEmailIdentityPolicies(svc *sesv2.Client, identityName string) error {
	if identityName == "" {
		return nil
	}
	output, err := svc.GetEmailIdentityPolicies(context.TODO(), &sesv2.GetEmailIdentityPoliciesInput{
		EmailIdentity: &identityName,
	})
	if err != nil {
		if sesv2NotFound(err) {
			return nil
		}
		return err
	}
	if output == nil {
		return nil
	}
	policyNames := make([]string, 0, len(output.Policies))
	for policyName := range output.Policies {
		policyNames = append(policyNames, policyName)
	}
	sort.Strings(policyNames)
	for _, policyName := range policyNames {
		if resource, ok := newSESV2EmailIdentityPolicyResource(identityName, policyName, output.Policies[policyName]); ok {
			g.Resources = append(g.Resources, resource)
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

func newSESV2ConfigurationSetEventDestinationResource(configurationSetName string, destination sesv2types.EventDestination) (terraformutils.Resource, bool) {
	eventDestinationName := StringValue(destination.Name)
	if configurationSetName == "" || eventDestinationName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2ConfigurationSetEventDestinationImportID(configurationSetName, eventDestinationName),
		sesv2ResourceName("configuration_set_event_destination", configurationSetName, eventDestinationName),
		sesv2ConfigurationSetEventDestinationResourceType,
		"aws",
		map[string]string{
			"configuration_set_name": configurationSetName,
			"event_destination_name": eventDestinationName,
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSESV2ContactListResource(output *sesv2.GetContactListOutput) (terraformutils.Resource, bool) {
	if output == nil {
		return terraformutils.Resource{}, false
	}
	contactListName := StringValue(output.ContactListName)
	if contactListName == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"contact_list_name": contactListName,
	}
	if description := StringValue(output.Description); description != "" {
		attributes["description"] = description
	}
	additionalFields := map[string]interface{}{}
	topics, ok := sesv2ContactListTopics(output.Topics)
	if !ok {
		return terraformutils.Resource{}, false
	}
	if len(topics) > 0 {
		additionalFields["topic"] = topics
	}
	return terraformutils.NewResource(
		sesv2ContactListImportID(contactListName),
		sesv2ResourceName("contact_list", contactListName),
		sesv2ContactListResourceType,
		"aws",
		attributes,
		sesv2AllowEmptyValues,
		additionalFields,
	), true
}

func sesv2ContactListTopics(topics []sesv2types.Topic) ([]interface{}, bool) {
	if len(topics) == 0 {
		return nil, true
	}
	sortedTopics := append([]sesv2types.Topic(nil), topics...)
	sort.SliceStable(sortedTopics, func(i, j int) bool {
		return sesv2TopicSortKey(sortedTopics[i]) < sesv2TopicSortKey(sortedTopics[j])
	})
	values := make([]interface{}, 0, len(sortedTopics))
	for _, topic := range sortedTopics {
		topicName := StringValue(topic.TopicName)
		displayName := StringValue(topic.DisplayName)
		defaultSubscriptionStatus := string(topic.DefaultSubscriptionStatus)
		if topicName == "" || displayName == "" || defaultSubscriptionStatus == "" {
			return nil, false
		}
		value := map[string]interface{}{
			"default_subscription_status": defaultSubscriptionStatus,
			"display_name":                displayName,
			"topic_name":                  topicName,
		}
		if topic.Description != nil {
			value["description"] = StringValue(topic.Description)
		}
		values = append(values, value)
	}
	return values, true
}

func sesv2TopicSortKey(topic sesv2types.Topic) string {
	return strings.Join([]string{
		StringValue(topic.TopicName),
		StringValue(topic.DisplayName),
		string(topic.DefaultSubscriptionStatus),
		StringValue(topic.Description),
	}, "\x00")
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

func newSESV2EmailIdentityFeedbackAttributesResource(identityName string, output *sesv2.GetEmailIdentityOutput) (terraformutils.Resource, bool) {
	if identityName == "" || output == nil {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2EmailIdentityFeedbackAttributesImportID(identityName),
		sesv2ResourceName("email_identity_feedback_attributes", identityName),
		sesv2EmailIdentityFeedbackAttributesResourceType,
		"aws",
		map[string]string{
			"email_identity":           identityName,
			"email_forwarding_enabled": strconv.FormatBool(output.FeedbackForwardingStatus),
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSESV2EmailIdentityMailFromAttributesResource(identityName string, output *sesv2.GetEmailIdentityOutput) (terraformutils.Resource, bool) {
	if identityName == "" || output == nil || output.MailFromAttributes == nil {
		return terraformutils.Resource{}, false
	}
	mailFromDomain := StringValue(output.MailFromAttributes.MailFromDomain)
	behaviorOnMXFailure := string(output.MailFromAttributes.BehaviorOnMxFailure)
	if mailFromDomain == "" || behaviorOnMXFailure == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2EmailIdentityMailFromAttributesImportID(identityName),
		sesv2ResourceName("email_identity_mail_from_attributes", identityName),
		sesv2EmailIdentityMailFromAttributesResourceType,
		"aws",
		map[string]string{
			"behavior_on_mx_failure": behaviorOnMXFailure,
			"email_identity":         identityName,
			"mail_from_domain":       mailFromDomain,
		},
		sesv2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newSESV2EmailIdentityPolicyResource(identityName, policyName, policy string) (terraformutils.Resource, bool) {
	if identityName == "" || policyName == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		sesv2EmailIdentityPolicyImportID(identityName, policyName),
		sesv2ResourceName("email_identity_policy", identityName, policyName),
		sesv2EmailIdentityPolicyResourceType,
		"aws",
		map[string]string{
			"email_identity": identityName,
			"policy":         policy,
			"policy_name":    policyName,
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

func sesv2ConfigurationSetEventDestinationImportID(configurationSetName, eventDestinationName string) string {
	return strings.Join([]string{configurationSetName, eventDestinationName}, sesv2ResourceIDSeparator)
}

func sesv2ContactListImportID(contactListName string) string {
	return contactListName
}

func sesv2DedicatedIPPoolImportID(poolName string) string {
	return poolName
}

func sesv2EmailIdentityImportID(identityName string) string {
	return identityName
}

func sesv2EmailIdentityFeedbackAttributesImportID(identityName string) string {
	return identityName
}

func sesv2EmailIdentityMailFromAttributesImportID(identityName string) string {
	return identityName
}

func sesv2EmailIdentityPolicyImportID(identityName, policyName string) string {
	return strings.Join([]string{identityName, policyName}, sesv2ResourceIDSeparator)
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
