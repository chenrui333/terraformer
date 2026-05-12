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
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/providerproto"
	"github.com/chenrui333/terraformer/terraformutils/typedjson"
	"github.com/zclconf/go-cty/cty"
)

func TestRestartPreservesCurrentProviderOnInitError(t *testing.T) {
	currentProvider := &providerproto.GRPCProvider{}
	currentSchema := &providerproto.GetProviderSchemaResponse{}
	provider := &ProviderWrapper{
		Provider:     currentProvider,
		providerName: "missing-provider-for-restart-test",
		config:       cty.EmptyObjectVal,
		schema:       currentSchema,
	}

	if err := provider.Restart(); err == nil {
		t.Fatal("Restart() error = nil, want provider initialization error")
	}
	if provider.Provider != currentProvider {
		t.Fatal("Restart() replaced the current provider after initialization failed")
	}
	if provider.schema != currentSchema {
		t.Fatal("Restart() replaced the current schema after initialization failed")
	}
}

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
					"name": "sample",
					"namespace": "default",
					"resourceVersion": "123",
					"uid": "uid-123",
					"managedFields": [
						{
							"manager": "controller"
						}
					]
				},
				"status": {
					"phase": "Ready"
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
	metadata := manifest["metadata"].(map[string]interface{})
	if metadata["name"] != "sample" {
		t.Fatalf("manifest.metadata.name = %v, want %q", metadata["name"], "sample")
	}
	for _, key := range []string{"resourceVersion", "uid", "managedFields"} {
		if _, ok := metadata[key]; ok {
			t.Fatalf("manifest.metadata.%s was not stripped", key)
		}
	}
	if _, ok := manifest["status"]; ok {
		t.Fatal("manifest.status was not stripped")
	}
	object, ok := attributes["object"].(map[string]interface{})
	if !ok {
		t.Fatalf("object type = %T, want map[string]interface{}", attributes["object"])
	}
	if object["kind"] != "Widget" {
		t.Fatalf("object.kind = %v, want %q", object["kind"], "Widget")
	}
	objectMetadata := object["metadata"].(map[string]interface{})
	if objectMetadata["uid"] != "uid-123" {
		t.Fatalf("object.metadata.uid = %v, want %q", objectMetadata["uid"], "uid-123")
	}
	if _, ok := object["status"]; !ok {
		t.Fatal("object.status was not preserved")
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

func TestPopulateKubernetesManifestFromObjectPreservesJSONNumbers(t *testing.T) {
	state := &tfcompat.InstanceState{
		TypedAttributes: json.RawMessage("{\"manifest\":{},\"object\":{\"apiVersion\":\"example.com/v1\",\"kind\":\"Widget\",\"metadata\":{\"name\":\"sample\"},\"spec\":{\"bigInteger\":9007199254740993,\"preciseDecimal\":0.1234567890123456789}}}"),
	}

	populateKubernetesManifestFromObject(kubernetesManifestResourceType, state)

	attributes, err := typedjson.UnmarshalObject(state.TypedAttributes)
	if err != nil {
		t.Fatalf("TypedAttributes unmarshal error = %v", err)
	}
	manifest := attributes["manifest"].(map[string]interface{})
	manifestSpec := manifest["spec"].(map[string]interface{})
	assertJSONNumber(t, manifestSpec["bigInteger"], "9007199254740993")
	assertJSONNumber(t, manifestSpec["preciseDecimal"], "0.1234567890123456789")

	object := attributes["object"].(map[string]interface{})
	objectSpec := object["spec"].(map[string]interface{})
	assertJSONNumber(t, objectSpec["bigInteger"], "9007199254740993")
	assertJSONNumber(t, objectSpec["preciseDecimal"], "0.1234567890123456789")
}

func TestPreserveKubernetesManifestID(t *testing.T) {
	previous := &tfcompat.InstanceState{ID: "apiVersion=example.com/v1,kind=Widget,name=sample"}
	next := &tfcompat.InstanceState{}

	preserveKubernetesManifestID(kubernetesManifestResourceType, next, previous)
	if next.ID != previous.ID {
		t.Fatalf("manifest ID = %q, want %q", next.ID, previous.ID)
	}

	next.ID = "provider-id"
	preserveKubernetesManifestID(kubernetesManifestResourceType, next, previous)
	if next.ID != "provider-id" {
		t.Fatalf("manifest ID = %q, want existing provider ID", next.ID)
	}

	nonManifest := &tfcompat.InstanceState{}
	preserveKubernetesManifestID("kubernetes_service_v1", nonManifest, previous)
	if nonManifest.ID != "" {
		t.Fatalf("non-manifest ID = %q, want empty", nonManifest.ID)
	}
}

func TestPreservePriorStateID(t *testing.T) {
	previous := &tfcompat.InstanceState{
		ID: "arn:aws:logs:us-east-1:123456789012:anomaly-detector:detector-1",
		Meta: map[string]interface{}{
			tfcompat.MetaKeyPreserveIDAfterRefresh: true,
		},
	}
	next := &tfcompat.InstanceState{}

	preservePriorStateID(next, previous)
	if next.ID != previous.ID {
		t.Fatalf("preserved ID = %q, want %q", next.ID, previous.ID)
	}

	next.ID = "provider-id"
	preservePriorStateID(next, previous)
	if next.ID != "provider-id" {
		t.Fatalf("existing ID = %q, want provider-id", next.ID)
	}

	withoutOptIn := &tfcompat.InstanceState{ID: "import-id"}
	next = &tfcompat.InstanceState{}
	preservePriorStateID(next, withoutOptIn)
	if next.ID != "" {
		t.Fatalf("unmarked ID = %q, want empty", next.ID)
	}
}

func assertJSONNumber(t *testing.T, value interface{}, want string) {
	t.Helper()
	number, ok := value.(json.Number)
	if !ok {
		t.Fatalf("number type = %T, want json.Number", value)
	}
	if number.String() != want {
		t.Fatalf("number = %s, want %s", number.String(), want)
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
