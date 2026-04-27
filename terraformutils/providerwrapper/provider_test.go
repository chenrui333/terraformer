package providerwrapper

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

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
