// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"
	"strings"

	gitLab_terraforming "github.com/chenrui333/terraformer/providers/gitlab"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdGitLabImporter(options ImportOptions) *cobra.Command {
	token := ""
	baseURL := ""
	groups := []string{}
	cmd := &cobra.Command{
		Use:   "gitlab",
		Short: "Import current state to Terraform configuration from GitLab",
		Long:  "Import current state to Terraform configuration from GitLab",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, group := range groups {
				provider := newGitLabProvider()
				options.PathPattern = originalPathPattern
				options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}", "{provider}/"+group)
				log.Println(provider.GetName() + " importing group " + group)
				err := Import(provider, options, []string{group, token, baseURL})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newGitLabProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "repository", "repository=id1:id2:id4")
	cmd.PersistentFlags().StringVarP(&token, "token", "t", "", "YOUR_GITLAB_TOKEN or env param GITLAB_TOKEN")
	cmd.PersistentFlags().StringSliceVarP(&groups, "group", "", []string{}, "paths to groups")
	cmd.PersistentFlags().StringVarP(&baseURL, "base-url", "", "", "")
	return cmd
}

func newGitLabProvider() terraformutils.ProviderGenerator {
	return &gitLab_terraforming.GitLabProvider{}
}
