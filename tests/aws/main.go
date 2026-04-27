// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/chenrui333/terraformer/cmd"
	aws_terraforming "github.com/chenrui333/terraformer/providers/aws"
)

func main() {
	tCommand := cmd.NewCmdRoot()
	pathPattern := "{output}/{provider}/"
	tCommand.SetArgs([]string{
		"import",
		"aws",
		"--regions=ap-southeast-1",
		"--resources=ssm",
		"--profile=personal",
		"--verbose",
		"--compact",
		"--path-pattern=" + pathPattern,
	})
	start := time.Now()
	if err := tCommand.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	log.Printf("Importing took %s", time.Since(start))
	start = time.Now()
	runTerraform(pathPattern)
	log.Printf("Terraform init + plan took %s", time.Since(start))
}

func runTerraform(pathPattern string) {
	rootPath, _ := os.Getwd()
	provider := &aws_terraforming.AWSProvider{}

	currentPath := cmd.Path(pathPattern, provider.GetName(), "", cmd.DefaultPathOutput)
	if err := os.Chdir(currentPath); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	tfCmd := exec.Command("sh", "-c", "terraform init && terraform plan")
	tfCmd.Stdout = os.Stdout
	tfCmd.Stderr = os.Stderr
	err := tfCmd.Run()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	err = os.Chdir(rootPath)
	if err != nil {
		log.Println(err)
	}
}
