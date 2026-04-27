// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/budgets"
	"github.com/aws/aws-sdk-go-v2/service/budgets/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

type BudgetsGenerator struct {
	AWSService
}

func (g *BudgetsGenerator) createResources(budgets []types.Budget, account *string) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, budget := range budgets {
		resourceName := StringValue(budget.BudgetName)
		resources = append(resources, terraformutils.NewSimpleResource(
			fmt.Sprintf("%s:%s", *account, resourceName),
			resourceName,
			"aws_budgets_budget",
			"aws",
			[]string{}))
	}
	return resources
}

func (g *BudgetsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	budgetsSvc := budgets.NewFromConfig(config)

	account, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}

	output, err := budgetsSvc.DescribeBudgets(context.TODO(), &budgets.DescribeBudgetsInput{AccountId: account})
	if err != nil {
		return err
	}

	g.Resources = g.createResources(output.Budgets, account)
	return nil
}
