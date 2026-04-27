package honeycombio

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	hnyclient "github.com/honeycombio/terraform-provider-honeycombio/client"
)

type BoardGenerator struct {
	HoneycombService
}

func (g *BoardGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to initialize Honeycomb client: %w", err)
	}

	boards, err := client.Boards.List(context.TODO())
	if err != nil {
		return fmt.Errorf("unable to list Honeycomb boards: %w", err)
	}

	for _, board := range boards {
		// all of a board's queries must be in our list of target datasets or we don't import it
		onlyValidDatasets := true
		for _, query := range boardQueryPanels(board) {
			dataset := boardQueryDataset(query)
			if _, exists := g.datasets[dataset]; !exists {
				onlyValidDatasets = false
				break
			}
		}

		if onlyValidDatasets {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				board.ID,
				board.ID,
				"honeycombio_board",
				"honeycombio",
				[]string{},
			))
		}
	}

	return nil
}

func boardQueryPanels(board hnyclient.Board) []hnyclient.BoardQueryPanel {
	queryPanels := make([]hnyclient.BoardQueryPanel, 0, len(board.Panels))
	for _, panel := range board.Panels {
		if panel.QueryPanel == nil {
			continue
		}
		if panel.PanelType != "" && panel.PanelType != hnyclient.BoardPanelTypeQuery {
			continue
		}
		queryPanels = append(queryPanels, *panel.QueryPanel)
	}

	return queryPanels
}

func boardQueryDataset(query hnyclient.BoardQueryPanel) string {
	if query.Dataset == "" {
		// assume an unset dataset is an environment-wide query
		return environmentWideDatasetSlug
	}

	return query.Dataset
}
