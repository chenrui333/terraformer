//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package main

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const filePrefix = "provider_cmd_"
const fileSuffix = ".go"
const packageCmdPath = "cmd"

type providerCommand struct {
	Name string
	File string
}

type buildTarget struct {
	Provider   providerCommand
	GOOS       string
	GOARCH     string
	BinaryName string
	OutputPath string
}

type buildOptions struct {
	RepoRoot       string
	OutputDir      string
	Arches         []string
	OSes           []string
	ProviderFilter []string
	LDFlags        string
	DryRun         bool
}

func envList(name string, defaults []string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaults
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	items := make([]string, 0, len(fields))
	for _, field := range fields {
		item := strings.TrimSpace(field)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func loadOptions() (buildOptions, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return buildOptions{}, fmt.Errorf("get working directory: %w", err)
	}

	outputDir := os.Getenv("TERRAFORMER_BUILD_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "."
	}
	outputDir, err = filepath.Abs(outputDir)
	if err != nil {
		return buildOptions{}, fmt.Errorf("resolve output directory: %w", err)
	}

	return buildOptions{
		RepoRoot:       repoRoot,
		OutputDir:      outputDir,
		Arches:         envList("TERRAFORMER_BUILD_GOARCH", []string{"amd64", "arm64"}),
		OSes:           envList("TERRAFORMER_BUILD_GOOS", []string{"linux", "windows", "mac"}),
		ProviderFilter: envList("TERRAFORMER_BUILD_PROVIDERS", nil),
		LDFlags:        os.Getenv("TERRAFORMER_LDFLAGS"),
		DryRun:         envBool("TERRAFORMER_BUILD_DRY_RUN"),
	}, nil
}

func main() {
	options, err := loadOptions()
	if err != nil {
		log.Fatal("err:", err)
	}
	if err := run(options); err != nil {
		log.Fatal("err:", err)
	}
}

func run(options buildOptions) (err error) {
	if err := os.MkdirAll(options.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("create output directory %s: %w", options.OutputDir, err)
	}

	var providers []providerCommand
	if err := timed("provider enumeration", func() error {
		var enumerateErr error
		providers, enumerateErr = enumerateProviders(filepath.Join(options.RepoRoot, packageCmdPath))
		return enumerateErr
	}); err != nil {
		return err
	}

	selectedProviders, err := selectProviders(providers, options.ProviderFilter)
	if err != nil {
		return err
	}

	targets, err := buildTargets(selectedProviders, options.OSes, options.Arches, options.OutputDir)
	if err != nil {
		return err
	}
	log.Printf("Selected %d provider command(s), %d build target(s)", len(selectedProviders), len(targets))

	if options.DryRun {
		for _, target := range targets {
			log.Printf("dry-run target provider=%s GOOS=%s GOARCH=%s output=%s", target.Provider.Name, target.GOOS, target.GOARCH, target.OutputPath)
		}
		return nil
	}

	var workspace string
	if err := timed("setup temp source tree", func() error {
		var setupErr error
		workspace, setupErr = createSourceWorkspace(options.RepoRoot)
		return setupErr
	}); err != nil {
		return err
	}
	defer func() {
		cleanupErr := timed("cleanup temp source tree", func() error {
			return os.RemoveAll(workspace)
		})
		if err == nil && cleanupErr != nil {
			err = cleanupErr
		}
	}()

	for _, target := range targets {
		phase := fmt.Sprintf("provider build %s %s/%s", target.Provider.Name, target.GOOS, target.GOARCH)
		if err := timed(phase, func() error {
			return buildProviderBinary(workspace, providers, target, options.LDFlags)
		}); err != nil {
			return err
		}
	}

	return nil
}

func timed(name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Round(time.Millisecond)
	if err != nil {
		log.Printf("timing phase=%q status=failed duration=%s", name, duration)
		return err
	}
	log.Printf("timing phase=%q status=success duration=%s", name, duration)
	return nil
}

func enumerateProviders(cmdDir string) ([]providerCommand, error) {
	files, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read command directory %s: %w", cmdDir, err)
	}

	providers := make([]providerCommand, 0, len(files))
	for _, f := range files {
		if !f.Type().IsRegular() || !strings.HasPrefix(f.Name(), filePrefix) || !strings.HasSuffix(f.Name(), fileSuffix) {
			continue
		}
		providerName := strings.TrimSuffix(strings.TrimPrefix(f.Name(), filePrefix), fileSuffix)
		providers = append(providers, providerCommand{Name: providerName, File: f.Name()})
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no provider command files found in %s", cmdDir)
	}
	return providers, nil
}

