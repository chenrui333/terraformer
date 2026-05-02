// SPDX-License-Identifier: Apache-2.0

package launchdarkly

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func TestAIConfigVariationAttributesSeedsToolKeys(t *testing.T) {
	got := aiConfigVariationAttributes("proj", "assistant", "helpful", []ldapi.VariationTool{
		{Key: "web-search"},
	})

	want := map[string]string{
		"project_key": "proj",
		"config_key":  "assistant",
		"key":         "helpful",
		"tool_keys.#": "1",
		fmt.Sprintf("tool_keys.%d", terraformutils.HashString("web-search")): "web-search",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("aiConfigVariationAttributes() = %#v, want %#v", got, want)
	}
}

func TestAIConfigVariationResourceNameIncludesConfigKey(t *testing.T) {
	got := aiConfigVariationResourceName("proj", "assistant", "Default", "default")
	want := "proj-assistant-Default-default"
	if got != want {
		t.Fatalf("aiConfigVariationResourceName() = %q, want %q", got, want)
	}

	other := aiConfigVariationResourceName("proj", "summarizer", "Default", "default")
	if got == other {
		t.Fatalf("aiConfigVariationResourceName() generated duplicate names %q and %q", got, other)
	}
}
