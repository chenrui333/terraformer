// SPDX-License-Identifier: Apache-2.0

package grafana

import (
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"os"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/go-cleanhttp"
)

type GrafanaService struct { //nolint
	terraformutils.Service
}

func (s *GrafanaService) buildClient() (*gapi.Client, error) {
	auth := strings.SplitN(s.Args["auth"].(string), ":", 2)
	cli := cleanhttp.DefaultClient()
	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{}

	// TLS Config
	tlsKey := s.Args["tls_key"].(string)
	tlsCert := s.Args["tls_cert"].(string)
	caCert := s.Args["ca_cert"].(string)
	insecure := s.Args["insecure_skip_verify"].(bool)

	if caCert != "" {
		ca, err := os.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		transport.TLSClientConfig.RootCAs = pool
	}

	if tlsKey != "" && tlsCert != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	if insecure {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	cli.Transport = transport
	cfg := gapi.Config{
		Client: cli,
		OrgID:  int64(s.Args["org_id"].(int)),
	}

	if len(auth) == 2 {
		cfg.BasicAuth = url.UserPassword(auth[0], auth[1])
	} else {
		cfg.APIKey = auth[0]
	}

	client, err := gapi.New(s.Args["url"].(string), cfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}
