// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/chenrui333/terraformer/cmd"
)

type TerraformerWriter struct {
	io.Writer
}

func (t TerraformerWriter) Write(p []byte) (n int, err error) {
	if !strings.Contains(string(p), "[TRACE]") && !strings.Contains(string(p), "[DEBUG]") { // hide TF GRPC client log messages
		return os.Stdout.Write(p)
	}
	return len(p), nil
}

func main() {
	log.SetOutput(TerraformerWriter{})
	err := cmd.Execute()
	cmd.FinalizeReport()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if cmd.HasReportFailures() {
		os.Exit(1)
	}
}
