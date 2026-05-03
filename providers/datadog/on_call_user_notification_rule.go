// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	// OnCallUserNotificationRuleAllowEmptyValues ...
	OnCallUserNotificationRuleAllowEmptyValues = []string{}
)

// OnCallUserNotificationRuleGenerator ...
type OnCallUserNotificationRuleGenerator struct {
	DatadogService
}

func (g *OnCallUserNotificationRuleGenerator) createResources(userID string, userNotificationRules []datadogV2.OnCallNotificationRuleData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, userNotificationRule := range userNotificationRules {
		resource, err := g.createResource(userID, userNotificationRule)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *OnCallUserNotificationRuleGenerator) createResource(userID string, userNotificationRule datadogV2.OnCallNotificationRuleData) (terraformutils.Resource, error) {
	notificationRuleID := userNotificationRule.GetId()
	if notificationRuleID == "" {
		return terraformutils.Resource{}, fmt.Errorf("On-Call user notification rule missing id")
	}
	if userID == "" {
		return terraformutils.Resource{}, fmt.Errorf("On-Call user notification rule %q missing user id", notificationRuleID)
	}

	return terraformutils.NewResource(
		onCallUserChildProviderImportID(userID, notificationRuleID),
		fmt.Sprintf("on_call_user_notification_rule_%s_%s", userID, notificationRuleID),
		"datadog_on_call_user_notification_rule",
		"datadog",
		map[string]string{
			"user_id": userID,
		},
		OnCallUserNotificationRuleAllowEmptyValues,
		map[string]interface{}{},
	), nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each On-Call user notification rule create 1 TerraformResource.
// Need On-Call User Notification Rule ID formatted as '<user_id>,<rule_id>' for provider import.
func (g *OnCallUserNotificationRuleGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	onCallAPI := datadogV2.NewOnCallApi(datadogClient)
	usersAPI := datadogV2.NewUsersApi(datadogClient)

	resources, filtered, err := g.filteredResources(auth, onCallAPI)
	if err != nil {
		return err
	}
	if filtered {
		g.Resources = resources
		return nil
	}

	userIDs, err := listDatadogUserIDs(auth, usersAPI)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		userNotificationRules, err := listOnCallUserNotificationRules(auth, onCallAPI, userID)
		if err != nil {
			return err
		}
		userResources, err := g.createResources(userID, userNotificationRules)
		if err != nil {
			return err
		}
		resources = append(resources, userResources...)
	}

	g.Resources = resources
	return nil
}

func (g *OnCallUserNotificationRuleGenerator) filteredResources(auth context.Context, api *datadogV2.OnCallApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("on_call_user_notification_rule") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseOnCallUserChildImportIDs(filter.AcceptableValues, "rule")
			if err != nil {
				return nil, true, err
			}
			for _, filterID := range filterIDs {
				userNotificationRule, err := getOnCallUserNotificationRule(auth, api, filterID.userID, filterID.childID)
				if err != nil {
					return nil, true, err
				}
				resource, err := g.createResource(filterID.userID, userNotificationRule)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = onCallUserChildProviderImportIDs(filterIDs)
		case "user_id":
			filtered = true
			for _, userID := range filter.AcceptableValues {
				userNotificationRules, err := listOnCallUserNotificationRules(auth, api, userID)
				if err != nil {
					return nil, true, err
				}
				userResources, err := g.createResources(userID, userNotificationRules)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, userResources...)
			}
		}
	}

	return resources, filtered, nil
}

func getOnCallUserNotificationRule(auth context.Context, api *datadogV2.OnCallApi, userID string, ruleID string) (datadogV2.OnCallNotificationRuleData, error) {
	userNotificationRule, httpResp, err := api.GetUserNotificationRule(auth, userID, ruleID)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		return datadogV2.OnCallNotificationRuleData{}, err
	}

	data := userNotificationRule.GetData()
	if data.GetId() == "" {
		data.SetId(ruleID)
	}
	return data, nil
}

func listOnCallUserNotificationRules(auth context.Context, api *datadogV2.OnCallApi, userID string) ([]datadogV2.OnCallNotificationRuleData, error) {
	include := "channel"
	userNotificationRules, httpResp, err := api.ListUserNotificationRules(auth, userID, datadogV2.ListUserNotificationRulesOptionalParameters{Include: &include})
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return []datadogV2.OnCallNotificationRuleData{}, nil
		}
		return nil, err
	}
	return userNotificationRules.GetData(), nil
}
