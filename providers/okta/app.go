// SPDX-License-Identifier: Apache-2.0

package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v6/okta"
)

// NOTE: The Okta SDK ListApplications() method does not support applications by type at this time. So
//
//	we have to create the application filter by our self.
type oktaApplicationSummary struct {
	ID         string
	Name       string
	Label      string
	SignOnMode string
}

func getApplications(ctx context.Context, client *okta.APIClient, signOnMode string) ([]okta.ListApplications200ResponseInner, error) {
	supportedApps, err := getAllApplications(ctx, client)
	if err != nil {
		return nil, err
	}

	var filterApps []okta.ListApplications200ResponseInner
	for _, app := range supportedApps {
		summary, ok := getApplicationSummary(app)
		if ok && summary.SignOnMode == signOnMode {
			filterApps = append(filterApps, app)
		}
	}
	return filterApps, nil
}

func getAllApplications(ctx context.Context, client *okta.APIClient) ([]okta.ListApplications200ResponseInner, error) {
	apps, resp, err := client.ApplicationAPI.ListApplications(ctx).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var nextAppSet []okta.ListApplications200ResponseInner
		resp, err = resp.Next(&nextAppSet)
		if err != nil {
			return nil, err
		}
		apps = append(apps, nextAppSet...)
	}

	var supportedApps []okta.ListApplications200ResponseInner
	for _, app := range apps {
		summary, ok := getApplicationSummary(app)
		if !ok {
			continue
		}
		//NOTE: Okta provider does not support the following app type/name
		if summary.Name == "template_wsfed" ||
			summary.Name == "template_swa_two_page" ||
			summary.Name == "okta_enduser" ||
			summary.Name == "okta_browser_plugin" ||
			summary.Name == "saasure" {
			continue
		}
		supportedApps = append(supportedApps, app)
	}

	return supportedApps, nil
}

func getApplicationSummary(app okta.ListApplications200ResponseInner) (oktaApplicationSummary, bool) {
	switch {
	case app.AutoLoginApplication != nil:
		return applicationSummaryFromFields(
			app.AutoLoginApplication.GetId(),
			app.AutoLoginApplication.GetName(),
			app.AutoLoginApplication.GetLabel(),
			app.AutoLoginApplication.GetSignOnMode(),
		)
	case app.BasicAuthApplication != nil:
		return applicationSummaryFromFields(
			app.BasicAuthApplication.GetId(),
			app.BasicAuthApplication.GetName(),
			app.BasicAuthApplication.GetLabel(),
			app.BasicAuthApplication.GetSignOnMode(),
		)
	case app.BookmarkApplication != nil:
		return applicationSummaryFromFields(
			app.BookmarkApplication.GetId(),
			app.BookmarkApplication.GetName(),
			app.BookmarkApplication.GetLabel(),
			app.BookmarkApplication.GetSignOnMode(),
		)
	case app.BrowserPluginApplication != nil:
		return applicationSummaryFromFields(
			app.BrowserPluginApplication.GetId(),
			app.BrowserPluginApplication.GetName(),
			app.BrowserPluginApplication.GetLabel(),
			app.BrowserPluginApplication.GetSignOnMode(),
		)
	case app.OpenIdConnectApplication != nil:
		return applicationSummaryFromFields(
			app.OpenIdConnectApplication.GetId(),
			app.OpenIdConnectApplication.GetName(),
			app.OpenIdConnectApplication.GetLabel(),
			app.OpenIdConnectApplication.GetSignOnMode(),
		)
	case app.Saml11Application != nil:
		return applicationSummaryFromFields(
			app.Saml11Application.GetId(),
			app.Saml11Application.GetName(),
			app.Saml11Application.GetLabel(),
			app.Saml11Application.GetSignOnMode(),
		)
	case app.SamlApplication != nil:
		return applicationSummaryFromFields(
			app.SamlApplication.GetId(),
			app.SamlApplication.GetName(),
			app.SamlApplication.GetLabel(),
			app.SamlApplication.GetSignOnMode(),
		)
	case app.SecurePasswordStoreApplication != nil:
		return applicationSummaryFromFields(
			app.SecurePasswordStoreApplication.GetId(),
			app.SecurePasswordStoreApplication.GetName(),
			app.SecurePasswordStoreApplication.GetLabel(),
			app.SecurePasswordStoreApplication.GetSignOnMode(),
		)
	case app.WsFederationApplication != nil:
		return applicationSummaryFromFields(
			app.WsFederationApplication.GetId(),
			app.WsFederationApplication.GetName(),
			app.WsFederationApplication.GetLabel(),
			app.WsFederationApplication.GetSignOnMode(),
		)
	default:
		return oktaApplicationSummary{}, false
	}
}

func applicationSummaryFromFields(id, name, label, signOnMode string) (oktaApplicationSummary, bool) {
	if id == "" {
		return oktaApplicationSummary{}, false
	}
	return oktaApplicationSummary{
		ID:         id,
		Name:       name,
		Label:      label,
		SignOnMode: signOnMode,
	}, true
}
