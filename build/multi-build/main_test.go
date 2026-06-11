package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEnumerateProvidersPreservesLegacyNames(t *testing.T) {
	cmdDir := filepath.Join(t.TempDir(), "cmd")
	mustMkdir(t, cmdDir)
	writeFile(t, filepath.Join(cmdDir, "provider_cmd_google.go"), "package cmd\n")
	writeFile(t, filepath.Join(cmdDir, "provider_cmd_aws.go"), "package cmd\n")
	writeFile(t, filepath.Join(cmdDir, "provider_cmd_aws_test.go"), "package cmd\n")
	writeFile(t, filepath.Join(cmdDir, "root.go"), "package cmd\n")

	providers, err := enumerateProviders(cmdDir)
	if err != nil {
		t.Fatalf("enumerateProviders() error = %v", err)
	}

	got := providerNames(providers)
	want := []string{"aws", "aws_test", "google"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("provider names = %#v, want %#v", got, want)
	}
}

func TestSelectProvidersRejectsUnknownFilter(t *testing.T) {
	providers := []providerCommand{{Name: "aws", File: "provider_cmd_aws.go"}}

	_, err := selectProviders(providers, []string{"aws", "missing"})
	if err == nil {
		t.Fatal("selectProviders() error = nil, want missing provider error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("selectProviders() error = %q, want missing provider name", err)
	}
}

func TestBuildTargetArtifactNames(t *testing.T) {
	provider := providerCommand{Name: "aws", File: "provider_cmd_aws.go"}
	outputDir := t.TempDir()

	tests := []struct {
		osName string
		arch   string
		goos   string
		name   string
	}{
		{osName: "linux", arch: "amd64", goos: "linux", name: "terraformer-aws-linux-amd64"},
		{osName: "windows", arch: "arm64", goos: "windows", name: "terraformer-aws-windows-arm64.exe"},
		{osName: "darwin", arch: "amd64", goos: "darwin", name: "terraformer-aws-darwin-amd64"},
		{osName: "mac", arch: "arm64", goos: "darwin", name: "terraformer-aws-darwin-arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.osName+"/"+tt.arch, func(t *testing.T) {
			target, err := newBuildTarget(provider, tt.osName, tt.arch, outputDir)
			if err != nil {
				t.Fatalf("newBuildTarget() error = %v", err)
			}
			if target.GOOS != tt.goos {
				t.Fatalf("GOOS = %q, want %q", target.GOOS, tt.goos)
			}
			if target.BinaryName != tt.name {
				t.Fatalf("BinaryName = %q, want %q", target.BinaryName, tt.name)
			}
			if target.OutputPath != filepath.Join(outputDir, tt.name) {
				t.Fatalf("OutputPath = %q, want %q", target.OutputPath, filepath.Join(outputDir, tt.name))
			}
		})
	}
}

func TestCopySourceTreeSkipsReleaseOutputs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "go.mod"), "module example.com/test\n")
	writeFile(t, filepath.Join(src, "cmd", "root.go"), "package cmd\n")
	writeFile(t, filepath.Join(src, ".goreleaser-extra", "provider-binaries", "terraformer-aws-linux-amd64"), "binary")
	writeFile(t, filepath.Join(src, "dist", "old"), "artifact")
	writeFile(t, filepath.Join(src, "cmd", "tmp", "provider_cmd_google.go"), "package cmd\n")

	if err := copySourceTree(src, dst); err != nil {
		t.Fatalf("copySourceTree() error = %v", err)
	}

	assertExists(t, filepath.Join(dst, "go.mod"))
	assertExists(t, filepath.Join(dst, "cmd", "root.go"))
	assertNotExists(t, filepath.Join(dst, ".goreleaser-extra"))
	assertNotExists(t, filepath.Join(dst, "dist"))
	assertNotExists(t, filepath.Join(dst, "cmd", "tmp"))
}

func TestPrepareProviderWorkspaceMutatesOnlyWorkspaceAndCleansUp(t *testing.T) {
	src := t.TempDir()
	workspace := t.TempDir()
	rootCode := strings.Join([]string{
		"package cmd",
		"newCmdAwsImporter,",
		"newCmdGoogleImporter,",
		"newAWSProvider,",
		"newGoogleProvider,",
		"",
	}, "\n")
	writeFile(t, filepath.Join(src, "cmd", "root.go"), rootCode)
	writeFile(t, filepath.Join(src, "cmd", "provider_cmd_aws.go"), "package cmd\n")
	writeFile(t, filepath.Join(src, "cmd", "provider_cmd_google.go"), "package cmd\n")

	if err := copySourceTree(src, workspace); err != nil {
		t.Fatalf("copySourceTree() error = %v", err)
	}
	providers, err := enumerateProviders(filepath.Join(workspace, "cmd"))
	if err != nil {
		t.Fatalf("enumerateProviders() error = %v", err)
	}
	selected := providerCommand{Name: "aws", File: "provider_cmd_aws.go"}

	cleanup, err := prepareProviderWorkspace(workspace, selected, providers)
	if err != nil {
		t.Fatalf("prepareProviderWorkspace() error = %v", err)
	}

	assertExists(t, filepath.Join(workspace, "cmd", "provider_cmd_aws.go"))
	assertNotExists(t, filepath.Join(workspace, "cmd", "provider_cmd_google.go"))
	assertExists(t, filepath.Join(workspace, "cmd", "tmp", "provider_cmd_google.go"))

	prunedRoot := readFile(t, filepath.Join(workspace, "cmd", "root.go"))
	if strings.Contains(prunedRoot, "// newCmdAwsImporter") || strings.Contains(prunedRoot, "// newAWSProvider") {
		t.Fatalf("selected provider registrations were commented:\n%s", prunedRoot)
	}
	if !strings.Contains(prunedRoot, "// newCmdGoogleImporter") || !strings.Contains(prunedRoot, "// newGoogleProvider") {
		t.Fatalf("removed provider registrations were not commented:\n%s", prunedRoot)
	}

	if got := readFile(t, filepath.Join(src, "cmd", "root.go")); got != rootCode {
		t.Fatalf("source root.go was mutated:\n%s", got)
	}
	assertExists(t, filepath.Join(src, "cmd", "provider_cmd_google.go"))

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}
	if got := readFile(t, filepath.Join(workspace, "cmd", "root.go")); got != rootCode {
		t.Fatalf("workspace root.go after cleanup = %q, want original", got)
	}
	assertExists(t, filepath.Join(workspace, "cmd", "provider_cmd_google.go"))
	assertNotExists(t, filepath.Join(workspace, "cmd", "tmp"))
}

func providerNames(providers []providerCommand) []string {
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		names = append(names, provider.Name)
	}
	return names
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(content)
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("%q exists, want absent", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}
