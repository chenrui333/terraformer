// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformutils

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
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
	attributes := instance["attributes_flat"].(map[string]interface{})
	if got := attributes["id"].(string); got != "vpc-123" {
		t.Fatalf("flat id attribute = %q, want %q", got, "vpc-123")
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
	resource.InstanceState.Meta = map[string]interface{}{"schema_version": 2}
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
