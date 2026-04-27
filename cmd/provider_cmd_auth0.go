// SPDX-License-Identifier: Apache-2.0
//
//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cmd

import (
	"errors"
	"os"

	auth0_terraforming "github.com/chenrui333/terraformer/providers/auth0"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAuth0Importer(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth0",
		Short: "Import current state to Terraform configuration from Auth0",
		Long:  "Import current state to Terraform configuration from Auth0",
		RunE: func(_ *cobra.Command, _ []string) error {
			domain := os.Getenv("AUTH0_DOMAIN")
			if len(domain) == 0 {
				return errors.New("Domain for Auth0 must be set through `AUTH0_DOMAIN` env var")
			}
			clientID := os.Getenv("AUTH0_CLIENT_ID")
			if len(clientID) == 0 {
				return errors.New("Client ID for Auht0 must be set through `AUTH0_CLIENT_ID` env var")
			}
			clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
			if len(clientSecret) == 0 {
				return errors.New("Clien Secret for Auth0 must be set through `AUTH0_CLIENT_SECRET` env var")
			}

			provider := newAuth0Provider()
			err := Import(provider, options, []string{domain, clientID, clientSecret})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newAuth0Provider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "action", "action=name1:name2:name3")
	return cmd
}

func newAuth0Provider() terraformutils.ProviderGenerator {
	return &auth0_terraforming.Auth0Provider{}
}
