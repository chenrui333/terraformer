// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/chimesdkvoice"
	chimesdkvoicetypes "github.com/aws/aws-sdk-go-v2/service/chimesdkvoice/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	chimeSDKVoiceGlobalSettingsResourceType        = "aws_chimesdkvoice_global_settings"
	chimeSDKVoiceSIPMediaApplicationResourceType   = "aws_chimesdkvoice_sip_media_application"
	chimeSDKVoiceSIPRuleResourceType               = "aws_chimesdkvoice_sip_rule"
	chimeSDKVoiceVoiceProfileDomainResourceType    = "aws_chimesdkvoice_voice_profile_domain"
	chimeSDKVoiceAWSRegionAttribute                = "aws" + "_region"
	chimeSDKVoiceGlobalSettingsResourceName        = "global_settings"
	chimeSDKVoiceGlobalSettingsMissingAccountIDLog = "Skipping Chime SDK Voice global settings: unable to get account ID: %v"
)

var chimeSDKVoiceAllowEmptyValues = []string{"tags."}

type ChimeSDKVoiceGenerator struct {
	AWSService
}

func (g *ChimeSDKVoiceGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := chimesdkvoice.NewFromConfig(config)
	if chimeSDKVoiceShouldLoadGlobalSettings(g.GetArgs()["region"].(string)) {
		if err := g.loadGlobalSettings(svc, config); err != nil {
			return err
		}
	}
	if err := g.loadSIPMediaApplications(svc); err != nil {
		return err
	}
	if err := g.loadSIPRules(svc); err != nil {
		return err
	}
	return g.loadVoiceProfileDomains(svc)
}

