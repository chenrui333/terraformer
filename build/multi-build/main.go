//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const filePrefix = "provider_cmd_"
const fileSuffix = ".go"
const packageCmdPath = "cmd"

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

func main() {
	// provider := os.Args[1]
	outputDir := os.Getenv("TERRAFORMER_BUILD_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		log.Fatal("err:", err)
	}

	allProviders := []string{}
	files, err := os.ReadDir(packageCmdPath)
	if err != nil {
		log.Println(err)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), filePrefix) {
			providerName := strings.ReplaceAll(f.Name(), filePrefix, "")
			providerName = strings.ReplaceAll(providerName, fileSuffix, "")
			allProviders = append(allProviders, providerName)
		}
	}
	arches := envList("TERRAFORMER_BUILD_GOARCH", []string{"amd64", "arm64"})
	oses := envList("TERRAFORMER_BUILD_GOOS", []string{"linux", "windows", "mac"})
	providerFilter := map[string]bool{}
	for _, provider := range envList("TERRAFORMER_BUILD_PROVIDERS", nil) {
		providerFilter[provider] = true
	}
	for _, arch := range arches {
		for _, OS := range oses {
			for _, provider := range allProviders {
				if len(providerFilter) > 0 && !providerFilter[provider] {
					continue
				}
				GOOS := ""
				binaryName := ""
				switch OS {
				case "linux":
					GOOS = "linux"
					binaryName = "terraformer-" + provider + "-linux-" + arch
				case "windows":
					GOOS = "windows"
					binaryName = "terraformer-" + provider + "-windows-" + arch + ".exe"
				case "darwin", "mac":
					GOOS = "darwin"
					binaryName = "terraformer-" + provider + "-darwin-" + arch
				default:
					log.Fatal("err: unsupported TERRAFORMER_BUILD_GOOS value: ", OS)
				}
				log.Println("Build terraformer with "+provider+" provider...", "GOOS=", GOOS, " for GOARCH=", arch)
				deletedProvider := []string{}
				for _, f := range files {
					if strings.HasPrefix(f.Name(), filePrefix) {
						if !strings.HasPrefix(f.Name(), filePrefix+provider+fileSuffix) {
							providerName := strings.ReplaceAll(f.Name(), filePrefix, "")
							providerName = strings.ReplaceAll(providerName, fileSuffix, "")
							deletedProvider = append(deletedProvider, providerName)
						}
					}
				}
				// move files for deleted providers
				err := os.MkdirAll(packageCmdPath+"/tmp", os.ModePerm)
				if err != nil {
					log.Fatal("err:", err)
				}
				for _, provider := range deletedProvider {
					err := os.Rename(packageCmdPath+"/"+filePrefix+provider+fileSuffix, packageCmdPath+"/tmp/"+filePrefix+provider+fileSuffix)
					if err != nil {
						log.Println(err)
					}
				}
				restoreProviderFiles := func() {
					for _, provider := range deletedProvider {
						err := os.Rename(packageCmdPath+"/tmp/"+filePrefix+provider+fileSuffix, packageCmdPath+"/"+filePrefix+provider+fileSuffix)
						if err != nil {
							log.Println(err)
						}
					}
				}

				// comment deleted providers in code
				rootCode, err := os.ReadFile(packageCmdPath + "/root.go")
				if err != nil {
					restoreProviderFiles()
					log.Fatal("err:", err)
				}
				cleanupCodeAndFiles := func() {
					if err := os.WriteFile(packageCmdPath+"/root.go", rootCode, os.ModePerm); err != nil {
						log.Println(err)
					}
					restoreProviderFiles()
				}
				lines := strings.Split(string(rootCode), "\n")
				newRootCodeLines := make([]string, len(lines))
				for i, line := range lines {
					for _, provider := range deletedProvider {
						if strings.Contains(strings.ToLower(line), "newcmd"+provider+"importer") {
							line = "// " + line
						}
						if strings.Contains(strings.ToLower(line), "new"+provider+"provider") {
							line = "// " + line
						}
					}
					newRootCodeLines[i] = line
				}
				newRootCode := strings.Join(newRootCodeLines, "\n")
				err = os.WriteFile(packageCmdPath+"/root.go", []byte(newRootCode), os.ModePerm)
				if err != nil {
					cleanupCodeAndFiles()
					log.Fatal("err:", err)
				}

				// build....
				binaryPath := filepath.Join(outputDir, binaryName)
				args := []string{"build", "-v", "-o", binaryPath}
				if ldflags := os.Getenv("TERRAFORMER_LDFLAGS"); ldflags != "" {
					args = append(args, "-ldflags", ldflags)
				}
				cmd := exec.Command("go", args...)
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, "GOOS="+GOOS)
				cmd.Env = append(cmd.Env, "GOARCH="+arch)
				var outb, errb bytes.Buffer
				cmd.Stdout = &outb
				cmd.Stderr = &errb
				err = cmd.Run()
				if err != nil {
					cleanupCodeAndFiles()
					log.Fatal("err:", errb.String())
				}
				fmt.Println(outb.String())

				// revert code and files
				err = os.WriteFile(packageCmdPath+"/root.go", rootCode, os.ModePerm)
				if err != nil {
					restoreProviderFiles()
					log.Fatal("err:", err)
				}
				restoreProviderFiles()
			}
		}
	}
}
