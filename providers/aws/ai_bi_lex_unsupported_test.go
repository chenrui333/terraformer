// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAIBiLexUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	rawEntries, ok := unsupported["resources"].([]interface{})
	if !ok {
		t.Fatal("unsupported resources file is missing resources list")
	}

	entries := map[string]struct {
		serviceFamily string
		status        string
	}{}
	for _, rawEntry := range rawEntries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		serviceFamily, _ := entry["service_family"].(string)
		status, _ := entry["status"].(string)
		reason, _ := entry["reason"].(string)
		evidence, _ := entry["evidence"].(string)
		references, _ := entry["references"].([]interface{})
		if reason == "" || evidence == "" || len(references) == 0 {
			t.Fatalf("%s unsupported entry is missing reason, evidence, or references", resource)
		}
		entries[resource] = struct {
			serviceFamily string
			status        string
		}{serviceFamily: serviceFamily, status: status}
	}

	expected := map[string]struct {
		serviceFamily string
		status        string
	}{
		"aws_bedrock_custom_model":                         {"bedrock", "unsupported"},
		"aws_bedrockagentcore_agent_runtime":               {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_agent_runtime_endpoint":      {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_api_key_credential_provider": {"bedrockagentcore", "unsupported"},
		"aws_bedrockagentcore_browser":                     {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_code_interpreter":            {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_gateway":                     {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_gateway_target":              {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_memory":                      {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_memory_strategy":             {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_oauth2_credential_provider":  {"bedrockagentcore", "unsupported"},
		"aws_bedrockagentcore_token_vault_cmk":             {"bedrockagentcore", "deferred"},
		"aws_bedrockagentcore_workload_identity":           {"bedrockagentcore", "deferred"},
		"aws_lexv2models_bot_version":                      {"lexv2models", "unsupported"},
		"aws_quicksight_account_settings":                  {"quicksight", "deferred"},
		"aws_quicksight_account_subscription":              {"quicksight", "deferred"},
		"aws_quicksight_analysis":                          {"quicksight", "deferred"},
		"aws_quicksight_custom_permissions":                {"quicksight", "deferred"},
		"aws_quicksight_dashboard":                         {"quicksight", "deferred"},
		"aws_quicksight_data_set":                          {"quicksight", "deferred"},
		"aws_quicksight_data_source":                       {"quicksight", "unsupported"},
		"aws_quicksight_iam_policy_assignment":             {"quicksight", "deferred"},
		"aws_quicksight_ingestion":                         {"quicksight", "unsafe-discovery"},
		"aws_quicksight_ip_restriction":                    {"quicksight", "deferred"},
		"aws_quicksight_key_registration":                  {"quicksight", "deferred"},
		"aws_quicksight_refresh_schedule":                  {"quicksight", "deferred"},
		"aws_quicksight_role_custom_permission":            {"quicksight", "deferred"},
		"aws_quicksight_role_membership":                   {"quicksight", "deferred"},
		"aws_quicksight_template":                          {"quicksight", "deferred"},
		"aws_quicksight_template_alias":                    {"quicksight", "deferred"},
		"aws_quicksight_theme":                             {"quicksight", "deferred"},
		"aws_quicksight_user":                              {"quicksight", "not-importable"},
		"aws_quicksight_user_custom_permission":            {"quicksight", "deferred"},
		"aws_sagemaker_device":                             {"sagemaker", "deferred"},
		"aws_sagemaker_hub":                                {"sagemaker", "deferred"},
		"aws_sagemaker_hyper_parameter_tuning_job":         {"sagemaker", "unsafe-discovery"},
		"aws_sagemaker_labeling_job":                       {"sagemaker", "unsafe-discovery"},
		"aws_sagemaker_model_card_export_job":              {"sagemaker", "unsafe-discovery"},
		"aws_sagemaker_training_job":                       {"sagemaker", "unsafe-discovery"},
	}

	for resource, want := range expected {
		got, ok := entries[resource]
		if !ok {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
		if got.serviceFamily != want.serviceFamily || got.status != want.status {
			t.Fatalf("%s entry = (%s, %s), want (%s, %s)", resource, got.serviceFamily, got.status, want.serviceFamily, want.status)
		}
	}
}
