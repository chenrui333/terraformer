// SPDX-License-Identifier: Apache-2.0

package terraformoutput

import (
	"os"
	"path/filepath"
	"testing"
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
