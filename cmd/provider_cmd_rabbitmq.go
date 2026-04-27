// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	rabbitmq_terraforming "github.com/chenrui333/terraformer/providers/rabbitmq"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

const (
	defaultRabbitMQEndpoint = "http://localhost:15672"
)

func newCmdRabbitMQImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rabbitmq",
		Short: "Import current state to Terraform configuration from RabbitMQ",
		Long:  "Import current state to Terraform configuration from RabbitMQ",
		RunE: func(_ *cobra.Command, _ []string) error {
			endpoint := os.Getenv("RABBITMQ_SERVER_URL")
			if len(endpoint) == 0 {
				endpoint = defaultRabbitMQEndpoint
			}
			username := os.Getenv("RABBITMQ_USERNAME")
			password := os.Getenv("RABBITMQ_PASSWORD")
			provider := newRabbitMQProvider()
			err := Import(provider, options, []string{endpoint, username, password})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newRabbitMQProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "vhosts", "type=id1:id2:id4")
	return cmd
}

func newRabbitMQProvider() terraformutils.ProviderGenerator {
	return &rabbitmq_terraforming.RBTProvider{}
}
