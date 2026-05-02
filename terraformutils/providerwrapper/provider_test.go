package providerwrapper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestIgnoredAttributes(t *testing.T) {
	attributes := map[string]*configschema.Attribute{
		"computed_attribute": {
			Type:     cty.Number,
			Computed: true,
		},
		"required_attribute": {
			Type:     cty.String,
			Required: true,
		},
	}

	testCases := map[string]struct {
		block                map[string]*configschema.NestedBlock
		ignoredAttributes    []string
		notIgnoredAttributes []string
	}{
		"nesting_set": {map[string]*configschema.NestedBlock{
			"attribute_one": {
				Block: configschema.Block{
					Attributes: attributes,
				},
				Nesting: configschema.NestingSet,
			},
		}, []string{"nesting_set.attribute_one.computed_attribute"},
			[]string{"nesting_set.attribute_one.required_attribute"}},
		"nesting_list": {map[string]*configschema.NestedBlock{
			"attribute_one": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{},
					BlockTypes: map[string]*configschema.NestedBlock{
						"attribute_two_nested": {
							Nesting: configschema.NestingList,
							Block: configschema.Block{
								Attributes: attributes,
							},
						},
					},
				},
				Nesting: configschema.NestingList,
			},
		}, []string{"nesting_list.0.attribute_one.0.attribute_two_nested.computed_attribute"},
			[]string{"nesting_list.0.attribute_one.0.attribute_two_nested.required_attribute"}},
	}

	for key, tc := range testCases {
		t.Run(key, func(t *testing.T) {
			provider := ProviderWrapper{}
			readOnlyAttributes := provider.readObjBlocks(tc.block, []string{}, key)
			for _, attr := range tc.ignoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); !ignored {
					t.Errorf("attribute \"%s\" was not ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}

			for _, attr := range tc.notIgnoredAttributes {
				if ignored := isAttributeIgnored(attr, readOnlyAttributes); ignored {
					t.Errorf("attribute \"%s\" was ignored. Pattern list: %s", attr, readOnlyAttributes)
				}
			}
		})
	}
}

func TestPopulateKubernetesManifestFromObject(t *testing.T) {
	state := &tfcompat.InstanceState{
		Attributes: map[string]string{
			"id": "apiVersion=example.com/v1,kind=Widget,name=sample",
		},
		TypedAttributes: json.RawMessage(`{
			"id": "apiVersion=example.com/v1,kind=Widget,name=sample",
			"manifest": {},
			"object": {
				"apiVersion": "example.com/v1",
				"kind": "Widget",
				"metadata": {
					"name": "sample"
				}
			}
		}`),
	}

	populateKubernetesManifestFromObject(kubernetesManifestResourceType, state)

	attributes := map[string]interface{}{}
	if err := json.Unmarshal(state.TypedAttributes, &attributes); err != nil {
		t.Fatalf("TypedAttributes unmarshal error = %v", err)
	}
	manifest, ok := attributes["manifest"].(map[string]interface{})
	if !ok {
		t.Fatalf("manifest type = %T, want map[string]interface{}", attributes["manifest"])
	}
	if manifest["apiVersion"] != "example.com/v1" {
		t.Fatalf("manifest.apiVersion = %v, want %q", manifest["apiVersion"], "example.com/v1")
	}
	if manifest["kind"] != "Widget" {
		t.Fatalf("manifest.kind = %v, want %q", manifest["kind"], "Widget")
	}
	object, ok := attributes["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("object type = %T, want map[string]interface{}", attributes["object"])
	}
	if object["kind"] != "Widget" {
		t.Fatalf("object.kind = %v, want %q", object["kind"], "Widget")
	}
	if !state.HasCurrentTypedAttributes() {
		t.Fatal("typed attributes were not marked current after manifest population")
	}
}

func TestPopulateKubernetesManifestFromObjectPreservesExistingManifest(t *testing.T) {
	state := &tfcompat.InstanceState{
		Attributes: map[string]string{
			"id": "apiVersion=example.com/v1,kind=Widget,name=sample",
		},
		TypedAttributes: json.RawMessage(`{
			"manifest": {
				"apiVersion": "example.com/v1",
				"kind": "Widget",
				"metadata": {
					"name": "configured"
				}
			},
			"object": {
				"apiVersion": "example.com/v1",
				"kind": "Widget",
				"metadata": {
					"name": "sample"
				}
			}
		}`),
	}

	populateKubernetesManifestFromObject(kubernetesManifestResourceType, state)

	attributes := map[string]interface{}{}
	if err := json.Unmarshal(state.TypedAttributes, &attributes); err != nil {
		t.Fatalf("TypedAttributes unmarshal error = %v", err)
	}
	manifest := attributes["manifest"].(map[string]interface{})
	metadata := manifest["metadata"].(map[string]interface{})
	if metadata["name"] != "configured" {
		t.Fatalf("manifest.metadata.name = %v, want %q", metadata["name"], "configured")
	}
	object, ok := attributes["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("object type = %T, want map[string]interface{}", attributes["object"])
	}
	if object["kind"] != "Widget" {
		t.Fatalf("object.kind = %v, want %q", object["kind"], "Widget")
	}
}

