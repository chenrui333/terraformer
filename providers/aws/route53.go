// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/smithy-go"
)

var route53AllowEmptyValues = []string{}

var route53AdditionalFields = map[string]interface{}{}

const (
	route53ZoneResourceType             = "aws_route53_zone"
	route53RecordResourceType           = "aws_route53_record"
	route53HealthCheckResourceType      = "aws_route53_health_check"
	route53QueryLogResourceType         = "aws_route53_query_log"
	route53DelegationSetResourceType    = "aws_route53_delegation_set"
	route53KeySigningKeyResourceType    = "aws_route53_key_signing_key"
	route53HostedZoneDNSSECResourceType = "aws_route53_hosted_zone_dnssec"
	route53IDSeparator                  = ","
)

type Route53Generator struct {
	AWSService
}

func (g *Route53Generator) createZonesResources(svc *route53.Client) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	p := route53.NewListHostedZonesPaginator(svc, &route53.ListHostedZonesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("list Route53 hosted zones: %w", err)
		}
		for _, zone := range page.HostedZones {
			zoneID := cleanZoneID(StringValue(zone.Id))
			resources = append(resources, terraformutils.NewResource(
				zoneID,
				zoneID+"_"+strings.TrimSuffix(StringValue(zone.Name), "."),
				route53ZoneResourceType,
				"aws",
				map[string]string{
					"name":          StringValue(zone.Name),
					"force_destroy": "false",
				},
				route53AllowEmptyValues,
				route53AdditionalFields,
			))
			records, err := g.createRecordsResources(svc, zoneID)
			if err != nil {
				return nil, err
			}
			resources = append(resources, records...)
			dnssecResources, err := g.createDNSSECResources(svc, zoneID)
			if err != nil {
				log.Printf("skipping Route 53 DNSSEC discovery for hosted zone %s: %v", zoneID, err)
			} else {
				resources = append(resources, dnssecResources...)
			}
		}
	}
	return resources, nil
}

func (Route53Generator) createRecordsResources(svc *route53.Client, zoneID string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	var sets *route53.ListResourceRecordSetsOutput
	var err error
	listParams := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	for {
		sets, err = svc.ListResourceRecordSets(context.TODO(), listParams)
		if err != nil {
			return nil, fmt.Errorf("list Route53 records for zone %s: %w", zoneID, err)
		}
		for _, record := range sets.ResourceRecordSets {
			recordName := wildcardUnescape(StringValue(record.Name))
			typeString := string(record.Type)
			resources = append(resources, terraformutils.NewResource(
				fmt.Sprintf("%s_%s_%s_%s", zoneID, recordName, typeString, StringValue(record.SetIdentifier)),
				fmt.Sprintf("%s_%s_%s_%s", zoneID, recordName, typeString, StringValue(record.SetIdentifier)),
				route53RecordResourceType,
				"aws",
				map[string]string{
					"name":           strings.TrimSuffix(recordName, "."),
					"zone_id":        zoneID,
					"type":           typeString,
					"set_identifier": StringValue(record.SetIdentifier),
				},
				route53AllowEmptyValues,
				route53AdditionalFields,
			))
		}

		if sets.IsTruncated {
			listParams.StartRecordName = sets.NextRecordName
			listParams.StartRecordType = sets.NextRecordType
			listParams.StartRecordIdentifier = sets.NextRecordIdentifier
		} else {
			break
		}
	}
	return resources, nil
}

func (Route53Generator) createHealthChecksResources(svc *route53.Client) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	p := route53.NewListHealthChecksPaginator(svc, &route53.ListHealthChecksInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("list Route53 health checks: %w", err)
		}
		for _, healthCheck := range page.HealthChecks {
			healthCheckStringType := string(healthCheck.HealthCheckConfig.Type)

			resources = append(resources, terraformutils.NewSimpleResource(
				StringValue(healthCheck.Id),
				fmt.Sprintf("%s_%s", StringValue(healthCheck.Id), healthCheckStringType),
				route53HealthCheckResourceType,
				"aws",
				route53AllowEmptyValues,
			))
		}
	}
	return resources, nil
}

func (Route53Generator) createQueryLogResources(svc *route53.Client) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	p := route53.NewListQueryLoggingConfigsPaginator(svc, &route53.ListQueryLoggingConfigsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if route53ResourceNotFound(err) {
			return resources, nil
		}
		if err != nil {
			return nil, err
		}
		for _, queryLogConfig := range page.QueryLoggingConfigs {
			if resource, ok := newRoute53QueryLogResource(queryLogConfig); ok {
				resources = append(resources, resource)
			}
		}
	}
	return resources, nil
}

