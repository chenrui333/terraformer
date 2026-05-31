// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v6"
)

type AccountFeatureGenerator struct {
	HerokuService
}

func (g AccountFeatureGenerator) createResources(accountFeatureList []heroku.AccountFeature) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, accountFeature := range accountFeatureList {
		resources = append(resources, terraformutils.NewResource(
			accountFeature.ID,
			accountFeature.Name,
			"heroku_account_feature",
			"heroku",
			map[string]string{"name": accountFeature.Name},
			[]string{},
			map[string]interface{}{}))
	}
	return resources
}

func (g *AccountFeatureGenerator) InitResources() error {
	svc := g.generateService()
	ctx := context.Background()
	list := []heroku.AccountFeature{}

	accountFeatures, err := svc.AccountFeatureList(ctx, &heroku.ListRange{Field: "id"})
	if err != nil {
		return err
	}
	for _, accountFeature := range accountFeatures {
		if accountFeature.Enabled {
			list = append(list, accountFeature)
		}
	}
	g.Resources = g.createResources(list)
	return nil
}
