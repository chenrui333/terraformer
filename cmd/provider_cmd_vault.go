// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	vault_terraforming "github.com/chenrui333/terraformer/providers/vault"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdVaultImporter(options ImportOptions) *cobra.Command {
	var token, address string
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Import current state to Terraform configuration from Vault",
		Long:  "Import current state to Terraform configuration from Vault",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newVaultProvider()
			err := Import(provider, options, []string{address, token})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newVaultProvider()))
	cmd.PersistentFlags().StringVarP(&address, "address", "a", "", "env param VAULT_ADDR")
	cmd.PersistentFlags().StringVarP(&token, "token", "t", "", "env param VAULT_TOKEN")
	baseProviderFlags(cmd.PersistentFlags(), &options, "", "")
	return cmd
}

func newVaultProvider() terraformutils.ProviderGenerator {
	return &vault_terraforming.Provider{}
}
