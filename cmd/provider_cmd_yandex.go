// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log"
	"strings"

	yandex_terraforming "github.com/chenrui333/terraformer/providers/yandex"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdYandexImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yandex",
		Short: "Import current state to Terraform configuration from Yandex Cloud",
		Long:  "Import current state to Terraform configuration from Yandex Cloud",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			// iterate over provided folder_ids
			for _, folderID := range options.Projects {
				provider := newYandexProvider()
				options.PathPattern = originalPathPattern
				options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}/{service}", "{provider}/"+folderID+"/{service}")
				log.Println(provider.GetName() + " importing folder id " + folderID)
				err := Import(provider, options, []string{folderID})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newYandexProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "instance,disk", "")
	cmd.Flags().StringSliceVarP(&options.Projects, "folder_ids", "", []string{}, "folder_id_1,folder_id_2")
	_ = cmd.MarkFlagRequired("folder_ids")
	return cmd
}

func newYandexProvider() terraformutils.ProviderGenerator {
	return &yandex_terraforming.YandexProvider{}
}