func (Route53Generator) createDelegationSetResources(svc *route53.Client) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	input := &route53.ListReusableDelegationSetsInput{}
	for {
		page, err := svc.ListReusableDelegationSets(context.TODO(), input)
		if route53ResourceNotFound(err) {
			return resources, nil
		}
		if err != nil {
			return nil, err
		}
		for _, delegationSet := range page.DelegationSets {
			if resource, ok := newRoute53DelegationSetResource(delegationSet); ok {
				resources = append(resources, resource)
			}
		}
		if !page.IsTruncated {
			break
		}
		input.Marker = page.NextMarker
	}
	return resources, nil
}

func (Route53Generator) createDNSSECResources(svc *route53.Client, zoneID string) ([]terraformutils.Resource, error) {
	if zoneID == "" {
		return nil, nil
	}
	output, err := svc.GetDNSSEC(context.TODO(), &route53.GetDNSSECInput{
		HostedZoneId: aws.String(zoneID),
	})
	if route53ResourceNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, nil
	}
	var resources []terraformutils.Resource
	if resource, ok := newRoute53HostedZoneDNSSECResource(zoneID, output); ok {
		resources = append(resources, resource)
	}
	for _, keySigningKey := range output.KeySigningKeys {
		if resource, ok := newRoute53KeySigningKeyResource(zoneID, keySigningKey); ok {
			resources = append(resources, resource)
		}
	}
	return resources, nil
}

// Generate TerraformResources from AWS API,
// create terraform resource for each zone + each record
func (g *Route53Generator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := route53.NewFromConfig(config)

	resources, err := g.createZonesResources(svc)
	if err != nil {
		return err
	}
	g.Resources = resources
	healthCheckResources, err := g.createHealthChecksResources(svc)
	if err != nil {
		return err
	}
	g.Resources = append(g.Resources, healthCheckResources...)
	queryLogResources, err := g.createQueryLogResources(svc)
	if err != nil {
		log.Printf("skipping Route 53 query log discovery: %v", err)
	} else {
		g.Resources = append(g.Resources, queryLogResources...)
	}
	delegationSetResources, err := g.createDelegationSetResources(svc)
	if err != nil {
		log.Printf("skipping Route 53 delegation set discovery: %v", err)
	} else {
		g.Resources = append(g.Resources, delegationSetResources...)
	}

	return nil
}

func (g *Route53Generator) PostConvertHook() error {
	for i, resource := range g.Resources {
		resourceType := resource.InstanceInfo.Type
		if resourceType == route53ZoneResourceType {
			continue
		}

		if resourceType == route53HealthCheckResourceType {
			if _, childHealthChecksExist := resource.Item["child_healthchecks"]; !childHealthChecksExist {
				if _, childHealthCheckThreshholdExist := resource.Item["child_health_threshold"]; childHealthCheckThreshholdExist {
					delete(g.Resources[i].Item, "child_health_threshold")
				}
			}
			continue
		}

		item := resource.Item
		if item == nil {
			continue
		}
		switch resourceType {
		case route53RecordResourceType, route53QueryLogResourceType:
			g.replaceHostedZoneReference(i, "zone_id")
		case route53KeySigningKeyResourceType, route53HostedZoneDNSSECResourceType:
			g.replaceHostedZoneReference(i, "hosted_zone_id")
		}
		if _, aliasExist := resource.Item["alias"]; aliasExist {
			if _, ttlExist := resource.Item["ttl"]; ttlExist {
				delete(g.Resources[i].Item, "ttl")
			}
		}
	}
	return nil
}

func (g *Route53Generator) replaceHostedZoneReference(resourceIndex int, attributeName string) {
	if resourceIndex < 0 || resourceIndex >= len(g.Resources) || g.Resources[resourceIndex].Item == nil {
		return
	}
	zoneID, ok := g.Resources[resourceIndex].Item[attributeName].(string)
	if !ok || zoneID == "" {
		return
	}
	for _, resourceZone := range g.Resources {
		if resourceZone.InstanceInfo == nil || resourceZone.InstanceInfo.Type != route53ZoneResourceType {
			continue
		}
		if zoneID == resourceZone.InstanceState.ID {
			g.Resources[resourceIndex].Item[attributeName] = "${aws_route53_zone." + resourceZone.ResourceName + ".zone_id}"
			return
		}
	}
}

