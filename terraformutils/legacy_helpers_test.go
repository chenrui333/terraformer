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
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHashString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty", input: "", want: 0},
		{name: "short string", input: "abc", want: 891568578},
		{name: "known crc", input: "hello", want: 907060870},
		{name: "ebs volume attachment key", input: "/dev/sda-i-123-vol-123-", want: 1315432815},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HashString(tc.input); got != tc.want {
				t.Fatalf("HashString(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestReadPathOrContents(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "credentials.json")
	if err := os.WriteFile(path, []byte("{\"type\":\"service_account\"}"), 0o600); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name         string
		value        string
		wantContents string
		wantPath     bool
	}{
		{
			name:         "empty",
			value:        "",
			wantContents: "",
			wantPath:     false,
		},
		{
			name:         "literal contents",
			value:        "inline-secret",
			wantContents: "inline-secret",
			wantPath:     false,
		},
		{
			name:         "existing path",
			value:        path,
			wantContents: "{\"type\":\"service_account\"}",
			wantPath:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotContents, gotPath, err := ReadPathOrContents(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if gotContents != tc.wantContents {
				t.Fatalf("contents = %q, want %q", gotContents, tc.wantContents)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("wasPath = %t, want %t", gotPath, tc.wantPath)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input map[string]interface{}
		want  map[string]string
	}{
		{
			name: "primitive values",
			input: map[string]interface{}{
				"enabled": true,
				"name":    "main",
				"port":    443,
			},
			want: map[string]string{
				"enabled": "true",
				"name":    "main",
				"port":    "443",
			},
		},
		{
			name: "nested list of maps",
			input: map[string]interface{}{
				"ingress": []map[interface{}]interface{}{
					{
						"ports": []string{"80", "443"},
						"rule":  "web",
					},
				},
			},
			want: map[string]string{
				"ingress.#":         "1",
				"ingress.0.ports.#": "2",
				"ingress.0.ports.0": "80",
				"ingress.0.ports.1": "443",
				"ingress.0.rule":    "web",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Flatten(tc.input); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Flatten() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
