// SPDX-License-Identifier: Apache-2.0

package terraformutils

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/zclconf/go-cty/cty"
)

func TestPrintTfStateWritesV4ProviderSourceState(t *testing.T) {
	stateBytes := testStateBytes(t)

	var state map[string]interface{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}

	if got := int(state["version"].(float64)); got != 4 {
		t.Fatalf("state version = %d, want 4", got)
	}
	if got := state["terraform_version"].(string); got != tfStateTerraformVersion {
		t.Fatalf("terraform version = %q, want %q", got, tfStateTerraformVersion)
	}
	outputs := state["outputs"].(map[string]interface{})
	output := outputs["aws_vpc_main_id"].(map[string]interface{})
	if got := output["value"].(string); got != "vpc-123" {
		t.Fatalf("output value = %q, want %q", got, "vpc-123")
	}

	resources := state["resources"].([]interface{})
	if len(resources) != 1 {
		t.Fatalf("resource count = %d, want 1", len(resources))
	}
	gotResource := resources[0].(map[string]interface{})
	if got := gotResource["provider"].(string); got != "provider[\"registry.terraform.io/hashicorp/aws\"]" {
		t.Fatalf("provider address = %q", got)
	}
	if got := gotResource["type"].(string); got != "aws_vpc" {
		t.Fatalf("resource type = %q, want %q", got, "aws_vpc")
	}
	if got := gotResource["name"].(string); got != "tfer--main" {
		t.Fatalf("resource name = %q, want %q", got, "tfer--main")
	}

	instances := gotResource["instances"].([]interface{})
	instance := instances[0].(map[string]interface{})
	if got := int(instance["schema_version"].(float64)); got != 2 {
		t.Fatalf("schema version = %d, want 2", got)
	}
	if _, ok := instance["attributes_flat"]; ok {
		t.Fatal("state instance contains legacy attributes_flat")
	}
	attributes := instance["attributes"].(map[string]interface{})
	if got := attributes["id"].(string); got != "vpc-123" {
		t.Fatalf("id attribute = %q, want %q", got, "vpc-123")
	}
	if got := attributes["enable_dns_hostnames"].(bool); !got {
		t.Fatal("enable_dns_hostnames = false, want true")
	}
	tags := attributes["tags"].(map[string]interface{})
	if got := tags["env"].(string); got != "test" {
		t.Fatalf("tags.env = %q, want %q", got, "test")
	}
	sensitiveAttributes := instance["sensitive_attributes"].([]interface{})
	if len(sensitiveAttributes) != 0 {
		t.Fatalf("sensitive attributes = %v, want empty", sensitiveAttributes)
	}
}

func TestPrintTfStateFallsBackToAttributesFlat(t *testing.T) {
	resource := NewResource(
		"vpc-123",
		"main",
		"aws_vpc",
		"aws",
		map[string]string{"cidr_block": "10.0.0.0/16"},
		nil,
		nil,
	)
	resource.InstanceState.Meta = map[string]interface{}{"schema_version": 2}

	stateBytes, err := PrintTfState([]Resource{resource})
	if err != nil {
		t.Fatal(err)
	}

	var state map[string]interface{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}

	resources := state["resources"].([]interface{})
	gotResource := resources[0].(map[string]interface{})
	instances := gotResource["instances"].([]interface{})
	instance := instances[0].(map[string]interface{})

	if _, ok := instance["attributes"]; ok {
		t.Fatal("state instance contains typed attributes, want attributes_flat fallback")
	}
	attributesFlat := instance["attributes_flat"].(map[string]interface{})
	if got := attributesFlat["id"].(string); got != "vpc-123" {
		t.Fatalf("id attribute = %q, want %q", got, "vpc-123")
	}
	if got := attributesFlat["cidr_block"].(string); got != "10.0.0.0/16" {
		t.Fatalf("cidr_block attribute = %q, want %q", got, "10.0.0.0/16")
	}
	if _, ok := instance["sensitive_attributes"]; ok {
		t.Fatal("state instance contains sensitive_attributes for flat fallback")
	}
}

func TestPrintTfStatePreservesManifestObjectTypedAttribute(t *testing.T) {
	resource := NewSimpleResource(
		"apiVersion=example.com/v1,kind=Widget,name=sample",
		"example.com/v1/Widget/sample",
		"kubernetes_manifest",
		"kubernetes",
		nil,
	)
	resource.InstanceState.TypedAttributes = json.RawMessage("{\"manifest\":{\"apiVersion\":\"example.com/v1\",\"kind\":\"Widget\"},\"object\":{\"apiVersion\":\"example.com/v1\",\"kind\":\"Widget\",\"status\":{\"phase\":\"Ready\"}}}")

	stateBytes, err := PrintTfState([]Resource{resource})
	if err != nil {
		t.Fatal(err)
	}

	var state map[string]interface{}
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatal(err)
	}
	resources := state["resources"].([]interface{})
	gotResource := resources[0].(map[string]interface{})
	instances := gotResource["instances"].([]interface{})
	instance := instances[0].(map[string]interface{})
	attributes := instance["attributes"].(map[string]interface{})
	object, ok := attributes["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("object attribute type = %T, want map[string]interface{}", attributes["object"])
	}
	if object["kind"] != "Widget" {
		t.Fatalf("object.kind = %v, want %q", object["kind"], "Widget")
	}
}

