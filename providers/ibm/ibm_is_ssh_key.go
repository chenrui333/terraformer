// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"fmt"
	"log"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

// SSHKeyGenerator ...
type SSHKeyGenerator struct {
	IBMService
}

func (g SSHKeyGenerator) createSSHKeyResources(sshKeyID, sshKeyName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		sshKeyID,
		normalizeResourceName(sshKeyName, true),
		"ibm_is_ssh_key",
		"ibm",
		[]string{})
	return resources
}

// InitResources ...
func (g *SSHKeyGenerator) InitResources() error {
	region := g.Args["region"].(string)
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		log.Fatal("No API key set")
	}

	isURL := GetVPCEndPoint(region)
	iamURL := GetAuthEndPoint()
	vpcoptions := &vpcv1.VpcV1Options{
		URL: isURL,
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
			URL:    iamURL,
		},
	}
	vpcclient, err := vpcv1.NewVpcV1(vpcoptions)
	if err != nil {
		return err
	}
	options := &vpcv1.ListKeysOptions{}
	keys, response, err := vpcclient.ListKeys(options)
	if err != nil {
		return fmt.Errorf("error fetching SSH Keys %w\n%s", err, response)
	}

	for _, key := range keys.Keys {
		g.Resources = append(g.Resources, g.createSSHKeyResources(*key.ID, *key.Name))
	}
	return nil
}
