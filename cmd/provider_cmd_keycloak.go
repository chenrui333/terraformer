// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log"
	"os"
	"strconv"
	"strings"

	keycloak_terraforming "github.com/chenrui333/terraformer/providers/keycloak"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

const (
	defaultKeycloakEndpoint              = "https://localhost:8443"
	defaultKeycloakBasePath              = "" // Override with `export KEYCLOAK_BASE_PATH=/auth` for the legacy version of Keycloak.
	defaultKeycloakRealm                 = "master"
	defaultKeycloakClientTimeout         = int64(30)
	defaultKeycloakTLSInsecureSkipVerify = false
	defaultRedHatSSO                     = false
)

func newCmdKeycloakImporter(options ImportOptions) *cobra.Command {
	targets := []string{}
	cmd := &cobra.Command{
		Use:   "keycloak",
		Short: "Import current state to Terraform configuration from Keycloak",
		Long:  "Import current state to Terraform configuration from Keycloak",
		RunE: func(_ *cobra.Command, _ []string) error {
			url := os.Getenv("KEYCLOAK_URL")
			if len(url) == 0 {
				url = defaultKeycloakEndpoint
			}
			basePath, ok := os.LookupEnv("KEYCLOAK_BASE_PATH")
			if !ok {
				basePath = defaultKeycloakBasePath
			}
			redHatSSO, err := strconv.ParseBool(os.Getenv("RED_HAT_SSO"))
			if err != nil {
				redHatSSO = defaultRedHatSSO
			}
			clientID := os.Getenv("KEYCLOAK_CLIENT_ID")
			clientSecret := os.Getenv("KEYCLOAK_CLIENT_SECRET")
			realm := os.Getenv("KEYCLOAK_REALM")
			if len(realm) == 0 {
				realm = defaultKeycloakRealm
			}
			clientTimeout, err := strconv.ParseInt(os.Getenv("KEYCLOAK_CLIENT_TIMEOUT"), 10, 64)
			if err != nil {
				clientTimeout = defaultKeycloakClientTimeout
			}
			tlsInsecureSkipVerify, err := strconv.ParseBool(os.Getenv("KEYCLOAK_TLS_INSECURE_SKIP_VERIFY"))
			if err != nil {
				tlsInsecureSkipVerify = defaultKeycloakTLSInsecureSkipVerify
			}
			caCert := os.Getenv("KEYCLOAK_CACERT")
			if len(caCert) == 0 {
				caCert = "-"
			}
			if len(targets) > 0 {
				originalPathPattern := options.PathPattern
				for _, target := range targets {
					provider := newKeycloakProvider()
					log.Println(provider.GetName() + " importing realm " + target)
					options.PathPattern = originalPathPattern
					options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}", "{provider}/"+target)
					err := Import(provider, options, []string{url, basePath, clientID, clientSecret, realm, strconv.FormatInt(clientTimeout, 10), caCert, strconv.FormatBool(tlsInsecureSkipVerify), strconv.FormatBool(redHatSSO), target})
					if err != nil {
						return err
					}
				}
			} else {
				provider := newKeycloakProvider()
				log.Println(provider.GetName() + " importing all realms")
				err := Import(provider, options, []string{url, basePath, clientID, clientSecret, realm, strconv.FormatInt(clientTimeout, 10), caCert, strconv.FormatBool(tlsInsecureSkipVerify), strconv.FormatBool(redHatSSO), "-"})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newKeycloakProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "realms", "type=id1:id2:id4")
	cmd.PersistentFlags().StringSliceVarP(&targets, "targets", "", []string{}, "")
	return cmd
}

func newKeycloakProvider() terraformutils.ProviderGenerator {
	return &keycloak_terraforming.KeycloakProvider{}
}
