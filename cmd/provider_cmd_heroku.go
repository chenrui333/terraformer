// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package cmd

import (
	"errors"
	"os"

	heroku_terraforming "github.com/chenrui333/terraformer/providers/heroku"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdHerokuImporter(options ImportOptions) *cobra.Command {
	var apiKey, team string

	cmd := &cobra.Command{
		Use:   "heroku",
		Short: "Import current state to Terraform configuration from Heroku",
		Long:  "Import current state to Terraform configuration from Heroku",
		RunE: func(_ *cobra.Command, _ []string) error {
			if apiKey = os.Getenv("HEROKU_API_KEY"); apiKey == "" {
				return errors.New("Requires HEROKU_API_KEY env var")
			}
			provider := newHerokuProvider()
			err := Import(provider, options, []string{apiKey, team})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newHerokuProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "app,addon", "app=ID")
	cmd.PersistentFlags().StringVarP(&team, "team", "", "", "")
	return cmd
}

func newHerokuProvider() terraformutils.ProviderGenerator {
	return &heroku_terraforming.HerokuProvider{}
}
