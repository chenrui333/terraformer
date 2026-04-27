// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package main

import (
	"log"
	"os"
	"os/exec"
	"sort"

	"github.com/chenrui333/terraformer/cmd"
	"github.com/chenrui333/terraformer/terraformutils"

	commercetools_terraforming "github.com/chenrui333/terraformer/providers/commercetools"
)

const command = "terraform init && terraform plan"

func main() {
	clientID := os.Getenv("CTP_CLIENT_ID")
	clientScope := os.Getenv("CTP_CLIENT_SCOPE")
	clientSecret := os.Getenv("CTP_CLIENT_SECRET")
	projectKey := os.Getenv("CTP_PROJECT_KEY")
	baseURL := "https://api.sphere.io"
	tokenURL := "https://auth.sphere.io"

	services := []string{}
	provider := &commercetools_terraforming.CommercetoolsProvider{}
	for service := range provider.GetSupportedService() {
		services = append(services, service)
	}
	sort.Strings(services)
	provider = &commercetools_terraforming.CommercetoolsProvider{
		Provider: terraformutils.Provider{},
	}
	err := cmd.Import(provider, cmd.ImportOptions{
		Resources:   services,
		PathPattern: cmd.DefaultPathPattern,
		PathOutput:  cmd.DefaultPathOutput,
		State:       "local",
		Connect:     true,
	}, []string{clientID, clientScope, clientSecret, projectKey, baseURL, tokenURL})
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	rootPath, _ := os.Getwd()
	for _, serviceName := range services {
		currentPath := cmd.Path(cmd.DefaultPathPattern, provider.GetName(), serviceName, cmd.DefaultPathOutput)
		if err := os.Chdir(currentPath); err != nil {
			log.Println(err)
			os.Exit(1)
		}
		cmd := exec.Command("sh", "-c", command)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		err := os.Chdir(rootPath)
		if err != nil {
			log.Println(err)
		}
	}
}
