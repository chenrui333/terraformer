// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"os"

	logzio_terraforming "github.com/chenrui333/terraformer/providers/logzio"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

const (
	defaultBaseURL = "https://api.logz.io"
)

func newCmdLogzioImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logzio",
		Short: "Import current state to Terraform configuration from Logz.io",
		Long:  "Import current state to Terraform configuration from Logz.io",
		RunE: func(_ *cobra.Command, _ []string) error {
			token := os.Getenv("LOGZIO_API_TOKEN")
			if len(token) == 0 {
				return errors.New("API Token for Logz.io must be set through `LOGZIO_API_TOKEN` env var")
			}
			baseURL := os.Getenv("LOGZIO_BASE_URL")
			if len(baseURL) == 0 {
				baseURL = defaultBaseURL
			}

			provider := newLogzioProvider()
			err := Import(provider, options, []string{token, baseURL})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newLogzioProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "repository", "alert=id1:id2:id4")
	return cmd
}

func newLogzioProvider() terraformutils.ProviderGenerator {
	return &logzio_terraforming.LogzioProvider{}
}
