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
	// OnCallUserNotificationChannelAllowEmptyValues ...
	OnCallUserNotificationChannelAllowEmptyValues = []string{"email.formats"}
)

// OnCallUserNotificationChannelGenerator ...
type OnCallUserNotificationChannelGenerator struct {
	DatadogService
}

func (g *OnCallUserNotificationChannelGenerator) createResources(userID string, userNotificationChannels []datadogV2.NotificationChannelData) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	for _, userNotificationChannel := range userNotificationChannels {
		resource, skipped, err := g.createResource(userID, userNotificationChannel)
		if err != nil {
			return nil, err
		}
		if skipped {
			continue
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (g *OnCallUserNotificationChannelGenerator) createResource(userID string, userNotificationChannel datadogV2.NotificationChannelData) (terraformutils.Resource, bool, error) {
	notificationChannelID := userNotificationChannel.GetId()
	if notificationChannelID == "" {
		return terraformutils.Resource{}, false, fmt.Errorf("On-Call user notification channel missing id")
	}
	if userID == "" {
		return terraformutils.Resource{}, false, fmt.Errorf("On-Call user notification channel %q missing user id", notificationChannelID)
	}
	if !isSupportedOnCallUserNotificationChannel(userNotificationChannel) {
		return terraformutils.Resource{}, true, nil
	}

	return terraformutils.NewResource(
		notificationChannelID,
		fmt.Sprintf("on_call_user_notification_channel_%s_%s", userID, notificationChannelID),
		"datadog_on_call_user_notification_channel",
		"datadog",
		map[string]string{
			"user_id": userID,
		},
		OnCallUserNotificationChannelAllowEmptyValues,
		map[string]interface{}{},
	), false, nil
}

// InitResources Generate TerraformResources from Datadog API,
// from each On-Call user notification channel create 1 TerraformResource.
// Need On-Call User Notification Channel ID formatted as '<user_id>:<channel_id>' for filter lookup.
func (g *OnCallUserNotificationChannelGenerator) InitResources() error {
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
		userNotificationChannels, err := listOnCallUserNotificationChannels(auth, onCallAPI, userID)
		if err != nil {
			return err
		}
		userResources, err := g.createResources(userID, userNotificationChannels)
		if err != nil {
			return err
		}
		resources = append(resources, userResources...)
	}

	g.Resources = resources
	return nil
}

func (g *OnCallUserNotificationChannelGenerator) filteredResources(auth context.Context, api *datadogV2.OnCallApi) ([]terraformutils.Resource, bool, error) {
	resources := []terraformutils.Resource{}
	filtered := false

	for filterIndex, filter := range g.Filter {
		if !filter.IsApplicable("on_call_user_notification_channel") {
			continue
		}

		switch filter.FieldPath {
		case "id":
			filtered = true
			filterIDs, err := parseOnCallUserChildImportIDs(filter.AcceptableValues, "channel")
			if err != nil {
				return nil, true, err
			}
			for _, filterID := range filterIDs {
				userNotificationChannel, err := getOnCallUserNotificationChannel(auth, api, filterID.userID, filterID.childID)
				if err != nil {
					return nil, true, err
				}
				resource, skipped, err := g.createResource(filterID.userID, userNotificationChannel)
				if err != nil {
					return nil, true, err
				}
				if skipped {
					continue
				}
				resources = append(resources, resource)
			}
			g.Filter[filterIndex].AcceptableValues = onCallUserChildIDs(filterIDs)
		case "user_id":
			filtered = true
			for _, userID := range filter.AcceptableValues {
				userNotificationChannels, err := listOnCallUserNotificationChannels(auth, api, userID)
				if err != nil {
					return nil, true, err
				}
				userResources, err := g.createResources(userID, userNotificationChannels)
				if err != nil {
					return nil, true, err
				}
				resources = append(resources, userResources...)
			}
		}
	}

	return resources, filtered, nil
}

func getOnCallUserNotificationChannel(auth context.Context, api *datadogV2.OnCallApi, userID string, channelID string) (datadogV2.NotificationChannelData, error) {
	userNotificationChannel, httpResp, err := api.GetUserNotificationChannel(auth, userID, channelID)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		return datadogV2.NotificationChannelData{}, err
	}

	data := userNotificationChannel.GetData()
	if data.GetId() == "" {
		data.SetId(channelID)
	}
	return data, nil
}

func listOnCallUserNotificationChannels(auth context.Context, api *datadogV2.OnCallApi, userID string) ([]datadogV2.NotificationChannelData, error) {
	userNotificationChannels, httpResp, err := api.ListUserNotificationChannels(auth, userID)
	if httpResp != nil && httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
			return []datadogV2.NotificationChannelData{}, nil
		}
		return nil, err
	}
	return userNotificationChannels.GetData(), nil
}

func isSupportedOnCallUserNotificationChannel(userNotificationChannel datadogV2.NotificationChannelData) bool {
	attributes := userNotificationChannel.GetAttributes()
	config, ok := attributes.GetConfigOk()
	if !ok {
		return false
	}

	switch config.GetActualInstance().(type) {
	case *datadogV2.NotificationChannelEmailConfig, *datadogV2.NotificationChannelPhoneConfig:
		return true
	default:
		return false
	}
}
