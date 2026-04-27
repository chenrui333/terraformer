// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"os/exec"
	"sort"

	"github.com/chenrui333/terraformer/cmd"
	"github.com/chenrui333/terraformer/terraformutils"

	rabbitmq_terraforming "github.com/chenrui333/terraformer/providers/rabbitmq"
)

const command = "terraform init && terraform plan"

func main() {
	endpoint := os.Getenv("RABBITMQ_SERVER_URL")
	username := os.Getenv("RABBITMQ_USERNAME")
	password := os.Getenv("RABBITMQ_PASSWORD")

	services := []string{}
	provider := &rabbitmq_terraforming.RBTProvider{}
	for service := range provider.GetSupportedService() {
		services = append(services, service)
	}
	sort.Strings(services)
	provider = &rabbitmq_terraforming.RBTProvider{
		Provider: terraformutils.Provider{},
	}
	err := cmd.Import(provider, cmd.ImportOptions{
		Resources:   services,
		PathPattern: cmd.DefaultPathPattern,
		PathOutput:  cmd.DefaultPathOutput,
		State:       "local",
		Connect:     true,
	}, []string{endpoint, username, password})
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
