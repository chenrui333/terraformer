package opal

import (
	"context"
	"fmt"

	"github.com/chenrui333/terraformer/terraformutils"
	opalsdk "github.com/opalsecurity/opal-go"
)

type OwnerGenerator struct {
	OpalService
}

func (g *OwnerGenerator) createResources(owners []opalsdk.Owner, countByName map[string]int) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	for _, owner := range owners {
		resourceID, err := opalRequiredString("opal_owner", "owner_id", owner.OwnerId)
		if err != nil {
			return nil, err
		}
		name := opalUniqueResourceName(opalResourceDisplayName(owner.Name, resourceID), countByName)

		resources = append(resources, terraformutils.NewSimpleResource(
			resourceID,
			name,
			"opal_owner",
			"opal",
			[]string{},
		))
	}

	return resources, nil
}

func (g *OwnerGenerator) InitResources() error {
	client, err := g.newClient()
	if err != nil {
		return fmt.Errorf("unable to list opal owners: %w", err)
	}

	owners, _, err := client.OwnersAPI.GetOwners(context.TODO()).Execute()
	if err != nil {
		return fmt.Errorf("unable to list opal owners: %w", err)
	}

	countByName := make(map[string]int)

	for {
		resources, err := g.createResources(owners.Results, countByName)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)

		if !owners.HasNext() || owners.Next == nil {
			break
		}

		owners, _, err = client.OwnersAPI.GetOwners(context.TODO()).Cursor(*owners.Next).Execute()
		if err != nil {
			return fmt.Errorf("unable to list opal owners: %w", err)
		}
	}

	return nil
}
