// SPDX-License-Identifier: Apache-2.0

package terraformoutput

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/zclconf/go-cty/cty"
)

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"json format", "json", "tf.json"},
		{"hcl format", "hcl", "tf"},
		{"empty format", "", "tf"},
		{"unknown format", "yaml", "tf"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetFileExtension(tc.format); got != tc.want {
				t.Errorf("GetFileExtension(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}

func TestBucketPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"with trailing slash", "generated/aws/vpc/", "generated/aws/vpc"},
		{"without trailing slash", "generated/aws/vpc", "generated/aws/vpc"},
		{"single slash", "/", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := BucketState{}
			if got := b.BucketPrefix(tc.path); got != tc.want {
				t.Errorf("BucketPrefix(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestBucketGetTfData(t *testing.T) {
	tests := []struct {
		name       string
		bucketName string
		path       string
		wantBucket string
		wantPrefix string
	}{
		{"with gs prefix", "gs://my-bucket", "generated/aws/vpc/", "my-bucket", "generated/aws/vpc"},
		{"without gs prefix", "my-bucket", "state/", "my-bucket", "state"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := BucketState{Name: tc.bucketName}
			data := b.BucketGetTfData(tc.path)

			dataMap, ok := data.(map[string]interface{})
			if !ok {
				t.Fatalf("result is not map[string]interface{}, got %T", data)
			}
			terraform, ok := dataMap["terraform"].(map[string]interface{})
			if !ok {
				t.Fatal("missing terraform key")
			}
			backends, ok := terraform["backend"].([]map[string]interface{})
			if !ok || len(backends) != 1 {
				t.Fatal("missing or invalid backend key")
			}
			gcs, ok := backends[0]["gcs"].(map[string]interface{})
			if !ok {
				t.Fatal("missing gcs key")
			}
			if gcs["bucket"] != tc.wantBucket {
				t.Errorf("bucket = %v, want %q", gcs["bucket"], tc.wantBucket)
			}
			if gcs["prefix"] != tc.wantPrefix {
				t.Errorf("prefix = %v, want %q", gcs["prefix"], tc.wantPrefix)
			}
		})
	}
}

func TestPrintFileWritesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provider.tf")
	want := []byte("terraform {}")

	if err := PrintFile(path, want); err != nil {
		t.Fatalf("PrintFile() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("PrintFile() wrote %q, want %q", got, want)
	}
}

func TestPrintFileReturnsWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "provider.tf")
	if err := PrintFile(path, []byte("terraform {}")); err == nil {
		t.Fatal("PrintFile() error = nil, want write error")
	}
}

func TestOutputHclFilesSkipsManifestIDOutput(t *testing.T) {
	path := t.TempDir()
	resource := terraformutils.NewSimpleResource(
		"apiVersion=example.com/v1,kind=Widget,name=sample",
		"example.com/v1/Widget/sample",
		"kubernetes_manifest",
		"kubernetes",
		nil,
	)
	resource.Item = map[string]interface{}{
		"manifest": map[string]interface{}{
			"apiVersion": "example.com/v1",
			"kind":       "Widget",
		},
	}

	if err := OutputHclFiles([]terraformutils.Resource{resource}, &testProvider{name: "kubernetes"}, path, "example.com/v1/widgets", false, "hcl", true); err != nil {
		t.Fatalf("OutputHclFiles() error = %v", err)
	}

	outputsPath := filepath.Join(path, "outputs.tf")
	if _, err := os.Stat(outputsPath); !os.IsNotExist(err) {
		t.Fatalf("outputs.tf exists for manifest resource, stat err = %v", err)
	}
}

func TestOutputHclFilesKeepsNativeIDOutput(t *testing.T) {
	path := t.TempDir()
	resource := terraformutils.NewResource(
		"svc-123",
		"sample",
		"kubernetes_service_v1",
		"kubernetes",
		map[string]string{"id": "svc-123"},
		nil,
		nil,
	)
	resource.Item = map[string]interface{}{"metadata": map[string]interface{}{"name": "sample"}}

	if err := OutputHclFiles([]terraformutils.Resource{resource}, &testProvider{name: "kubernetes"}, path, "services", false, "hcl", true); err != nil {
		t.Fatalf("OutputHclFiles() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(path, "outputs.tf"))
	if err != nil {
		t.Fatalf("ReadFile(outputs.tf) error = %v", err)
	}
	if !strings.Contains(string(data), "kubernetes_service_v1."+resource.ResourceName+".id") {
		t.Fatalf("outputs.tf missing native id reference:\n%s", data)
	}
}

type testProvider struct {
	terraformutils.Provider
	name string
}

func (p testProvider) GetName() string { return p.name }

func (p testProvider) InitService(_ string, _ bool) error { return nil }

func (p testProvider) GetConfig() cty.Value { return cty.EmptyObjectVal }

func (p testProvider) GetBasicConfig() cty.Value { return cty.EmptyObjectVal }

func (p testProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p testProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}
