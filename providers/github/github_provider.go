// SPDX-License-Identifier: Apache-2.0

package github

import (
	"os"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type GithubProvider struct { //nolint
	terraformutils.Provider
	owner          string
	token          string
	baseURL        string
	appID          int64
	installationID int64
	pem            string
}

func (p GithubProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p GithubProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{
		"provider": map[string]interface{}{
			"github": map[string]interface{}{
				"owner": p.owner,
			},
		},
	}
}

func (p *GithubProvider) GetConfig() cty.Value {
	if p.appID != 0 && p.installationID != 0 && p.pem != "" {
		return cty.ObjectVal(map[string]cty.Value{
			"owner": cty.StringVal(p.owner),
			"app_auth": cty.ListVal(
				[]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":              cty.NumberIntVal(p.appID),
						"installation_id": cty.NumberIntVal(p.installationID),
						"pem_file":        cty.StringVal(p.pem),
					}),
				},
			),
		})
	}
	return cty.ObjectVal(map[string]cty.Value{
		"owner":    cty.StringVal(p.owner),
		"token":    cty.StringVal(p.token),
		"base_url": cty.StringVal(p.baseURL),
	})
}

// Init GithubProvider with owner
func (p *GithubProvider) Init(args []string) error {
	if len(args) < 1 || args[0] == "" {
		return errors.New("github: owner is required")
	}

	p.owner = args[0]
	p.token = ""
	p.baseURL = githubDefaultURL
	p.appID = 0
	p.installationID = 0
	p.pem = ""

	if appIDValue := os.Getenv("GITHUB_APP_ID"); appIDValue != "" {
		appID, err := strconv.ParseInt(appIDValue, 10, 64)
		if err != nil {
			return err
		}
		p.appID = appID
	}
	if installationIDValue := os.Getenv("GITHUB_APP_INSTALLATION_ID"); installationIDValue != "" {
		installationID, err := strconv.ParseInt(installationIDValue, 10, 64)
		if err != nil {
			return err
		}
		p.installationID = installationID
	}
	if pem := os.Getenv("GITHUB_APP_PEM_FILE"); pem != "" {
		p.pem = strings.ReplaceAll(pem, `\n`, "\n")
	}

	if len(args) > 1 && args[1] != "" {
		p.token = args[1]
	} else {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" && (p.appID == 0 || p.installationID == 0 || p.pem == "") {
			return errors.New("token requirement")
		}
		p.token = token
	}
	if len(args) > 2 && args[2] != "" {
		p.baseURL = args[2]
	}
	return nil
}

func (p *GithubProvider) GetName() string {
	return "github"
}

func (p *GithubProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New(p.GetName() + ": " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		"owner":           p.owner,
		"token":           p.token,
		"base_url":        p.baseURL,
		"app_id":          p.appID,
		"installation_id": p.installationID,
		"pem":             p.pem,
	})
	return nil
}

// GetSupportedService return map of support service for Github
func (p *GithubProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"members":               &MembersGenerator{},
		"organization":          &OrganizationGenerator{},
		"organization_blocks":   &OrganizationBlockGenerator{},
		"organization_projects": &OrganizationProjectGenerator{},
		"organization_webhooks": &OrganizationWebhooksGenerator{},
		"repositories":          &RepositoriesGenerator{},
		"teams":                 &TeamsGenerator{},
		"user_ssh_keys":         &UserSSHKeyGenerator{},
	}
}