func selectProviders(providers []providerCommand, filter []string) ([]providerCommand, error) {
	if len(filter) == 0 {
		return providers, nil
	}

	requested := make(map[string]bool, len(filter))
	for _, provider := range filter {
		requested[provider] = true
	}

	selected := make([]providerCommand, 0, len(filter))
	found := make(map[string]bool, len(filter))
	for _, provider := range providers {
		if requested[provider.Name] {
			selected = append(selected, provider)
			found[provider.Name] = true
		}
	}

	missing := make([]string, 0)
	for provider := range requested {
		if !found[provider] {
			missing = append(missing, provider)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("provider command filter includes unknown provider(s): %s", strings.Join(missing, ", "))
	}

	return selected, nil
}

func buildTargets(providers []providerCommand, oses []string, arches []string, outputDir string) ([]buildTarget, error) {
	targets := make([]buildTarget, 0, len(providers)*len(oses)*len(arches))
	for _, arch := range arches {
		for _, osName := range oses {
			for _, provider := range providers {
				target, err := newBuildTarget(provider, osName, arch, outputDir)
				if err != nil {
					return nil, err
				}
				targets = append(targets, target)
			}
		}
	}
	return targets, nil
}

func newBuildTarget(provider providerCommand, osName string, arch string, outputDir string) (buildTarget, error) {
	goos := ""
	binaryName := ""
	switch osName {
	case "linux":
		goos = "linux"
		binaryName = "terraformer-" + provider.Name + "-linux-" + arch
	case "windows":
		goos = "windows"
		binaryName = "terraformer-" + provider.Name + "-windows-" + arch + ".exe"
	case "darwin", "mac":
		goos = "darwin"
		binaryName = "terraformer-" + provider.Name + "-darwin-" + arch
	default:
		return buildTarget{}, fmt.Errorf("unsupported TERRAFORMER_BUILD_GOOS value: %s", osName)
	}

	return buildTarget{
		Provider:   provider,
		GOOS:       goos,
		GOARCH:     arch,
		BinaryName: binaryName,
		OutputPath: filepath.Join(outputDir, binaryName),
	}, nil
}

func createSourceWorkspace(repoRoot string) (string, error) {
	workspace, err := os.MkdirTemp("", "terraformer-provider-build-*")
	if err != nil {
		return "", fmt.Errorf("create temp source tree: %w", err)
	}
	if err := copySourceTree(repoRoot, workspace); err != nil {
		_ = os.RemoveAll(workspace)
		return "", err
	}
	return workspace, nil
}

func copySourceTree(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() && skipSourceDir(rel) {
			return filepath.SkipDir
		}

		dst := filepath.Join(dstRoot, rel)
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		switch {
		case d.IsDir():
			return os.MkdirAll(dst, info.Mode().Perm())
		case info.Mode().Type() == 0:
			return copyFile(path, dst, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", path, err)
			}
			if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
				return err
			}
			return os.Symlink(target, dst)
		default:
			return nil
		}
	})
}

func skipSourceDir(rel string) bool {
	switch rel {
	case ".git", ".goreleaser-extra", "dist", filepath.Join(packageCmdPath, "tmp"):
		return true
	default:
		return false
	}
}

func copyFile(src string, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dst, err)
	}
	return nil
}

func buildProviderBinary(workspace string, providers []providerCommand, target buildTarget, ldflags string) (err error) {
	cleanup, err := prepareProviderWorkspace(workspace, target.Provider, providers)
	if err != nil {
		return err
	}
	defer func() {
		cleanupErr := cleanup()
		if err == nil && cleanupErr != nil {
			err = cleanupErr
		}
	}()

	log.Println("Build terraformer with "+target.Provider.Name+" provider...", "GOOS=", target.GOOS, " for GOARCH=", target.GOARCH)
	args := []string{"build", "-v", "-o", target.OutputPath}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = workspace
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOOS="+target.GOOS, "GOARCH="+target.GOARCH)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s: %w: %s", target.BinaryName, err, errb.String())
	}
	fmt.Print(outb.String())
	return nil
}

func prepareProviderWorkspace(workspace string, selected providerCommand, providers []providerCommand) (func() error, error) {
	cmdDir := filepath.Join(workspace, packageCmdPath)
	selectedPath := filepath.Join(cmdDir, selected.File)
	if _, err := os.Stat(selectedPath); err != nil {
		return nil, fmt.Errorf("selected provider command %s is missing at %s: %w", selected.Name, selectedPath, err)
	}

	tmpDir := filepath.Join(cmdDir, "tmp")
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create provider temp dir: %w", err)
	}

	movedProviders := make([]providerCommand, 0, len(providers)-1)
	for _, provider := range providers {
		if provider.Name == selected.Name {
			continue
		}
		src := filepath.Join(cmdDir, provider.File)
		dst := filepath.Join(tmpDir, provider.File)
		if err := os.Rename(src, dst); err != nil {
			return nil, fmt.Errorf("move provider command %s out of build package: %w", provider.File, err)
		}
		movedProviders = append(movedProviders, provider)
	}

	rootPath := filepath.Join(cmdDir, "root.go")
	rootCode, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rootPath, err)
	}
	rootMode := fs.FileMode(0o644)
	if info, err := os.Stat(rootPath); err == nil {
		rootMode = info.Mode().Perm()
	}

	newRootCode := pruneRootProviderRegistrations(string(rootCode), movedProviders)
	if err := os.WriteFile(rootPath, []byte(newRootCode), rootMode); err != nil {
		return nil, fmt.Errorf("write pruned root.go: %w", err)
	}

	cleanup := func() error {
		var errs []string
		if err := os.WriteFile(rootPath, rootCode, rootMode); err != nil {
			errs = append(errs, err.Error())
		}
		for _, provider := range movedProviders {
			src := filepath.Join(tmpDir, provider.File)
			dst := filepath.Join(cmdDir, provider.File)
			if err := os.Rename(src, dst); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if err := os.Remove(tmpDir); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err.Error())
		}
		if len(errs) > 0 {
			return fmt.Errorf("cleanup provider workspace: %s", strings.Join(errs, "; "))
		}
		return nil
	}

	return cleanup, nil
}

func pruneRootProviderRegistrations(rootCode string, removedProviders []providerCommand) string {
	lines := strings.Split(rootCode, "\n")
	newRootCodeLines := make([]string, len(lines))
	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		for _, provider := range removedProviders {
			if strings.Contains(lowerLine, "newcmd"+provider.Name+"importer") || strings.Contains(lowerLine, "new"+provider.Name+"provider") {
				line = "// " + line
				break
			}
		}
		newRootCodeLines[i] = line
	}
	return strings.Join(newRootCodeLines, "\n")
}
