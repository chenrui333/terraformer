// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var cloudtrailAllowEmptyValues = []string{"tags."}

type CloudTrailGenerator struct {
	AWSService
}

func (g *CloudTrailGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := cloudtrail.NewFromConfig(config)

	if err := g.addTrails(svc); err != nil {
		return err
	}
	if err := g.addEventDataStores(svc); err != nil {
		log.Printf("Skipping CloudTrail event data stores: %v", err)
	}
	return nil
}

func (g *CloudTrailGenerator) addTrails(svc *cloudtrail.Client) error {
	output, err := svc.DescribeTrails(context.TODO(), &cloudtrail.DescribeTrailsInput{})
	if err != nil {
		return err
	}
	for _, trail := range output.TrailList {
		resourceName := StringValue(trail.Name)
		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			resourceName,
			resourceName,
			"aws_cloudtrail",
			"aws",
			cloudtrailAllowEmptyValues))
	}
	return nil
}

func eventDataStoreToResource(arn, name string) terraformutils.Resource {
	if name == "" {
		name = arnLastSegment(arn, "/")
	}
	return terraformutils.NewSimpleResource(
		arn,
		name,
		"aws_cloudtrail_event_data_store",
		"aws",
		cloudtrailAllowEmptyValues,
	)
}

func (g *CloudTrailGenerator) addEventDataStores(svc *cloudtrail.Client) error {
	p := cloudtrail.NewListEventDataStoresPaginator(svc, &cloudtrail.ListEventDataStoresInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, eds := range page.EventDataStores {
			if eds.EventDataStoreArn == nil {
				continue
			}
			edsARN := *eds.EventDataStoreArn
			detail, err := svc.GetEventDataStore(context.TODO(), &cloudtrail.GetEventDataStoreInput{
				EventDataStore: &edsARN,
			})
			if err != nil {
				log.Printf("Skipping event data store %s: %v", edsARN, err)
				continue
			}
			if detail.Status == types.EventDataStoreStatusPendingDeletion {
				continue
			}
			g.Resources = append(g.Resources, eventDataStoreToResource(edsARN, StringValue(detail.Name)))
		}
	}
	return nil
}
