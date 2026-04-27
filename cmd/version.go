package cmd

import (
	"fmt"

	terraformerVersion "github.com/chenrui333/terraformer/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Terraformer",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("Terraformer " + terraformerVersion.Version)
	},
}
