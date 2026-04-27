// SPDX-License-Identifier: Apache-2.0

package honeycombio

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	hnyclient "github.com/honeycombio/terraform-provider-honeycombio/client"
)

type HoneycombService struct {
	terraformutils.Service
	datasets map[string]hnyclient.Dataset
}

func (s *HoneycombService) newClient() (*hnyclient.Client, error) {
	enableDebug, _ := strconv.ParseBool(os.Getenv("HONEYCOMBIO_DEBUG"))

	client, err := hnyclient.NewClientWithConfig(&hnyclient.Config{
		APIKey:    s.GetArgs()["api_key"].(string),
		APIUrl:    s.GetArgs()["api_url"].(string),
		UserAgent: fmt.Sprintf("terraformer-honeycombio/%s", honeycombTerraformerProviderVersion),
		Debug:     enableDebug,
	})
	if err != nil {
		return client, fmt.Errorf("unable to initialize Honeycomb client: %w", err)
	}

	ctx := context.TODO()
	ds := s.GetArgs()["datasets"].([]string)
	s.datasets = make(map[string]hnyclient.Dataset)
	if len(ds) == 0 {
		// assume all datasets
		datasets, err := client.Datasets.List(ctx)
		if err != nil {
			return client, fmt.Errorf("unable to list Honeycomb datasets: %w", err)
		}
		for _, d := range datasets {
			s.datasets[d.Name] = d
		}
		if !s.isClassicEnvironment() {
			s.datasets[environmentWideDatasetSlug] = s.environmentWideDataset()
		}
	} else {
		// verify the provided datasets exist
		for _, d := range ds {
			if d == environmentWideDatasetSlug {
				if s.isClassicEnvironment() {
					return client, fmt.Errorf("%q provided as a dataset but the API key is for a Classic environment", environmentWideDatasetSlug)
				}
				s.datasets[environmentWideDatasetSlug] = s.environmentWideDataset()
				continue
			}
			ds, err := client.Datasets.Get(ctx, d)
			if err != nil {
				return client, fmt.Errorf("unable to get Honeycomb dataset %q: %w", d, err)
			}
			s.datasets[ds.Name] = *ds
		}
	}

	return client, nil
}

func (s *HoneycombService) isClassicEnvironment() bool {
	return len(s.GetArgs()["api_key"].(string)) == 32
}

const environmentWideDatasetSlug = "__all__"

func (s *HoneycombService) environmentWideDataset() hnyclient.Dataset {
	return hnyclient.Dataset{Name: environmentWideDatasetSlug, Slug: environmentWideDatasetSlug}
}
