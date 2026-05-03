// SPDX-License-Identifier: Apache-2.0

package digitalocean

import (
	"errors"
	"os"

	"github.com/chenrui333/terraformer/terraformutils"
)

type DigitalOceanProvider struct { //nolint
	terraformutils.Provider
	token string
}

func (p *DigitalOceanProvider) Init(_ []string) error {
	p.token = ""

	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if token == "" {
		return errors.New("set DIGITALOCEAN_TOKEN env var")
	}
	p.token = token

	return nil
}

func (p *DigitalOceanProvider) GetName() string {
	return "digitalocean"
}

func (p *DigitalOceanProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (DigitalOceanProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p *DigitalOceanProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	return map[string]terraformutils.ServiceGenerator{
		"cdn":                &CDNGenerator{},
		"certificate":        &CertificateGenerator{},
		"database_cluster":   &DatabaseClusterGenerator{},
		"domain":             &DomainGenerator{},
		"droplet":            &DropletGenerator{},
		"droplet_snapshot":   &DropletSnapshotGenerator{},
		"firewall":           &FirewallGenerator{},
		"floating_ip":        &FloatingIPGenerator{},
		"kubernetes_cluster": &KubernetesClusterGenerator{},
		"loadbalancer":       &LoadBalancerGenerator{},
		"project":            &ProjectGenerator{},
		"ssh_key":            &SSHKeyGenerator{},
		"tag":                &TagGenerator{},
		"volume":             &VolumeGenerator{},
		"volume_snapshot":    &VolumeSnapshotGenerator{},
		"vpc":                &VPCGenerator{},
	}
}

func (p *DigitalOceanProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	var isSupported bool
	if _, isSupported = p.GetSupportedService()[serviceName]; !isSupported {
		return errors.New("digitalocean: " + serviceName + " not supported service")
	}
	p.Service = p.GetSupportedService()[serviceName]
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	p.Service.SetArgs(map[string]interface{}{
		"token": p.token,
	})
	return nil
}
