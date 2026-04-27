// SPDX-License-Identifier: Apache-2.0
//
//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cmd

import (
	"errors"
	"os"

	okta_terraforming "github.com/chenrui333/terraformer/providers/okta"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdOktaImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "okta",
		Short: "Import current State to terraform configuration from okta",
		Long:  "Import current State to terraform configuration from okta",
		RunE: func(_ *cobra.Command, _ []string) error {
			token := os.Getenv("OKTA_API_TOKEN")
			if len(token) == 0 {
				return errors.New("API Token for Okta must be set through `OKTA_API_TOKEN` env var")
			}
			baseURL := os.Getenv("OKTA_BASE_URL")
			if len(baseURL) == 0 {
				return errors.New("Base URL for Okta must be set through `OKTA_BASE_URL` env var")
			}
			orgName := os.Getenv("OKTA_ORG_NAME")
			if len(orgName) == 0 {
				return errors.New("Org Name for Okta must be set through `OKTA_ORG_NAME` env var")
			}

			provider := newOktaProvider()
			err := Import(provider, options, []string{orgName, token, baseURL})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newOktaProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "user", "okta_user=user1:user2:user3")
	return cmd
}

func newOktaProvider() terraformutils.ProviderGenerator {
	return &okta_terraforming.OktaProvider{}
}