func newRoute53QueryLogResource(queryLogConfig route53types.QueryLoggingConfig) (terraformutils.Resource, bool) {
	id := StringValue(queryLogConfig.Id)
	zoneID := cleanZoneID(StringValue(queryLogConfig.HostedZoneId))
	logGroupARN := StringValue(queryLogConfig.CloudWatchLogsLogGroupArn)
	if id == "" || zoneID == "" || logGroupARN == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		route53ResourceName("query_log", zoneID, id),
		route53QueryLogResourceType,
		"aws",
		map[string]string{
			"cloudwatch_log_group_arn": logGroupARN,
			"zone_id":                  zoneID,
		},
		route53AllowEmptyValues,
		route53AdditionalFields,
	), true
}

func newRoute53DelegationSetResource(delegationSet route53types.DelegationSet) (terraformutils.Resource, bool) {
	id := cleanDelegationSetID(StringValue(delegationSet.Id))
	if id == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		id,
		route53ResourceName("delegation_set", id),
		route53DelegationSetResourceType,
		"aws",
		map[string]string{},
		route53AllowEmptyValues,
		route53AdditionalFields,
	), true
}

func newRoute53HostedZoneDNSSECResource(zoneID string, output *route53.GetDNSSECOutput) (terraformutils.Resource, bool) {
	if !route53HostedZoneDNSSECImportable(output) || zoneID == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		zoneID,
		route53ResourceName("hosted_zone_dnssec", zoneID),
		route53HostedZoneDNSSECResourceType,
		"aws",
		map[string]string{
			"hosted_zone_id": zoneID,
			"signing_status": StringValue(output.Status.ServeSignature),
		},
		route53AllowEmptyValues,
		route53AdditionalFields,
	), true
}

func newRoute53KeySigningKeyResource(zoneID string, keySigningKey route53types.KeySigningKey) (terraformutils.Resource, bool) {
	if !route53KeySigningKeyImportable(keySigningKey) || zoneID == "" {
		return terraformutils.Resource{}, false
	}
	name := StringValue(keySigningKey.Name)
	return terraformutils.NewResource(
		route53KeySigningKeyImportID(zoneID, name),
		route53ResourceName("key_signing_key", zoneID, name),
		route53KeySigningKeyResourceType,
		"aws",
		map[string]string{
			"hosted_zone_id":             zoneID,
			"key_management_service_arn": StringValue(keySigningKey.KmsArn),
			"name":                       name,
			"status":                     StringValue(keySigningKey.Status),
		},
		route53AllowEmptyValues,
		route53AdditionalFields,
	), true
}

func route53HostedZoneDNSSECImportable(output *route53.GetDNSSECOutput) bool {
	if output == nil || output.Status == nil {
		return false
	}
	signingStatus := StringValue(output.Status.ServeSignature)
	return signingStatus == "SIGNING" ||
		(signingStatus == "NOT_SIGNING" && len(output.KeySigningKeys) > 0)
}

func route53KeySigningKeyImportable(keySigningKey route53types.KeySigningKey) bool {
	status := StringValue(keySigningKey.Status)
	return StringValue(keySigningKey.Name) != "" &&
		StringValue(keySigningKey.KmsArn) != "" &&
		(status == "ACTIVE" || status == "INACTIVE")
}

func route53KeySigningKeyImportID(zoneID, name string) string {
	return strings.Join([]string{zoneID, name}, route53IDSeparator)
}

func route53ResourceName(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}

func wildcardUnescape(s string) string {
	return strings.Replace(s, "\\052", "*", 1)
}

func cleanDelegationSetID(id string) string {
	return cleanPrefix(id, "/delegationset/")
}

// cleanZoneID is used to remove the leading /hostedzone/
func cleanZoneID(id string) string {
	return cleanPrefix(id, "/hostedzone/")
}

// cleanPrefix removes a string prefix from an ID
func cleanPrefix(id, prefix string) string {
	return strings.TrimPrefix(id, prefix)
}

func route53ResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var dnssecNotFound *route53types.DNSSECNotFound
	if errors.As(err, &dnssecNotFound) {
		return true
	}
	var noSuchDelegationSet *route53types.NoSuchDelegationSet
	if errors.As(err, &noSuchDelegationSet) {
		return true
	}
	var noSuchHostedZone *route53types.NoSuchHostedZone
	if errors.As(err, &noSuchHostedZone) {
		return true
	}
	var noSuchKeySigningKey *route53types.NoSuchKeySigningKey
	if errors.As(err, &noSuchKeySigningKey) {
		return true
	}
	var noSuchQueryLoggingConfig *route53types.NoSuchQueryLoggingConfig
	if errors.As(err, &noSuchQueryLoggingConfig) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "DNSSECNotFound",
		"HostedZoneNotFound",
		"NoSuchDelegationSet",
		"NoSuchHostedZone",
		"NoSuchKeySigningKey",
		"NoSuchQueryLoggingConfig":
		return true
	default:
		return false
	}
}
