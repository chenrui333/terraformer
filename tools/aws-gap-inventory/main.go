// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	opts := options{}
	flag.StringVar(&opts.awsDir, "aws-dir", "providers/aws", "path to the Terraformer AWS provider source directory")
	flag.StringVar(&opts.docsPath, "docs", "docs/aws.md", "path to the AWS provider docs")
	flag.StringVar(&opts.format, "format", "markdown", "output format: markdown or json")
	flag.StringVar(&opts.providerSchema, "provider-schema", "", "optional Terraform AWS provider schema JSON from terraform providers schema -json")
	flag.StringVar(&opts.skipListPath, "skip-list", "providers/aws/unsupported_resources.json", "path to the AWS unsupported resource skip-list")
	flag.Parse()

	inv, err := buildInventory(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch opts.format {
	case "json":
		err = writeJSON(os.Stdout, inv)
	case "markdown":
		err = writeMarkdown(os.Stdout, inv, opts.providerSchema)
	default:
		err = fmt.Errorf("unsupported format %q", opts.format)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
