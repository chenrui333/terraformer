package honeycombio

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
)

type QueryAnnotationGenerator struct {
	HoneycombService
}

func (g *QueryAnnotationGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to initialize Honeycomb client: %v", err)
	}

	boards, err := client.Boards.List(context.TODO())
	if err != nil {
		return err
	}

	for _, board := range boards {
		for _, query := range boardQueryPanels(board) {
			if query.QueryAnnotationID == "" {
				continue
			}

			dataset := boardQueryDataset(query)
			if _, exists := g.datasets[dataset]; exists {
				g.Resources = append(g.Resources, terraformutils.NewResource(
					query.QueryAnnotationID,
					query.QueryAnnotationID,
					"honeycombio_query_annotation",
					"honeycombio",
					map[string]string{
						"query_id": query.QueryID,
						"dataset":  dataset,
					},
					[]string{},
					map[string]interface{}{},
				))
			}
		}
	}

	return nil
}
