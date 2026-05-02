// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package ibm

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	bluemix "github.com/IBM-Cloud/bluemix-go"
	"github.com/chenrui333/terraformer/terraformutils"

	ns "github.com/IBM-Cloud/bluemix-go/api/functions"
	"github.com/IBM-Cloud/bluemix-go/session"

	"github.com/apache/openwhisk-client-go/whisk"
)

// CloudFunctionGenerator ..
type CloudFunctionGenerator struct {
	IBMService
}

func (g CloudFunctionGenerator) loadPackages(namespace, pkgName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s:%s", namespace, pkgName),
		normalizeResourceName(fmt.Sprintf("%s_%s", namespace, pkgName), false),
		"ibm_function_package",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})
	return resource
}

func (g CloudFunctionGenerator) loadRules(namespace, ruleName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s:%s", namespace, ruleName),
		normalizeResourceName(ruleName, true),
		"ibm_function_rule",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})
	return resource
}

func (g CloudFunctionGenerator) loadTriggers(namespace, triggerName string) terraformutils.Resource {
	resource := terraformutils.NewResource(
		fmt.Sprintf("%s:%s", namespace, triggerName),
		normalizeResourceName(triggerName, true),
		"ibm_function_trigger",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{})
	return resource
}

/*
 *
 * Configure a HTTP client using the OpenWhisk properties (i.e. host, auth, iamtoken)
 * Only cf-based namespaces needs auth key value.
 * iam-based namespace don't have an auth key and needs only iam token for authorization.
 *
 */
func setupOpenWhiskClientConfigIAM(response ns.NamespaceResponse, c *bluemix.Config, region string) (*whisk.Client, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s.functions.cloud.ibm.com/api", region))
	if err != nil {
		return nil, err
	}
	wskClient, err := whisk.NewClient(http.DefaultClient, &whisk.Config{
		Host:    u.Host,
		Version: "v1",
	})
	if err != nil {
		return nil, err
	}

	if os.Getenv("TF_LOG") != "" {
		whisk.SetDebug(true)
	}

	// Configure whisk properties to handle iam-based/iam-migrated  namespaces.
	if response.IsIamEnabled() {
		additionalHeaders := make(http.Header)
		additionalHeaders.Add("Authorization", c.IAMAccessToken)
		additionalHeaders.Add("X-Namespace-Id", response.GetID())

		wskClient.Namespace = response.GetID()
		wskClient.AdditionalHeaders = additionalHeaders
		return wskClient, nil
	}

	return nil, fmt.Errorf("Failed to create whisk config object for IAM based namespace '%v'", response.GetName())
}

// InitResources ..
func (g *CloudFunctionGenerator) InitResources() error {
	region := g.Args["region"].(string)
	bmxConfig := &bluemix.Config{
		BluemixAPIKey: os.Getenv("IC_API_KEY"),
	}

	bmxConfig.Region = region

	sess, err := session.New(bmxConfig)
	if err != nil {
		return err
	}

	err = authenticateAPIKey(sess)
	if err != nil {
		return err
	}

	err = authenticateCF(sess)
	if err != nil {
		return err
	}

	nsClient, err := ns.New(sess)
	if err != nil {
		return err
	}

	nsList, err := nsClient.Namespaces().GetNamespaces()
	if err != nil {
		return err
	}

	for _, n := range nsList.Namespaces {
		// Namespace
		if !n.IsIamEnabled() {
			continue
		}

		// Build whisk object
		wskClient, err := setupOpenWhiskClientConfigIAM(n, sess.Config, region)
		if err != nil {
			return err
		}

		// Package
		packageService := wskClient.Packages
		pkgOptions := &whisk.PackageListOptions{
			Limit: 100,
			Skip:  0,
		}
		pkgs, pkgResp, err := packageService.List(pkgOptions)
		if pkgResp != nil && pkgResp.Body != nil {
			defer pkgResp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error retrieving IBM Cloud Function package: %w", err)
		}

		for _, p := range pkgs {
			g.Resources = append(g.Resources, g.loadPackages(n.GetName(), p.GetName()))
		}

		// Action
		actionService := wskClient.Actions
		actionOptions := &whisk.ActionListOptions{
			Limit: 100,
			Skip:  0,
		}
		actions, actionResp, err := actionService.List("", actionOptions)
		if actionResp != nil && actionResp.Body != nil {
			defer actionResp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error retrieving IBM Cloud Function action: %w", err)
		}

		for _, a := range actions {
			actionID := ""
			parts := strings.Split(a.Namespace, "/")
			if len(parts) == 2 {
				var pkgDependsOn []string
				pkgDependsOn = append(pkgDependsOn,
					"ibm_function_package."+terraformutils.TfSanitize(fmt.Sprintf("%s_%s", n.GetName(), parts[1])))
				actionID = fmt.Sprintf("%s/%s", parts[1], a.Name)
				g.Resources = append(g.Resources, terraformutils.NewResource(
					fmt.Sprintf("%s:%s", n.GetName(), actionID),
					normalizeResourceName(a.Name, true),
					"ibm_function_action",
					"ibm",
					map[string]string{},
					[]string{},
					map[string]interface{}{
						"depends_on": pkgDependsOn,
					}))
			} else {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					fmt.Sprintf("%s:%s", n.GetName(), a.Name),
					normalizeResourceName(a.Name, true),
					"ibm_function_action",
					"ibm",
					map[string]string{},
					[]string{},
					map[string]interface{}{}))
			}
		}

		// Rule
		ruleService := wskClient.Rules
		ruleOptions := &whisk.RuleListOptions{
			Limit: 100,
			Skip:  0,
		}
		rules, ruleResp, err := ruleService.List(ruleOptions)
		if ruleResp != nil && ruleResp.Body != nil {
			defer ruleResp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error retrieving IBM Cloud Function rule: %w", err)
		}

		for _, r := range rules {
			g.Resources = append(g.Resources, g.loadRules(n.GetName(), r.Name))
		}

		// Triggers
		triggerService := wskClient.Triggers
		triggerOptions := &whisk.TriggerListOptions{
			Limit: 100,
			Skip:  0,
		}
		triggers, triggerResp, err := triggerService.List(triggerOptions)
		if triggerResp != nil && triggerResp.Body != nil {
			defer triggerResp.Body.Close()
		}
		if err != nil {
			return fmt.Errorf("error retrieving IBM Cloud Function trigger: %w", err)
		}

		for _, t := range triggers {
			g.Resources = append(g.Resources, g.loadTriggers(n.GetName(), t.Name))
		}
	}

	return nil
}

func (g *CloudFunctionGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "ibm_function_action" {
			continue
		}
		for _, ri := range g.Resources {
			if ri.InstanceInfo.Type != "ibm_function_package" {
				continue
			}
			if len(strings.Split(r.InstanceState.Attributes["id"], "/")) == 2 {
				if strings.Split(r.InstanceState.Attributes["id"], "/")[0] == ri.InstanceState.Attributes["id"] {
					g.Resources[i].Item["name"] = "${ibm_function_package." + ri.ResourceName + ".name}" + "/" + r.InstanceState.Attributes["action_id"]
				}
			}
		}
	}
	return nil
}
