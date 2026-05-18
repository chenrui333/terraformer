// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"reflect"
	"testing"
)

func TestParseAndGroupResourcesIncludesChatbot(t *testing.T) {
	global, eastOnly, chatbot, regionalOnce, regional := parseAndGroupResources([]string{"iam", "notifications", "chatbot", "sns"})
	assertStringSlice(t, global, []string{"iam"})
	assertStringSlice(t, eastOnly, []string{"notifications"})
	assertStringSlice(t, chatbot, []string{"chatbot"})
	assertStringSlice(t, regionalOnce, nil)
	assertStringSlice(t, regional, []string{"sns"})
}

func TestParseAndGroupResourcesTreatsNetworkManagerAsRegionalOnce(t *testing.T) {
	global, eastOnly, chatbot, regionalOnce, regional := parseAndGroupResources([]string{"networkmanager", "sns"})
	assertStringSlice(t, global, nil)
	assertStringSlice(t, eastOnly, nil)
	assertStringSlice(t, chatbot, nil)
	assertStringSlice(t, regionalOnce, []string{"networkmanager"})
	assertStringSlice(t, regional, []string{"sns"})
}

func TestChatbotImportRegionsDeduplicatesEffectiveRegions(t *testing.T) {
	got := chatbotImportRegions([]string{
		"us-east-1",
		"eu-central-1",
		"us-west-2",
		"eu-west-1",
		"ap-southeast-1",
		"us-east-2",
		"ap-south-1",
	})
	want := []string{"us-west-2", "eu-west-1", "ap-southeast-1", "us-east-2"}
	assertStringSlice(t, got, want)
}

func TestChatbotPathPattern(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "shared provider path", path: "{output}/{provider}/", want: "{output}/{provider}/{service}/"},
		{name: "service path unchanged", path: "{output}/{provider}/{service}/", want: "{output}/{provider}/{service}/"},
		{name: "no provider trailing slash", path: "{output}/aws/", want: "{output}/aws/{service}/"},
		{name: "no provider no trailing slash", path: "{output}/aws", want: "{output}/aws/{service}/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatbotPathPattern(tt.path); got != tt.want {
				t.Fatalf("chatbotPathPattern(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}
