// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"
	"strings"

	github_terraforming "github.com/chenrui333/terraformer/providers/github"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdGithubImporter(options ImportOptions) *cobra.Command {
	token := ""
	baseURL := ""
	owner := []string{}
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Import current state to Terraform configuration from GitHub",
		Long:  "Import current state to Terraform configuration from GitHub",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, organization := range owner {
				provider := newGitHubProvider()
				options.PathPattern = originalPathPattern
				options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}", "{provider}/"+organization)
				log.Println(provider.GetName() + " importing organization " + organization)
				err := Import(provider, options, []string{organization, token, baseURL})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newGitHubProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "repository", "repository=id1:id2:id4")
	cmd.PersistentFlags().StringVarP(&token, "token", "t", "", "YOUR_GITHUB_TOKEN or env param GITHUB_TOKEN")
	cmd.PersistentFlags().StringSliceVarP(&owner, "owner", "", []string{}, "")
	cmd.PersistentFlags().StringVarP(&baseURL, "base-url", "", "", "")
	return cmd
}

func newGitHubProvider() terraformutils.ProviderGenerator {
	return &github_terraforming.GithubProvider{}
}
