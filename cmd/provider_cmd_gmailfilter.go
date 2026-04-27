// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	gmailfilter_terraforming "github.com/chenrui333/terraformer/providers/gmailfilter"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdGmailfilterImporter(options ImportOptions) *cobra.Command {
	var creds, impersonatedUserEmail string
	cmd := &cobra.Command{
		Use:   "gmailfilter",
		Short: "Import current state to Terraform configuration from Gmail",
		Long:  "Import current state to Terraform configuration from Gmail",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newGmailfilterProvider()
			err := Import(provider, options, []string{
				creds,
				impersonatedUserEmail,
			})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newGmailfilterProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "label,filter", "label=name1:name2")
	cmd.PersistentFlags().StringVarP(&creds, "credentials", "", "", "/path/to/client_secret.json")
	cmd.PersistentFlags().StringVarP(&impersonatedUserEmail, "email", "", "", "foobar@example.com")
	return cmd
}

func newGmailfilterProvider() terraformutils.ProviderGenerator {
	return &gmailfilter_terraforming.GmailfilterProvider{}
}
