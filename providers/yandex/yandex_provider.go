// SPDX-License-Identifier: Apache-2.0

package yandex

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

const KeyToken = "token"
const KeyFolderID = "folder_id"
const KeySaKeyFileOrContent = "sa_key_or_content"

type YandexProvider struct { //nolint
	terraformutils.Provider
	token              string
	saKeyFileOrContent string
	folderID           string
}

func (p *YandexProvider) Init(args []string) error {
	p.token = ""
	p.saKeyFileOrContent = ""
	p.folderID = ""

	token := ""
	saKeyFileOrContent := ""
	folderID := ""
	if ycToken, ok := os.LookupEnv("YC_TOKEN"); ok {
		token = ycToken
	}

	if envSaKeyFileOrContent, ok := os.LookupEnv("YC_SERVICE_ACCOUNT_KEY_FILE"); ok {
		saKeyFileOrContent = envSaKeyFileOrContent
	}

	if len(args) > 0 && args[0] != "" {
		//  first args is target folder ID
		folderID = args[0]
	} else if envFolderID := os.Getenv("YC_FOLDER_ID"); envFolderID != "" {
		folderID = envFolderID
	} else {
		return errors.New("set YC_FOLDER_ID env var")
	}
	p.token = token
	p.saKeyFileOrContent = saKeyFileOrContent
	p.folderID = folderID

	return nil
}

func (p *YandexProvider) GetName() string {
	return "yandex"
}

func (p *YandexProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (YandexProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *YandexProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"disk":     &DiskGenerator{},
		"instance": &InstanceGenerator{},
		"network":  &NetworkGenerator{},
		"subnet":   &SubnetGenerator{},
	}
}

func (p *YandexProvider) InitService(serviceName string, verbose bool) error {
	if !terraformutils.SelectProviderService(&p.Provider, p.GetSupportedService(), serviceName, verbose, p.GetName()) {
		return errors.New("yandex: " + serviceName + " not supported service")
	}
	p.Service.SetArgs(map[string]interface{}{
		KeyFolderID:           p.folderID,
		KeyToken:              p.token,
		KeySaKeyFileOrContent: p.saKeyFileOrContent,
	})
	return nil
}