func TestGetProviderFileNameUsesTerraform1DataDirCache(t *testing.T) {
	homeDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), ".terraform")
	t.Setenv("HOME", homeDir)
	t.Setenv("TF_DATA_DIR", dataDir)

	want := writeProviderBinary(t,
		filepath.Join(dataDir, "providers", "registry.terraform.io", "hashicorp", "aws", "1.0.0", runtime.GOOS+"_"+runtime.GOARCH),
		"terraform-provider-aws_v1.0.0",
	)

	got, err := getProviderFileName("aws")
	if err != nil {
		t.Fatalf("getProviderFileName returned error: %s", err)
	}
	if got != want {
		t.Fatalf("getProviderFileName = %q, want %q", got, want)
	}
}

func TestGetProviderFileNameUsesTerraform1HomePluginMirror(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("TF_DATA_DIR", filepath.Join(t.TempDir(), ".terraform"))

	want := writeProviderBinary(t,
		filepath.Join(homeDir, ".terraform.d", "plugins", "registry.terraform.io", "hashicorp", "aws", "1.0.0", runtime.GOOS+"_"+runtime.GOARCH),
		"terraform-provider-aws_v1.0.0",
	)

	got, err := getProviderFileName("aws")
	if err != nil {
		t.Fatalf("getProviderFileName returned error: %s", err)
	}
	if got != want {
		t.Fatalf("getProviderFileName = %q, want %q", got, want)
	}
}

func TestGetProviderFileNameIgnoresLegacyPluginDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), ".terraform")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TF_DATA_DIR", dataDir)

	writeProviderBinary(t,
		filepath.Join(dataDir, "plugins", runtime.GOOS+"_"+runtime.GOARCH),
		"terraform-provider-aws_v1.0.0",
	)

	got, _ := getProviderFileName("aws")
	if got != "" {
		t.Fatalf("getProviderFileName found legacy pre-1.9 plugin path %q", got)
	}
}

func TestGetProviderFileNameReturnsAllRegistryDirErrors(t *testing.T) {
	homeDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), ".terraform")
	t.Setenv("HOME", homeDir)
	t.Setenv("TF_DATA_DIR", dataDir)

	_, err := getProviderFileName("aws")
	if err == nil {
		t.Fatal("getProviderFileName returned nil error")
	}

	wantParts := []string{
		filepath.Join(dataDir, "providers", "registry.terraform.io"),
		filepath.Join(homeDir, ".terraform.d", "plugins", "registry.terraform.io"),
	}
	for _, want := range wantParts {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not include registry dir %q", err, want)
		}
	}
}

func TestGetProviderFileNameReturnsErrorWhenProviderMissing(t *testing.T) {
	homeDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), ".terraform")
	t.Setenv("HOME", homeDir)
	t.Setenv("TF_DATA_DIR", dataDir)

	registryDirs := []string{
		filepath.Join(dataDir, "providers", "registry.terraform.io"),
		filepath.Join(homeDir, ".terraform.d", "plugins", "registry.terraform.io"),
	}
	for _, registryDir := range registryDirs {
		if err := os.MkdirAll(registryDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := getProviderFileName("aws")
	if err == nil {
		t.Fatal("getProviderFileName returned nil error")
	}
	if got != "" {
		t.Fatalf("getProviderFileName = %q, want empty path", got)
	}
	if !strings.Contains(err.Error(), `provider "aws" not found`) {
		t.Fatalf("error %q does not include missing provider context", err)
	}
	for _, registryDir := range registryDirs {
		if !strings.Contains(err.Error(), registryDir) {
			t.Fatalf("error %q does not include registry dir %q", err, registryDir)
		}
	}
}

func writeProviderBinary(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test provider"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func isAttributeIgnored(name string, patterns []string) bool {
	ignored := false
	for _, pattern := range patterns {
		if match, _ := regexp.MatchString(pattern, name); match {
			ignored = true
			break
		}
	}
	return ignored
}