func (g *ChimeSDKVoiceGenerator) loadGlobalSettings(svc *chimesdkvoice.Client, config aws.Config) error {
	output, err := svc.GetGlobalSettings(context.TODO(), &chimesdkvoice.GetGlobalSettingsInput{})
	if err != nil {
		if chimeNotFound(err) {
			return nil
		}
		return err
	}
	accountID, err := g.getAccountNumber(config)
	if err != nil {
		log.Printf(chimeSDKVoiceGlobalSettingsMissingAccountIDLog, err)
		return nil
	}
	if resource, ok := newChimeSDKVoiceGlobalSettingsResource(StringValue(accountID), output); ok {
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *ChimeSDKVoiceGenerator) loadSIPMediaApplications(svc *chimesdkvoice.Client) error {
	input := &chimesdkvoice.ListSipMediaApplicationsInput{}
	for {
		page, err := svc.ListSipMediaApplications(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, application := range page.SipMediaApplications {
			applicationID := StringValue(application.SipMediaApplicationId)
			if applicationID == "" {
				continue
			}
			output, err := svc.GetSipMediaApplication(context.TODO(), &chimesdkvoice.GetSipMediaApplicationInput{
				SipMediaApplicationId: &applicationID,
			})
			if err != nil {
				if chimeNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newChimeSDKVoiceSIPMediaApplicationResource(output); ok {
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

func (g *ChimeSDKVoiceGenerator) loadSIPRules(svc *chimesdkvoice.Client) error {
	input := &chimesdkvoice.ListSipRulesInput{}
	for {
		page, err := svc.ListSipRules(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, rule := range page.SipRules {
			ruleID := StringValue(rule.SipRuleId)
			if ruleID == "" {
				continue
			}
			output, err := svc.GetSipRule(context.TODO(), &chimesdkvoice.GetSipRuleInput{
				SipRuleId: &ruleID,
			})
			if err != nil {
				if chimeNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newChimeSDKVoiceSIPRuleResource(output); ok {
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

func (g *ChimeSDKVoiceGenerator) loadVoiceProfileDomains(svc *chimesdkvoice.Client) error {
	input := &chimesdkvoice.ListVoiceProfileDomainsInput{}
	for {
		page, err := svc.ListVoiceProfileDomains(context.TODO(), input)
		if err != nil {
			return err
		}
		for _, domain := range page.VoiceProfileDomains {
			domainID := StringValue(domain.VoiceProfileDomainId)
			if domainID == "" {
				continue
			}
			output, err := svc.GetVoiceProfileDomain(context.TODO(), &chimesdkvoice.GetVoiceProfileDomainInput{
				VoiceProfileDomainId: &domainID,
			})
			if err != nil {
				if chimeNotFound(err) {
					continue
				}
				return err
			}
			if resource, ok := newChimeSDKVoiceVoiceProfileDomainResource(output); ok {
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

func newChimeSDKVoiceGlobalSettingsResource(accountID string, output *chimesdkvoice.GetGlobalSettingsOutput) (terraformutils.Resource, bool) {
	if accountID == "" || output == nil || output.VoiceConnector == nil || StringValue(output.VoiceConnector.CdrBucket) == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		chimeSDKVoiceGlobalSettingsImportID(accountID),
		chimeSDKVoiceResourceName(chimeSDKVoiceGlobalSettingsResourceName, accountID),
		chimeSDKVoiceGlobalSettingsResourceType,
		"aws",
		map[string]string{},
		chimeSDKVoiceAllowEmptyValues,
		map[string]interface{}{
			"voice_connector": []interface{}{map[string]interface{}{
				"cdr_bucket": StringValue(output.VoiceConnector.CdrBucket),
			}},
		},
	), true
}

func newChimeSDKVoiceSIPMediaApplicationResource(output *chimesdkvoice.GetSipMediaApplicationOutput) (terraformutils.Resource, bool) {
	if output == nil || output.SipMediaApplication == nil {
		return terraformutils.Resource{}, false
	}
	application := output.SipMediaApplication
	applicationID := StringValue(application.SipMediaApplicationId)
	name := StringValue(application.Name)
	awsRegion := StringValue(application.AwsRegion)
	endpoints := chimeSDKVoiceSIPMediaApplicationEndpointFields(application.Endpoints)
	if applicationID == "" || name == "" || awsRegion == "" || len(endpoints) == 0 {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		chimeSDKVoiceSIPMediaApplicationImportID(applicationID),
		chimeSDKVoiceResourceName("sip_media_application", name, applicationID),
		chimeSDKVoiceSIPMediaApplicationResourceType,
		"aws",
		map[string]string{
			"arn":                           StringValue(application.SipMediaApplicationArn),
			chimeSDKVoiceAWSRegionAttribute: awsRegion,
			"name":                          name,
		},
		chimeSDKVoiceAllowEmptyValues,
		map[string]interface{}{"endpoints": endpoints},
	), true
}

func newChimeSDKVoiceSIPRuleResource(output *chimesdkvoice.GetSipRuleOutput) (terraformutils.Resource, bool) {
	if output == nil || output.SipRule == nil {
		return terraformutils.Resource{}, false
	}
	rule := output.SipRule
	ruleID := StringValue(rule.SipRuleId)
	name := StringValue(rule.Name)
	triggerValue := StringValue(rule.TriggerValue)
	targetApplications := chimeSDKVoiceSIPRuleTargetApplicationFields(rule.TargetApplications)
	if ruleID == "" || name == "" || rule.TriggerType == "" || triggerValue == "" || len(targetApplications) == 0 {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"name":          name,
		"trigger_type":  string(rule.TriggerType),
		"trigger_value": triggerValue,
	}
	if rule.Disabled != nil {
		attributes["disabled"] = strconv.FormatBool(*rule.Disabled)
	}
	return terraformutils.NewResource(
		chimeSDKVoiceSIPRuleImportID(ruleID),
		chimeSDKVoiceResourceName("sip_rule", name, ruleID),
		chimeSDKVoiceSIPRuleResourceType,
		"aws",
		attributes,
		chimeSDKVoiceAllowEmptyValues,
		map[string]interface{}{"target_applications": targetApplications},
	), true
}

func newChimeSDKVoiceVoiceProfileDomainResource(output *chimesdkvoice.GetVoiceProfileDomainOutput) (terraformutils.Resource, bool) {
	if output == nil || output.VoiceProfileDomain == nil {
		return terraformutils.Resource{}, false
	}
	domain := output.VoiceProfileDomain
	domainID := StringValue(domain.VoiceProfileDomainId)
	name := StringValue(domain.Name)
	if domainID == "" || name == "" || domain.ServerSideEncryptionConfiguration == nil || StringValue(domain.ServerSideEncryptionConfiguration.KmsKeyArn) == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"arn":  StringValue(domain.VoiceProfileDomainArn),
		"id":   domainID,
		"name": name,
	}
	if description := StringValue(domain.Description); description != "" {
		attributes["description"] = description
	}
	return terraformutils.NewResource(
		chimeSDKVoiceVoiceProfileDomainImportID(domainID),
		chimeSDKVoiceResourceName("voice_profile_domain", name, domainID),
		chimeSDKVoiceVoiceProfileDomainResourceType,
		"aws",
		attributes,
		chimeSDKVoiceAllowEmptyValues,
		map[string]interface{}{
			"server_side_encryption_configuration": []interface{}{map[string]interface{}{
				"kms_key_arn": StringValue(domain.ServerSideEncryptionConfiguration.KmsKeyArn),
			}},
		},
	), true
}

func chimeSDKVoiceSIPMediaApplicationEndpointFields(endpoints []chimesdkvoicetypes.SipMediaApplicationEndpoint) []interface{} {
	result := make([]interface{}, 0, len(endpoints))
	for _, endpoint := range endpoints {
		lambdaARN := StringValue(endpoint.LambdaArn)
		if lambdaARN == "" {
			continue
		}
		result = append(result, map[string]interface{}{"lambda_arn": lambdaARN})
	}
	return result
}

func chimeSDKVoiceSIPRuleTargetApplicationFields(applications []chimesdkvoicetypes.SipRuleTargetApplication) []interface{} {
	result := make([]interface{}, 0, len(applications))
	for _, application := range applications {
		awsRegion := StringValue(application.AwsRegion)
		applicationID := StringValue(application.SipMediaApplicationId)
		if awsRegion == "" || applicationID == "" || application.Priority == nil {
			continue
		}
		result = append(result, map[string]interface{}{
			chimeSDKVoiceAWSRegionAttribute: awsRegion,
			"priority":                      int(*application.Priority),
			"sip_media_application_id":      applicationID,
		})
	}
	return result
}

func chimeSDKVoiceGlobalSettingsImportID(accountID string) string {
	return accountID
}

func chimeSDKVoiceSIPMediaApplicationImportID(applicationID string) string {
	return applicationID
}

func chimeSDKVoiceSIPRuleImportID(ruleID string) string {
	return ruleID
}

func chimeSDKVoiceVoiceProfileDomainImportID(domainID string) string {
	return domainID
}

func chimeSDKVoiceShouldLoadGlobalSettings(region string) bool {
	return region == NoRegion || region == MainRegionPublicPartition
}

func chimeSDKVoiceResourceName(parts ...string) string {
	return resourceNameWithLengthPrefixes(parts...)
}
