package connectivity

import (
	"strings"

	"github.com/labd/commercetools-go-sdk/platform"
	"golang.org/x/oauth2/clientcredentials"
)

type Client struct {
	*platform.Client
	projectKey string
}

func (c *Config) NewClient() (*Client, error) {
	client, err := platform.NewClient(&platform.ClientConfig{
		URL: c.BaseURL,
		Credentials: &clientcredentials.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			Scopes:       strings.Split(c.ClientScope, " "),
			TokenURL:     c.TokenURL,
		},
		UserAgent: "terraformer",
	})
	if err != nil {
		return nil, err
	}

	return &Client{Client: client, projectKey: c.ProjectKey}, nil
}

func (c *Client) Project() *platform.ByProjectKeyRequestBuilder {
	return c.WithProjectKey(c.projectKey)
}
