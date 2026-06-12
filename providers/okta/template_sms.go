// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/okta/okta-sdk-golang/v6/okta"
)

type SMSTemplateGenerator struct {
	OktaService
}

func (g SMSTemplateGenerator) createResources(smsTemplateList []okta.SmsTemplate) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, smsTemplate := range smsTemplateList {
		resources = append(resources, terraformutils.NewSimpleResource(
			smsTemplate.GetId(),
			"template_sms_"+smsTemplate.GetName(),
			"okta_template_sms",
			"okta",
			[]string{}))
	}
	return resources
}

func (g *SMSTemplateGenerator) InitResources() error {
	ctx, client, e := g.Client()
	if e != nil {
		return e
	}

	output, resp, err := client.TemplateAPI.ListSmsTemplates(ctx).Execute()
	if err != nil {
		return err
	}

	for resp.HasNextPage() {
		var nextSmsTemplateSet []okta.SmsTemplate
		resp, err = resp.Next(&nextSmsTemplateSet)
		if err != nil {
			return err
		}
		output = append(output, nextSmsTemplateSet...)
	}

	g.Resources = g.createResources(output)
	return nil
}
