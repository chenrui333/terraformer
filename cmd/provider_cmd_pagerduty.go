// Copyright 2020 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cmd

import (
	pagerduty_terraforming "github.com/GoogleCloudPlatform/terraformer/providers/pagerduty"

	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"
	"github.com/spf13/cobra"
)

func newCmdpagerdutyImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pagerduty",
		Short: "Import current state to Terraform configuration from PagerDuty",
		Long:  "Import current state to Terraform configuration from PagerDuty",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := newPagerDutyProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newPagerDutyProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "teams", "pagerduty_team=id1:id2:id3")
	return cmd
}

func newPagerDutyProvider() terraform_utils.ProviderGenerator {
	return &pagerduty_terraforming.PagerDutyProvider{}
}
