// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	pinpointsmsvoicev2types "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	pinpointSMSVoiceV2ConfigurationSetResourceType = "aws_pinpointsmsvoicev2_configuration_set"
	pinpointSMSVoiceV2OptOutListResourceType       = "aws_pinpointsmsvoicev2_opt_out_list"
	pinpointSMSVoiceV2PhoneNumberResourceType      = "aws_pinpointsmsvoicev2_phone_number"
)

var pinpointSMSVoiceV2AllowEmptyValues = []string{"tags."}

type PinpointSMSVoiceV2Generator struct {
	AWSService
}

func (g *PinpointSMSVoiceV2Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := pinpointsmsvoicev2.NewFromConfig(config)

	if err := g.loadConfigurationSets(svc); err != nil {
		return err
	}
	if err := g.loadOptOutLists(svc); err != nil {
		return err
	}
	if err := g.loadPhoneNumbers(svc); err != nil {
		return err
	}
	return nil
}

func (g *PinpointSMSVoiceV2Generator) loadConfigurationSets(svc *pinpointsmsvoicev2.Client) error {
	p := pinpointsmsvoicev2.NewDescribeConfigurationSetsPaginator(svc, &pinpointsmsvoicev2.DescribeConfigurationSetsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, configurationSet := range page.ConfigurationSets {
			if resource, ok := newPinpointSMSVoiceV2ConfigurationSetResource(configurationSet); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *PinpointSMSVoiceV2Generator) loadOptOutLists(svc *pinpointsmsvoicev2.Client) error {
	p := pinpointsmsvoicev2.NewDescribeOptOutListsPaginator(svc, &pinpointsmsvoicev2.DescribeOptOutListsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, optOutList := range page.OptOutLists {
			if resource, ok := newPinpointSMSVoiceV2OptOutListResource(optOutList); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *PinpointSMSVoiceV2Generator) loadPhoneNumbers(svc *pinpointsmsvoicev2.Client) error {
	p := pinpointsmsvoicev2.NewDescribePhoneNumbersPaginator(svc, &pinpointsmsvoicev2.DescribePhoneNumbersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, phoneNumber := range page.PhoneNumbers {
			if resource, ok := newPinpointSMSVoiceV2PhoneNumberResource(phoneNumber); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newPinpointSMSVoiceV2ConfigurationSetResource(configurationSet pinpointsmsvoicev2types.ConfigurationSetInformation) (terraformutils.Resource, bool) {
	name := StringValue(configurationSet.ConfigurationSetName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"id":   name,
		"name": name,
	}
	if arn := StringValue(configurationSet.ConfigurationSetArn); arn != "" {
		attributes["arn"] = arn
	}
	return terraformutils.NewResource(
		pinpointSMSVoiceV2NameImportID(name),
		pinpointSMSVoiceV2ResourceName("configuration_set", name),
		pinpointSMSVoiceV2ConfigurationSetResourceType,
		"aws",
		attributes,
		pinpointSMSVoiceV2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newPinpointSMSVoiceV2OptOutListResource(optOutList pinpointsmsvoicev2types.OptOutListInformation) (terraformutils.Resource, bool) {
	name := StringValue(optOutList.OptOutListName)
	if name == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"id":   name,
		"name": name,
	}
	if arn := StringValue(optOutList.OptOutListArn); arn != "" {
		attributes["arn"] = arn
	}
	return terraformutils.NewResource(
		pinpointSMSVoiceV2NameImportID(name),
		pinpointSMSVoiceV2ResourceName("opt_out_list", name),
		pinpointSMSVoiceV2OptOutListResourceType,
		"aws",
		attributes,
		pinpointSMSVoiceV2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newPinpointSMSVoiceV2PhoneNumberResource(phoneNumber pinpointsmsvoicev2types.PhoneNumberInformation) (terraformutils.Resource, bool) {
	phoneNumberID := StringValue(phoneNumber.PhoneNumberId)
	if phoneNumberID == "" ||
		StringValue(phoneNumber.IsoCountryCode) == "" ||
		phoneNumber.MessageType == "" ||
		len(phoneNumber.NumberCapabilities) == 0 ||
		phoneNumber.NumberType == "" ||
		StringValue(phoneNumber.PhoneNumber) == "" ||
		!pinpointSMSVoiceV2PhoneNumberImportable(phoneNumber) {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"id":               phoneNumberID,
		"iso_country_code": StringValue(phoneNumber.IsoCountryCode),
		"message_type":     string(phoneNumber.MessageType),
		"number_type":      string(phoneNumber.NumberType),
		"phone_number":     StringValue(phoneNumber.PhoneNumber),
	}
	if arn := StringValue(phoneNumber.PhoneNumberArn); arn != "" {
		attributes["arn"] = arn
	}
	if optOutListName := StringValue(phoneNumber.OptOutListName); optOutListName != "" {
		attributes["opt_out_list_name"] = optOutListName
	}
	return terraformutils.NewResource(
		pinpointSMSVoiceV2PhoneNumberImportID(phoneNumberID),
		pinpointSMSVoiceV2ResourceName("phone_number", StringValue(phoneNumber.PhoneNumber), phoneNumberID),
		pinpointSMSVoiceV2PhoneNumberResourceType,
		"aws",
		attributes,
		pinpointSMSVoiceV2AllowEmptyValues,
		map[string]interface{}{},
	), true
}

func pinpointSMSVoiceV2NameImportID(name string) string {
	return name
}

func pinpointSMSVoiceV2PhoneNumberImportID(phoneNumberID string) string {
	return phoneNumberID
}

func pinpointSMSVoiceV2ResourceName(parts ...string) string {
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

func pinpointSMSVoiceV2PhoneNumberImportable(phoneNumber pinpointsmsvoicev2types.PhoneNumberInformation) bool {
	if phoneNumber.NumberType == pinpointsmsvoicev2types.NumberTypeShortCode {
		return false
	}
	switch phoneNumber.Status {
	case "", pinpointsmsvoicev2types.NumberStatusActive:
		return true
	default:
		return false
	}
}

func pinpointSMSVoiceV2NotFound(err error) bool {
	var notFound *pinpointsmsvoicev2types.ResourceNotFoundException
	return errors.As(err, &notFound)
}
