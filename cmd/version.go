package cmd

import (
	"fmt"

	terraformerVersion "github.com/chenrui333/terraformer/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Terraformer",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Terraformer " + terraformerVersion.Version)
	},
}
