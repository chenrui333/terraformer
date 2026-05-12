// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"reflect"
	"testing"
)

func TestParseAndGroupResourcesIncludesChatbot(t *testing.T) {
	global, eastOnly, chatbot, regional := parseAndGroupResources([]string{"iam", "notifications", "chatbot", "sns"})
	assertStringSlice(t, global, []string{"iam"})
	assertStringSlice(t, eastOnly, []string{"notifications"})
	assertStringSlice(t, chatbot, []string{"chatbot"})
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

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}