func TestConvertTypedStatePreservesCurrentTypedAttributes(t *testing.T) {
	resource := NewResource(
		"kind-lemur",
		"example",
		"random_pet",
		"random",
		nil,
		nil,
		nil,
	)
	resource.InstanceState = tfcompat.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"id":        cty.StringVal("kind-lemur"),
		"length":    cty.NumberIntVal(2),
		"separator": cty.StringVal("-"),
	}), 0)
	originalTypedAttributes := append([]byte(nil), resource.InstanceState.TypedAttributes...)

	if err := resource.ConvertTypedState(nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(resource.InstanceState.TypedAttributes, originalTypedAttributes) {
		t.Fatalf("typed attributes were re-derived: got %s, want %s", resource.InstanceState.TypedAttributes, originalTypedAttributes)
	}
}

func TestPrintTfStateCanBeListedByTerraformCLI(t *testing.T) {
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform CLI not found")
	}

	statePath := filepath.Join(t.TempDir(), "terraform.tfstate")
	if err := os.WriteFile(statePath, testStateBytes(t), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(terraformPath, "state", "list", "-state="+statePath).CombinedOutput()
	if err != nil {
		t.Fatalf("terraform state list failed: %s\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != "aws_vpc.tfer--main" {
		t.Fatalf("terraform state list = %q, want %q", got, "aws_vpc.tfer--main")
	}
}

func TestPrintTfStateCanPlanWithTerraformCLI(t *testing.T) {
	if os.Getenv("TERRAFORMER_TFSTATE_PLAN_TEST") == "" {
		t.Skip("set TERRAFORMER_TFSTATE_PLAN_TEST to run provider-backed plan check")
	}
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform CLI not found")
	}

	resource := NewResource(
		"kind-lemur",
		"example",
		"random_pet",
		"random",
		nil,
		nil,
		nil,
	)
	resource.InstanceState = tfcompat.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"id":        cty.StringVal("kind-lemur"),
		"keepers":   cty.NullVal(cty.Map(cty.String)),
		"length":    cty.NumberIntVal(2),
		"prefix":    cty.NullVal(cty.String),
		"separator": cty.StringVal("-"),
	}), 0)

	testDir := t.TempDir()
	stateBytes, err := PrintTfState([]Resource{resource})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "terraform.tfstate"), stateBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	mainTF := []byte(`terraform {
  required_version = ">= 1.9, < 1.15"
  required_providers {
    random = {
      source = "hashicorp/random"
      version = "3.7.2"
    }
  }
}

resource "random_pet" "tfer--example" {
  length    = 2
  separator = "-"
}
`)
	if err := os.WriteFile(filepath.Join(testDir, "main.tf"), mainTF, 0o600); err != nil {
		t.Fatal(err)
	}

	initOut, err := exec.Command(terraformPath, "-chdir="+testDir, "init", "-backend=false", "-input=false", "-no-color").CombinedOutput()
	if err != nil {
		t.Fatalf("terraform init failed: %s\n%s", err, initOut)
	}
	planOut, err := exec.Command(terraformPath, "-chdir="+testDir, "plan", "-refresh=false", "-input=false", "-no-color", "-detailed-exitcode").CombinedOutput()
	if err != nil {
		t.Fatalf("terraform plan failed: %s\n%s", err, planOut)
	}
}

func testStateBytes(t *testing.T) []byte {
	t.Helper()

	resource := NewResource(
		"vpc-123",
		"main",
		"aws_vpc",
		"aws",
		map[string]string{"cidr_block": "10.0.0.0/16"},
		nil,
		nil,
	)
	resource.InstanceState = tfcompat.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"cidr_block":           cty.StringVal("10.0.0.0/16"),
		"enable_dns_hostnames": cty.BoolVal(true),
		"id":                   cty.StringVal("vpc-123"),
		"tags": cty.MapVal(map[string]cty.Value{
			"env": cty.StringVal("test"),
		}),
	}), 2)
	resource.Outputs = map[string]*tfcompat.OutputState{
		"aws_vpc_main_id": {
			Type:  "string",
			Value: "vpc-123",
		},
	}

	stateBytes, err := PrintTfState([]Resource{resource})
	if err != nil {
		t.Fatal(err)
	}
	return stateBytes
}
