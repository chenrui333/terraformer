// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	appconfigtypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAppConfigResourceIDsMatchProviderReadIDs(t *testing.T) {
	applicationID := "app123"
	configurationProfileID := "prof123"
	environmentID := "env123"

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "configuration profile is configuration profile then application",
			got:  appConfigConfigurationProfileResourceID(configurationProfileID, applicationID),
			want: "prof123:app123",
		},
		{
			name: "environment is environment then application",
			got:  appConfigEnvironmentResourceID(environmentID, applicationID),
			want: "env123:app123",
		},
		{
			name: "deployment is application environment deployment number",
			got:  appConfigDeploymentResourceID(applicationID, environmentID, 3),
			want: "app123/env123/3",
		},
		{
			name: "hosted configuration version is application profile version",
			got:  appConfigHostedConfigurationVersionResourceID(applicationID, configurationProfileID, 7),
			want: "app123/prof123/7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("resource ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestAppConfigResourceNamesIncludeIDs(t *testing.T) {
	application := newAppConfigApplicationResource("app123", "orders")
	otherApplication := newAppConfigApplicationResource("app456", "orders")
	if application.ResourceName == otherApplication.ResourceName {
		t.Fatalf("application resource names collide: %q", application.ResourceName)
	}

	profile := newAppConfigConfigurationProfileResource("app123", "prof123", "settings")
	otherProfile := newAppConfigConfigurationProfileResource("app456", "prof123", "settings")
	if profile.ResourceName == otherProfile.ResourceName {
		t.Fatalf("configuration profile resource names collide: %q", profile.ResourceName)
	}
}

func TestAppConfigResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "resource not found", err: &appconfigtypes.ResourceNotFoundException{}, want: true},
		{name: "wrapped resource not found", err: errors.Join(errors.New("boom"), &appconfigtypes.ResourceNotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appConfigResourceMissing(tt.err); got != tt.want {
				t.Fatalf("appConfigResourceMissing(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestAppConfigSkipsAWSManagedStandaloneResources(t *testing.T) {
	if appConfigDeploymentStrategyImportable("AppConfig.AllAtOnce") {
		t.Fatal("predefined deployment strategy should not be importable as a standalone resource")
	}
	if !appConfigDeploymentStrategyImportable("abc123") {
		t.Fatal("customer deployment strategy should be importable")
	}
	if appConfigExtensionImportable("AWS.AppConfig.DeploymentEventsToAmazonSNS", "AWS.AppConfig.DeploymentEventsToAmazonSNS") {
		t.Fatal("AWS-authored extension should not be importable as a standalone resource")
	}
	if !appConfigExtensionImportable("ext123", "notify") {
		t.Fatal("customer extension should be importable")
	}
}

func TestAppConfigFilterGatesApplicationAndChildDiscovery(t *testing.T) {
	applicationID := "app123"
	otherApplicationID := "app456"
	configurationProfileID := "prof123"
	otherConfigurationProfileID := "prof456"
	environmentID := "env123"
	otherEnvironmentID := "env456"
	application := newAppConfigApplicationResource(applicationID, "orders")
	otherApplication := newAppConfigApplicationResource(otherApplicationID, "other")
	configurationProfile := newAppConfigConfigurationProfileResource(applicationID, configurationProfileID, "settings")
	environment := newAppConfigEnvironmentResource(applicationID, environmentID, "prod")
	hostedVersion := newAppConfigHostedConfigurationVersionResource(applicationID, configurationProfileID, 7)
	deployment := newAppConfigDeploymentResource(applicationID, environmentID, 3)

	tests := []struct {
		name                  string
		filters               []terraformutils.ResourceFilter
		loadApplications      bool
		appendApplication     bool
		appendOther           bool
		loadChildren          bool
		loadOtherChildren     bool
		loadProfiles          bool
		loadOtherProfiles     bool
		loadHosted            bool
		loadOtherHosted       bool
		loadEnvironments      bool
		loadOtherEnvironments bool
		loadDeployments       bool
		loadOtherDeployments  bool
		appendProfile         bool
		appendEnvironment     bool
		appendHosted          bool
		appendDeployment      bool
	}{
		{
			name:                  "no filters imports applications and children",
			loadApplications:      true,
			appendApplication:     true,
			appendOther:           true,
			loadChildren:          true,
			loadOtherChildren:     true,
			loadProfiles:          true,
			loadOtherProfiles:     true,
			loadHosted:            true,
			loadOtherHosted:       true,
			loadEnvironments:      true,
			loadOtherEnvironments: true,
			loadDeployments:       true,
			loadOtherDeployments:  true,
			appendProfile:         true,
			appendEnvironment:     true,
			appendHosted:          true,
			appendDeployment:      true,
		},
		{
			name: "typed application id filter limits applications and children",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigApplicationResourceType, FieldPath: "id", AcceptableValues: []string{applicationID}},
			},
			loadApplications:     true,
			appendApplication:    true,
			loadChildren:         true,
			loadProfiles:         true,
			loadHosted:           true,
			loadOtherHosted:      true,
			loadEnvironments:     true,
			loadDeployments:      true,
			loadOtherDeployments: true,
			appendProfile:        true,
			appendEnvironment:    true,
			appendHosted:         true,
			appendDeployment:     true,
		},
		{
			name: "typed child id filter does not import parent applications",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigConfigurationProfileResourceType, FieldPath: "id", AcceptableValues: []string{appConfigConfigurationProfileResourceID(configurationProfileID, applicationID)}},
			},
			loadApplications: true,
			loadChildren:     true,
			loadProfiles:     true,
			appendProfile:    true,
		},
		{
			name: "typed parent and child id filters load matching child outside parent filter",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigApplicationResourceType, FieldPath: "id", AcceptableValues: []string{otherApplicationID}},
				{ServiceName: appConfigConfigurationProfileResourceType, FieldPath: "id", AcceptableValues: []string{appConfigConfigurationProfileResourceID(configurationProfileID, applicationID)}},
			},
			loadApplications:  true,
			appendOther:       true,
			loadChildren:      true,
			loadProfiles:      true,
			appendProfile:     true,
			loadOtherChildren: false,
		},
		{
			name: "typed hosted version id filter scopes profile version discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigHostedConfigurationVersionResourceType, FieldPath: "id", AcceptableValues: []string{appConfigHostedConfigurationVersionResourceID(applicationID, configurationProfileID, 7)}},
			},
			loadApplications: true,
			loadChildren:     true,
			loadProfiles:     true,
			loadHosted:       true,
			appendHosted:     true,
		},
		{
			name: "typed deployment id filter scopes environment deployment discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigDeploymentResourceType, FieldPath: "id", AcceptableValues: []string{appConfigDeploymentResourceID(applicationID, environmentID, 3)}},
			},
			loadApplications: true,
			loadChildren:     true,
			loadEnvironments: true,
			loadDeployments:  true,
			appendDeployment: true,
		},
		{
			name: "typed application non-id filter avoids child pre-load",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigApplicationResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadApplications:  true,
			appendApplication: true,
			appendOther:       true,
		},
		{
			name: "global id filter constrains typed child discovery",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{otherApplicationID}},
				{ServiceName: appConfigConfigurationProfileResourceType, FieldPath: "id", AcceptableValues: []string{appConfigConfigurationProfileResourceID(configurationProfileID, applicationID)}},
			},
			loadApplications: true,
			appendOther:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppConfigGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadApplications(); got != tt.loadApplications {
				t.Fatalf("shouldLoadApplications() = %t, want %t", got, tt.loadApplications)
			}
			if got := g.shouldAppendTopLevelResource(appConfigApplicationResourceType, application); got != tt.appendApplication {
				t.Fatalf("shouldAppendTopLevelResource(application) = %t, want %t", got, tt.appendApplication)
			}
			if got := g.shouldAppendTopLevelResource(appConfigApplicationResourceType, otherApplication); got != tt.appendOther {
				t.Fatalf("shouldAppendTopLevelResource(other application) = %t, want %t", got, tt.appendOther)
			}
			if got := g.shouldLoadApplicationChildren(application); got != tt.loadChildren {
				t.Fatalf("shouldLoadApplicationChildren(application) = %t, want %t", got, tt.loadChildren)
			}
			if got := g.shouldLoadApplicationChildren(otherApplication); got != tt.loadOtherChildren {
				t.Fatalf("shouldLoadApplicationChildren(other application) = %t, want %t", got, tt.loadOtherChildren)
			}
			if got := g.shouldLoadConfigurationProfiles(applicationID); got != tt.loadProfiles {
				t.Fatalf("shouldLoadConfigurationProfiles(application) = %t, want %t", got, tt.loadProfiles)
			}
			if got := g.shouldLoadConfigurationProfiles(otherApplicationID); got != tt.loadOtherProfiles {
				t.Fatalf("shouldLoadConfigurationProfiles(other application) = %t, want %t", got, tt.loadOtherProfiles)
			}
			if got := g.shouldLoadHostedConfigurationVersions(applicationID, configurationProfileID, appConfigHostedLocationURI); got != tt.loadHosted {
				t.Fatalf("shouldLoadHostedConfigurationVersions(profile) = %t, want %t", got, tt.loadHosted)
			}
			if got := g.shouldLoadHostedConfigurationVersions(applicationID, otherConfigurationProfileID, appConfigHostedLocationURI); got != tt.loadOtherHosted {
				t.Fatalf("shouldLoadHostedConfigurationVersions(other profile) = %t, want %t", got, tt.loadOtherHosted)
			}
			if got := g.shouldLoadHostedConfigurationVersions(applicationID, configurationProfileID, "s3://bucket/key"); got {
				t.Fatalf("shouldLoadHostedConfigurationVersions(non-hosted profile) = true, want false")
			}
			if got := g.shouldLoadEnvironments(applicationID); got != tt.loadEnvironments {
				t.Fatalf("shouldLoadEnvironments(application) = %t, want %t", got, tt.loadEnvironments)
			}
			if got := g.shouldLoadEnvironments(otherApplicationID); got != tt.loadOtherEnvironments {
				t.Fatalf("shouldLoadEnvironments(other application) = %t, want %t", got, tt.loadOtherEnvironments)
			}
			if got := g.shouldLoadDeployments(applicationID, environmentID); got != tt.loadDeployments {
				t.Fatalf("shouldLoadDeployments(environment) = %t, want %t", got, tt.loadDeployments)
			}
			if got := g.shouldLoadDeployments(applicationID, otherEnvironmentID); got != tt.loadOtherDeployments {
				t.Fatalf("shouldLoadDeployments(other environment) = %t, want %t", got, tt.loadOtherDeployments)
			}
			if got := g.shouldAppendApplicationChildResource(appConfigConfigurationProfileResourceType, configurationProfile); got != tt.appendProfile {
				t.Fatalf("shouldAppendApplicationChildResource(profile) = %t, want %t", got, tt.appendProfile)
			}
			if got := g.shouldAppendApplicationChildResource(appConfigEnvironmentResourceType, environment); got != tt.appendEnvironment {
				t.Fatalf("shouldAppendApplicationChildResource(environment) = %t, want %t", got, tt.appendEnvironment)
			}
			if got := g.shouldAppendApplicationChildResource(appConfigHostedConfigurationVersionResourceType, hostedVersion); got != tt.appendHosted {
				t.Fatalf("shouldAppendApplicationChildResource(hosted version) = %t, want %t", got, tt.appendHosted)
			}
			if got := g.shouldAppendApplicationChildResource(appConfigDeploymentResourceType, deployment); got != tt.appendDeployment {
				t.Fatalf("shouldAppendApplicationChildResource(deployment) = %t, want %t", got, tt.appendDeployment)
			}
		})
	}
}

func TestAppConfigFilterGatesTopLevelFamilies(t *testing.T) {
	strategy := newAppConfigDeploymentStrategyResource("strategy123", "linear")
	extension := newAppConfigExtensionResource("ext123", "notify")
	association := newAppConfigExtensionAssociationResource("assoc123")

	tests := []struct {
		name             string
		filters          []terraformutils.ResourceFilter
		loadApplications bool
		loadStrategies   bool
		loadExtensions   bool
		loadAssociations bool
		appendStrategy   bool
		appendExtension  bool
		appendAssoc      bool
	}{
		{
			name:             "no filters loads all top-level families",
			loadApplications: true,
			loadStrategies:   true,
			loadExtensions:   true,
			loadAssociations: true,
			appendStrategy:   true,
			appendExtension:  true,
			appendAssoc:      true,
		},
		{
			name: "typed deployment strategy filter only loads strategies",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigDeploymentStrategyResourceType, FieldPath: "id", AcceptableValues: []string{"strategy123"}},
			},
			loadStrategies: true,
			appendStrategy: true,
		},
		{
			name: "typed extension filter only loads extensions",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appConfigExtensionResourceType, FieldPath: "id", AcceptableValues: []string{"ext123"}},
			},
			loadExtensions:  true,
			appendExtension: true,
		},
		{
			name: "untyped id filter keeps broad scans available",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{"assoc123"}},
			},
			loadApplications: true,
			loadStrategies:   true,
			loadExtensions:   true,
			loadAssociations: true,
			appendAssoc:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppConfigGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadApplications(); got != tt.loadApplications {
				t.Fatalf("shouldLoadApplications() = %t, want %t", got, tt.loadApplications)
			}
			if got := g.shouldLoadTopLevelResourceType(appConfigDeploymentStrategyResourceType); got != tt.loadStrategies {
				t.Fatalf("shouldLoadTopLevelResourceType(strategy) = %t, want %t", got, tt.loadStrategies)
			}
			if got := g.shouldLoadTopLevelResourceType(appConfigExtensionResourceType); got != tt.loadExtensions {
				t.Fatalf("shouldLoadTopLevelResourceType(extension) = %t, want %t", got, tt.loadExtensions)
			}
			if got := g.shouldLoadTopLevelResourceType(appConfigExtensionAssociationResourceType); got != tt.loadAssociations {
				t.Fatalf("shouldLoadTopLevelResourceType(association) = %t, want %t", got, tt.loadAssociations)
			}
			if got := g.shouldAppendTopLevelResource(appConfigDeploymentStrategyResourceType, strategy); got != tt.appendStrategy {
				t.Fatalf("shouldAppendTopLevelResource(strategy) = %t, want %t", got, tt.appendStrategy)
			}
			if got := g.shouldAppendTopLevelResource(appConfigExtensionResourceType, extension); got != tt.appendExtension {
				t.Fatalf("shouldAppendTopLevelResource(extension) = %t, want %t", got, tt.appendExtension)
			}
			if got := g.shouldAppendTopLevelResource(appConfigExtensionAssociationResourceType, association); got != tt.appendAssoc {
				t.Fatalf("shouldAppendTopLevelResource(association) = %t, want %t", got, tt.appendAssoc)
			}
		})
	}
}
