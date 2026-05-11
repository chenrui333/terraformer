// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	guarddutytypes "github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

const (
	guardDutyDetectorResourceType                  = "aws_guardduty_detector"
	guardDutyFilterResourceType                    = "aws_guardduty_filter"
	guardDutyIPSetResourceType                     = "aws_guardduty_ipset"
	guardDutyMalwareProtectionPlanResourceType     = "aws_guardduty_malware_protection_plan"
	guardDutyMemberResourceType                    = "aws_guardduty_member"
	guardDutyOrganizationAdminAccountResourceType  = "aws_guardduty_organization_admin_account"
	guardDutyOrganizationConfigurationResourceType = "aws_guardduty_organization_configuration"
	guardDutyPublishingDestinationResourceType     = "aws_guardduty_publishing_destination"
	guardDutyThreatIntelSetResourceType            = "aws_guardduty_threatintelset"

	guardDutyResourceIDSeparator   = ":"
	guardDutyResourceNameSeparator = ":"
	guardDutyUpdatedAtField        = "updatedAt"

	guardDutyFilterUpdatedAtCriteriaAdditionalField = "__guardduty_filter_updated_at_criteria"

	guardDutyMemberRelationshipStatusCreated                     = "Created"
	guardDutyMemberRelationshipStatusDisabled                    = "Disabled"
	guardDutyMemberRelationshipStatusEnabled                     = "Enabled"
	guardDutyMemberRelationshipStatusEmailVerificationInProgress = "EmailVerificationInProgress"
	guardDutyMemberRelationshipStatusInvited                     = "Invited"
)

var guardDutyAllowEmptyValues = []string{"tags."}

// GuardDutyGenerator generates GuardDuty resources.
type GuardDutyGenerator struct {
	AWSService
}

// InitResources generates Terraform resources from the GuardDuty API.
func (g *GuardDutyGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := guardduty.NewFromConfig(config)
	if err := g.loadOrganizationAdminAccounts(svc); err != nil {
		if !guardDutyOrganizationResourceUnavailable(err) {
			return err
		}
	}
	if err := g.loadMalwareProtectionPlans(svc); err != nil {
		if !guardDutyMalwareProtectionPlansUnavailable(err) {
			return err
		}
	}
	detectors, e := listGuardDutyDetectors(svc)
	if e != nil {
		return e
	}

	for _, detectorID := range detectors {
		if detectorID == "" {
			continue
		}
		resourceStart := len(g.Resources)
		detector, err := svc.GetDetector(context.TODO(), &guardduty.GetDetectorInput{
			DetectorId: aws.String(detectorID),
		})
		if guardDutyResourceNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		resource, ok := newGuardDutyDetectorResource(detectorID, detector)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)

		if err := g.loadDetectorChildren(svc, detectorID); err != nil {
			if guardDutyResourceNotFound(err) {
				g.Resources = g.Resources[:resourceStart]
				continue
			}
			return err
		}
	}

	return nil
}

func (g *GuardDutyGenerator) PostConvertHook() error {
	for i := range g.Resources {
		resource := &g.Resources[i]
		if resource.InstanceInfo == nil || resource.InstanceInfo.Type != guardDutyFilterResourceType {
			continue
		}
		updatedAtCriteria := guardDutyUpdatedAtCriteriaFromAdditionalFields(resource.AdditionalFields)
		guardDutyRestoreUpdatedAtCriteria(resource, updatedAtCriteria)
	}
	return nil
}

func (g *GuardDutyGenerator) loadDetectorChildren(svc *guardduty.Client, detectorID string) error {
	if err := g.loadFilters(svc, detectorID); err != nil {
		return err
	}
	if err := g.loadIPSets(svc, detectorID); err != nil {
		return err
	}
	if err := g.loadThreatIntelSets(svc, detectorID); err != nil {
		return err
	}
	if err := g.loadMembers(svc, detectorID); err != nil {
		return err
	}
	if err := g.loadOrganizationConfiguration(svc, detectorID); err != nil {
		if guardDutyResourceNotFound(err) {
			return err
		}
		if !guardDutyOrganizationResourceUnavailable(err) {
			return err
		}
	}
	return g.loadPublishingDestinations(svc, detectorID)
}

func (g *GuardDutyGenerator) loadOrganizationAdminAccounts(svc *guardduty.Client) error {
	paginator := guardduty.NewListOrganizationAdminAccountsPaginator(svc, &guardduty.ListOrganizationAdminAccountsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if guardDutyOrganizationResourceUnavailable(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, adminAccount := range page.AdminAccounts {
			resource, ok := newGuardDutyOrganizationAdminAccountResource(adminAccount)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadFilters(svc *guardduty.Client, detectorID string) error {
	paginator := guardduty.NewListFiltersPaginator(svc, &guardduty.ListFiltersInput{
		DetectorId: aws.String(detectorID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, filterName := range page.FilterNames {
			if filterName == "" {
				continue
			}
			filter, err := svc.GetFilter(context.TODO(), &guardduty.GetFilterInput{
				DetectorId: aws.String(detectorID),
				FilterName: aws.String(filterName),
			})
			if guardDutyResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			resource, ok := newGuardDutyFilterResource(detectorID, filterName, filter)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadIPSets(svc *guardduty.Client, detectorID string) error {
	paginator := guardduty.NewListIPSetsPaginator(svc, &guardduty.ListIPSetsInput{
		DetectorId: aws.String(detectorID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if guardDutyMemberListingUnavailable(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, ipSetID := range page.IpSetIds {
			if ipSetID == "" {
				continue
			}
			ipSet, err := svc.GetIPSet(context.TODO(), &guardduty.GetIPSetInput{
				DetectorId: aws.String(detectorID),
				IpSetId:    aws.String(ipSetID),
			})
			if guardDutyResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			resource, ok := newGuardDutyIPSetResource(detectorID, ipSetID, ipSet)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadThreatIntelSets(svc *guardduty.Client, detectorID string) error {
	paginator := guardduty.NewListThreatIntelSetsPaginator(svc, &guardduty.ListThreatIntelSetsInput{
		DetectorId: aws.String(detectorID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, threatIntelSetID := range page.ThreatIntelSetIds {
			if threatIntelSetID == "" {
				continue
			}
			threatIntelSet, err := svc.GetThreatIntelSet(context.TODO(), &guardduty.GetThreatIntelSetInput{
				DetectorId:       aws.String(detectorID),
				ThreatIntelSetId: aws.String(threatIntelSetID),
			})
			if guardDutyResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			resource, ok := newGuardDutyThreatIntelSetResource(detectorID, threatIntelSetID, threatIntelSet)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadMembers(svc *guardduty.Client, detectorID string) error {
	paginator := guardduty.NewListMembersPaginator(svc, &guardduty.ListMembersInput{
		DetectorId:     aws.String(detectorID),
		OnlyAssociated: aws.String("false"),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, memberSummary := range page.Members {
			accountID := StringValue(memberSummary.AccountId)
			if accountID == "" {
				continue
			}
			member, err := getGuardDutyMember(svc, detectorID, accountID)
			if guardDutyResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			resource, ok := newGuardDutyMemberResource(detectorID, accountID, member)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadOrganizationConfiguration(svc *guardduty.Client, detectorID string) error {
	configuration, err := svc.DescribeOrganizationConfiguration(context.TODO(), &guardduty.DescribeOrganizationConfigurationInput{
		DetectorId: aws.String(detectorID),
	})
	if err != nil {
		return err
	}
	resource, ok := newGuardDutyOrganizationConfigurationResource(detectorID, configuration)
	if !ok {
		return nil
	}
	g.Resources = append(g.Resources, resource)
	return nil
}

func (g *GuardDutyGenerator) loadPublishingDestinations(svc *guardduty.Client, detectorID string) error {
	paginator := guardduty.NewListPublishingDestinationsPaginator(svc, &guardduty.ListPublishingDestinationsInput{
		DetectorId: aws.String(detectorID),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, destinationSummary := range page.Destinations {
			destinationID := StringValue(destinationSummary.DestinationId)
			if destinationID == "" {
				continue
			}
			destination, err := svc.DescribePublishingDestination(context.TODO(), &guardduty.DescribePublishingDestinationInput{
				DestinationId: aws.String(destinationID),
				DetectorId:    aws.String(detectorID),
			})
			if guardDutyResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			resource, ok := newGuardDutyPublishingDestinationResource(detectorID, destinationID, destination)
			if !ok {
				continue
			}
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func (g *GuardDutyGenerator) loadMalwareProtectionPlans(svc *guardduty.Client) error {
	planIDs, err := listGuardDutyMalwareProtectionPlans(svc)
	if err != nil {
		return err
	}
	for _, planID := range planIDs {
		if planID == "" {
			continue
		}
		plan, err := svc.GetMalwareProtectionPlan(context.TODO(), &guardduty.GetMalwareProtectionPlanInput{
			MalwareProtectionPlanId: aws.String(planID),
		})
		if guardDutyResourceNotFound(err) {
			continue
		}
		if guardDutyMalwareProtectionPlansUnavailable(err) {
			continue
		}
		if err != nil {
			return err
		}
		resource, ok := newGuardDutyMalwareProtectionPlanResource(planID, plan)
		if !ok {
			continue
		}
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func listGuardDutyDetectors(svc *guardduty.Client) ([]string, error) {
	var detectors []string
	paginator := guardduty.NewListDetectorsPaginator(svc, &guardduty.ListDetectorsInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		detectors = append(detectors, page.DetectorIds...)
	}
	return detectors, nil
}

func listGuardDutyMalwareProtectionPlans(svc *guardduty.Client) ([]string, error) {
	var planIDs []string
	input := &guardduty.ListMalwareProtectionPlansInput{}
	for {
		page, err := svc.ListMalwareProtectionPlans(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, plan := range page.MalwareProtectionPlans {
			planIDs = append(planIDs, StringValue(plan.MalwareProtectionPlanId))
		}
		if page.NextToken == nil {
			break
		}
		input.NextToken = page.NextToken
	}
	return planIDs, nil
}

func getGuardDutyMember(svc *guardduty.Client, detectorID, accountID string) (*guarddutytypes.Member, error) {
	output, err := svc.GetMembers(context.TODO(), &guardduty.GetMembersInput{
		AccountIds: []string{accountID},
		DetectorId: aws.String(detectorID),
	})
	if err != nil {
		return nil, err
	}
	if output == nil || len(output.Members) == 0 {
		return nil, nil
	}
	return &output.Members[0], nil
}

func newGuardDutyDetectorResource(detectorID string, detector *guardduty.GetDetectorOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || detector == nil {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"enable": strconv.FormatBool(detector.Status == guarddutytypes.DetectorStatusEnabled),
	}
	if detector.FindingPublishingFrequency != "" {
		attributes["finding_publishing_frequency"] = string(detector.FindingPublishingFrequency)
	}
	addGuardDutyDataSources(attributes, detector.DataSources) //nolint:staticcheck // DataSources backs Terraform's deprecated datasources block.

	return terraformutils.NewResource(
		detectorID,
		guardDutyResourceName("detector", detectorID),
		guardDutyDetectorResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyFilterResource(detectorID, filterName string, filter *guardduty.GetFilterOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || filter == nil {
		return terraformutils.Resource{}, false
	}
	name := StringValue(filter.Name)
	if name == "" {
		name = filterName
	}
	if name == "" || filter.Action == "" || filter.Rank == nil {
		return terraformutils.Resource{}, false
	}

	attributes, ok := guardDutyFindingCriteriaAttributes(filter.FindingCriteria)
	if !ok {
		return terraformutils.Resource{}, false
	}
	attributes["action"] = string(filter.Action)
	attributes["detector_id"] = detectorID
	attributes["name"] = name
	attributes["rank"] = strconv.Itoa(int(*filter.Rank))
	if description := StringValue(filter.Description); description != "" {
		attributes["description"] = description
	}
	additionalFields := map[string]interface{}{}
	if updatedAtCriteria := guardDutyUpdatedAtCriteria(filter.FindingCriteria); len(updatedAtCriteria) > 0 {
		additionalFields[guardDutyFilterUpdatedAtCriteriaAdditionalField] = updatedAtCriteria
	}

	return terraformutils.NewResource(
		guardDutyChildResourceID(detectorID, name),
		guardDutyResourceName("filter", detectorID, name),
		guardDutyFilterResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		additionalFields,
	), true
}

func newGuardDutyIPSetResource(detectorID, ipSetID string, ipSet *guardduty.GetIPSetOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || ipSetID == "" || ipSet == nil || ipSet.Status == guarddutytypes.IpSetStatusDeleted {
		return terraformutils.Resource{}, false
	}
	name := StringValue(ipSet.Name)
	if name == "" || ipSet.Format == "" || StringValue(ipSet.Location) == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"activate":    strconv.FormatBool(ipSet.Status == guarddutytypes.IpSetStatusActive),
		"detector_id": detectorID,
		"format":      string(ipSet.Format),
		"location":    StringValue(ipSet.Location),
		"name":        name,
	}

	return terraformutils.NewResource(
		guardDutyChildResourceID(detectorID, ipSetID),
		guardDutyResourceName("ipset", detectorID, name, ipSetID),
		guardDutyIPSetResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyMalwareProtectionPlanResource(planID string, plan *guardduty.GetMalwareProtectionPlanOutput) (terraformutils.Resource, bool) {
	if planID == "" || plan == nil || !guardDutyMalwareProtectionPlanStatusImportable(plan.Status) {
		return terraformutils.Resource{}, false
	}
	role := StringValue(plan.Role)
	bucket := guardDutyMalwareProtectionPlanBucket(plan)
	if bucket == nil {
		return terraformutils.Resource{}, false
	}
	bucketName := StringValue(bucket.BucketName)
	if role == "" || bucketName == "" {
		return terraformutils.Resource{}, false
	}

	attributes := map[string]string{
		"protected_resource.#":                         "1",
		"protected_resource.0.s3_bucket.#":             "1",
		"protected_resource.0.s3_bucket.0.bucket_name": bucketName,
		"role": role,
	}
	if arn := StringValue(plan.Arn); arn != "" {
		attributes["arn"] = arn
	}
	if plan.CreatedAt != nil {
		attributes["created_at"] = plan.CreatedAt.UTC().Format(time.RFC3339)
	}
	if plan.Status != "" {
		attributes["status"] = string(plan.Status)
	}
	guardDutyListAttributes(attributes, "protected_resource.0.s3_bucket.0.object_prefixes", bucket.ObjectPrefixes)
	addGuardDutyMalwareProtectionPlanActions(attributes, plan.Actions)

	return terraformutils.NewResource(
		guardDutyMalwareProtectionPlanResourceID(planID),
		guardDutyResourceName("malware_protection_plan", bucketName, planID),
		guardDutyMalwareProtectionPlanResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyMemberResource(detectorID, accountID string, member *guarddutytypes.Member) (terraformutils.Resource, bool) {
	if detectorID == "" || accountID == "" || member == nil || !guardDutyMemberImportable(member) {
		return terraformutils.Resource{}, false
	}
	if StringValue(member.AccountId) != accountID {
		return terraformutils.Resource{}, false
	}
	status := StringValue(member.RelationshipStatus)
	attributes := map[string]string{
		"account_id":  accountID,
		"detector_id": detectorID,
		"email":       StringValue(member.Email),
		"invite":      strconv.FormatBool(guardDutyMemberInviteEnabled(status)),
	}

	return terraformutils.NewResource(
		guardDutyMemberResourceID(detectorID, accountID),
		guardDutyResourceName("member", detectorID, accountID),
		guardDutyMemberResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyOrganizationAdminAccountResource(adminAccount guarddutytypes.AdminAccount) (terraformutils.Resource, bool) {
	accountID := StringValue(adminAccount.AdminAccountId)
	if accountID == "" || !guardDutyOrganizationAdminAccountImportable(adminAccount) {
		return terraformutils.Resource{}, false
	}

	return terraformutils.NewResource(
		guardDutyOrganizationAdminAccountResourceID(accountID),
		guardDutyResourceName("organization_admin_account", accountID),
		guardDutyOrganizationAdminAccountResourceType,
		"aws",
		map[string]string{"admin_account_id": accountID},
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyOrganizationConfigurationResource(detectorID string, configuration *guardduty.DescribeOrganizationConfigurationOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || configuration == nil || configuration.AutoEnableOrganizationMembers == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"auto_enable_organization_members": string(configuration.AutoEnableOrganizationMembers),
		"detector_id":                      detectorID,
	}
	addGuardDutyOrganizationDataSources(attributes, configuration.DataSources) //nolint:staticcheck // DataSources backs Terraform's deprecated datasources block.

	return terraformutils.NewResource(
		guardDutyOrganizationConfigurationResourceID(detectorID),
		guardDutyResourceName("organization_configuration", detectorID),
		guardDutyOrganizationConfigurationResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyPublishingDestinationResource(detectorID, destinationID string, destination *guardduty.DescribePublishingDestinationOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || destinationID == "" || destination == nil {
		return terraformutils.Resource{}, false
	}
	properties := destination.DestinationProperties
	if properties == nil || StringValue(properties.DestinationArn) == "" || StringValue(properties.KmsKeyArn) == "" || destination.DestinationType == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"destination_arn":  StringValue(properties.DestinationArn),
		"destination_type": string(destination.DestinationType),
		"detector_id":      detectorID,
		"kms_key_arn":      StringValue(properties.KmsKeyArn),
	}
	if id := StringValue(destination.DestinationId); id != "" {
		destinationID = id
		attributes["destination_id"] = id
	}

	return terraformutils.NewResource(
		guardDutyPublishingDestinationResourceID(detectorID, destinationID),
		guardDutyResourceName("publishing_destination", detectorID, destinationID),
		guardDutyPublishingDestinationResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newGuardDutyThreatIntelSetResource(detectorID, threatIntelSetID string, threatIntelSet *guardduty.GetThreatIntelSetOutput) (terraformutils.Resource, bool) {
	if detectorID == "" || threatIntelSetID == "" || threatIntelSet == nil || threatIntelSet.Status == guarddutytypes.ThreatIntelSetStatusDeleted {
		return terraformutils.Resource{}, false
	}
	name := StringValue(threatIntelSet.Name)
	if name == "" || threatIntelSet.Format == "" || StringValue(threatIntelSet.Location) == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"activate":    strconv.FormatBool(threatIntelSet.Status == guarddutytypes.ThreatIntelSetStatusActive),
		"detector_id": detectorID,
		"format":      string(threatIntelSet.Format),
		"location":    StringValue(threatIntelSet.Location),
		"name":        name,
	}

	return terraformutils.NewResource(
		guardDutyChildResourceID(detectorID, threatIntelSetID),
		guardDutyResourceName("threatintelset", detectorID, name, threatIntelSetID),
		guardDutyThreatIntelSetResourceType,
		"aws",
		attributes,
		guardDutyAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func addGuardDutyMalwareProtectionPlanActions(attributes map[string]string, actions *guarddutytypes.MalwareProtectionPlanActions) {
	if actions == nil || actions.Tagging == nil || actions.Tagging.Status == "" {
		return
	}
	attributes["actions.#"] = "1"
	attributes["actions.0.tagging.#"] = "1"
	attributes["actions.0.tagging.0.status"] = string(actions.Tagging.Status)
}

func addGuardDutyDataSources(attributes map[string]string, dataSources *guarddutytypes.DataSourceConfigurationsResult) {
	if dataSources == nil {
		return
	}
	attributes["datasources.#"] = "1"
	if dataSources.S3Logs != nil {
		attributes["datasources.0.s3_logs.#"] = "1"
		attributes["datasources.0.s3_logs.0.enable"] = strconv.FormatBool(dataSources.S3Logs.Status == guarddutytypes.DataSourceStatusEnabled)
	}
	if dataSources.Kubernetes != nil && dataSources.Kubernetes.AuditLogs != nil {
		attributes["datasources.0.kubernetes.#"] = "1"
		attributes["datasources.0.kubernetes.0.audit_logs.#"] = "1"
		attributes["datasources.0.kubernetes.0.audit_logs.0.enable"] = strconv.FormatBool(dataSources.Kubernetes.AuditLogs.Status == guarddutytypes.DataSourceStatusEnabled)
	}
	if dataSources.MalwareProtection != nil && dataSources.MalwareProtection.ScanEc2InstanceWithFindings != nil && dataSources.MalwareProtection.ScanEc2InstanceWithFindings.EbsVolumes != nil {
		attributes["datasources.0.malware_protection.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.0.enable"] = strconv.FormatBool(dataSources.MalwareProtection.ScanEc2InstanceWithFindings.EbsVolumes.Status == guarddutytypes.DataSourceStatusEnabled)
	}
}

func addGuardDutyOrganizationDataSources(attributes map[string]string, dataSources *guarddutytypes.OrganizationDataSourceConfigurationsResult) {
	if dataSources == nil {
		return
	}
	attributes["datasources.#"] = "1"
	if dataSources.S3Logs != nil && dataSources.S3Logs.AutoEnable != nil {
		attributes["datasources.0.s3_logs.#"] = "1"
		attributes["datasources.0.s3_logs.0.auto_enable"] = strconv.FormatBool(aws.ToBool(dataSources.S3Logs.AutoEnable))
	}
	if dataSources.Kubernetes != nil && dataSources.Kubernetes.AuditLogs != nil && dataSources.Kubernetes.AuditLogs.AutoEnable != nil {
		attributes["datasources.0.kubernetes.#"] = "1"
		attributes["datasources.0.kubernetes.0.audit_logs.#"] = "1"
		attributes["datasources.0.kubernetes.0.audit_logs.0.enable"] = strconv.FormatBool(aws.ToBool(dataSources.Kubernetes.AuditLogs.AutoEnable))
	}
	if dataSources.MalwareProtection != nil && dataSources.MalwareProtection.ScanEc2InstanceWithFindings != nil && dataSources.MalwareProtection.ScanEc2InstanceWithFindings.EbsVolumes != nil && dataSources.MalwareProtection.ScanEc2InstanceWithFindings.EbsVolumes.AutoEnable != nil {
		attributes["datasources.0.malware_protection.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.#"] = "1"
		attributes["datasources.0.malware_protection.0.scan_ec2_instance_with_findings.0.ebs_volumes.0.auto_enable"] = strconv.FormatBool(aws.ToBool(dataSources.MalwareProtection.ScanEc2InstanceWithFindings.EbsVolumes.AutoEnable))
	}
}

func guardDutyFindingCriteriaAttributes(criteria *guarddutytypes.FindingCriteria) (map[string]string, bool) {
	if criteria == nil || len(criteria.Criterion) == 0 {
		return nil, false
	}
	fields := make([]string, 0, len(criteria.Criterion))
	for field, condition := range criteria.Criterion {
		if field == "" || !guardDutyConditionConfigured(condition) {
			continue
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return nil, false
	}
	sort.Strings(fields)

	attributes := map[string]string{
		"finding_criteria.#":             "1",
		"finding_criteria.0.criterion.#": strconv.Itoa(len(fields)),
	}
	for i, field := range fields {
		condition := criteria.Criterion[field]
		prefix := fmt.Sprintf("finding_criteria.0.criterion.%d", i)
		attributes[prefix+".field"] = field
		guardDutyListAttributes(attributes, prefix+".equals", condition.Equals)
		guardDutyListAttributes(attributes, prefix+".not_equals", condition.NotEquals)
		guardDutyListAttributes(attributes, prefix+".matches", condition.Matches)
		guardDutyListAttributes(attributes, prefix+".not_matches", condition.NotMatches)
		guardDutyConditionIntAttribute(attributes, prefix+".greater_than", field, condition.GreaterThan)
		guardDutyConditionIntAttribute(attributes, prefix+".greater_than_or_equal", field, condition.GreaterThanOrEqual)
		guardDutyConditionIntAttribute(attributes, prefix+".less_than", field, condition.LessThan)
		guardDutyConditionIntAttribute(attributes, prefix+".less_than_or_equal", field, condition.LessThanOrEqual)
	}
	return attributes, true
}

func guardDutyConditionConfigured(condition guarddutytypes.Condition) bool {
	return len(condition.Equals) > 0 ||
		len(condition.NotEquals) > 0 ||
		len(condition.Matches) > 0 ||
		len(condition.NotMatches) > 0 ||
		condition.GreaterThan != nil ||
		condition.GreaterThanOrEqual != nil ||
		condition.LessThan != nil ||
		condition.LessThanOrEqual != nil
}

func guardDutyListAttributes(attributes map[string]string, prefix string, values []string) {
	if len(values) == 0 {
		return
	}
	attributes[prefix+".#"] = strconv.Itoa(len(values))
	for i, value := range values {
		attributes[fmt.Sprintf("%s.%d", prefix, i)] = value
	}
}

func guardDutyConditionIntAttribute(attributes map[string]string, key, field string, value *int64) {
	if value == nil || *value <= 0 {
		return
	}
	attributes[key] = guardDutyConditionIntValue(field, *value)
}

func guardDutyConditionIntValue(field string, value int64) string {
	if field == guardDutyUpdatedAtField {
		return time.UnixMilli(value).UTC().Format(time.RFC3339Nano)
	}
	return strconv.FormatInt(value, 10)
}

func guardDutyUpdatedAtCriteria(criteria *guarddutytypes.FindingCriteria) map[string]string {
	if criteria == nil {
		return nil
	}
	condition, ok := criteria.Criterion[guardDutyUpdatedAtField]
	if !ok {
		return nil
	}
	updatedAtCriteria := map[string]string{}
	guardDutyUpdatedAtCriteriaValue(updatedAtCriteria, "greater_than", condition.GreaterThan)
	guardDutyUpdatedAtCriteriaValue(updatedAtCriteria, "greater_than_or_equal", condition.GreaterThanOrEqual)
	guardDutyUpdatedAtCriteriaValue(updatedAtCriteria, "less_than", condition.LessThan)
	guardDutyUpdatedAtCriteriaValue(updatedAtCriteria, "less_than_or_equal", condition.LessThanOrEqual)
	return updatedAtCriteria
}

func guardDutyUpdatedAtCriteriaValue(criteria map[string]string, key string, value *int64) {
	if value == nil || *value <= 0 {
		return
	}
	criteria[key] = guardDutyConditionIntValue(guardDutyUpdatedAtField, *value)
}

func guardDutyUpdatedAtCriteriaFromAdditionalFields(additionalFields map[string]interface{}) map[string]string {
	raw, ok := additionalFields[guardDutyFilterUpdatedAtCriteriaAdditionalField]
	if !ok {
		return nil
	}
	switch criteria := raw.(type) {
	case map[string]string:
		return criteria
	case map[string]interface{}:
		updatedAtCriteria := map[string]string{}
		for key, value := range criteria {
			if text, ok := value.(string); ok {
				updatedAtCriteria[key] = text
			}
		}
		return updatedAtCriteria
	default:
		return nil
	}
}

func guardDutyRestoreUpdatedAtCriteria(resource *terraformutils.Resource, updatedAtCriteria map[string]string) {
	if resource == nil {
		return
	}
	guardDutyRestoreUpdatedAtCriteriaItem(resource.Item, updatedAtCriteria)
	if resource.InstanceState == nil {
		return
	}
	guardDutyRestoreUpdatedAtCriteriaAttributes(resource.InstanceState.Attributes, updatedAtCriteria)
	guardDutyRestoreUpdatedAtCriteriaTypedAttributes(resource.InstanceState, updatedAtCriteria)
}

func guardDutyRestoreUpdatedAtCriteriaItem(item map[string]interface{}, updatedAtCriteria map[string]string) {
	if item == nil {
		return
	}
	delete(item, guardDutyFilterUpdatedAtCriteriaAdditionalField)
	if len(updatedAtCriteria) == 0 {
		return
	}

	for _, findingCriteria := range guardDutyInterfaceSlice(item["finding_criteria"]) {
		findingCriteriaMap, ok := findingCriteria.(map[string]interface{})
		if !ok {
			continue
		}
		for _, criterion := range guardDutyInterfaceSlice(findingCriteriaMap["criterion"]) {
			criterionMap, ok := criterion.(map[string]interface{})
			if !ok || criterionMap["field"] != guardDutyUpdatedAtField {
				continue
			}
			for key, value := range updatedAtCriteria {
				if _, ok := criterionMap[key]; ok {
					criterionMap[key] = value
				}
			}
		}
	}
}

func guardDutyRestoreUpdatedAtCriteriaAttributes(attributes map[string]string, updatedAtCriteria map[string]string) {
	if len(attributes) == 0 || len(updatedAtCriteria) == 0 {
		return
	}
	for key, value := range attributes {
		if value != guardDutyUpdatedAtField || !strings.HasPrefix(key, "finding_criteria.") || !strings.HasSuffix(key, ".field") {
			continue
		}
		criterionPrefix := strings.TrimSuffix(key, ".field")
		for conditionKey, conditionValue := range updatedAtCriteria {
			attributeKey := criterionPrefix + "." + conditionKey
			if _, ok := attributes[attributeKey]; ok {
				attributes[attributeKey] = conditionValue
			}
		}
	}
}

func guardDutyRestoreUpdatedAtCriteriaTypedAttributes(state *tfcompat.InstanceState, updatedAtCriteria map[string]string) {
	if state == nil || len(state.TypedAttributes) == 0 || len(updatedAtCriteria) == 0 {
		return
	}
	var attributes map[string]interface{}
	if err := json.Unmarshal(state.TypedAttributes, &attributes); err != nil {
		return
	}
	guardDutyRestoreUpdatedAtCriteriaItem(attributes, updatedAtCriteria)
	raw, err := json.Marshal(attributes)
	if err != nil {
		return
	}
	state.SetTypedAttributes(raw)
}

func guardDutyInterfaceSlice(value interface{}) []interface{} {
	switch values := value.(type) {
	case []interface{}:
		return values
	case []map[string]interface{}:
		result := make([]interface{}, 0, len(values))
		for _, value := range values {
			result = append(result, value)
		}
		return result
	default:
		return nil
	}
}

func guardDutyChildResourceID(detectorID, resourceID string) string {
	return strings.Join([]string{detectorID, resourceID}, guardDutyResourceIDSeparator)
}

func guardDutyMalwareProtectionPlanResourceID(planID string) string {
	return planID
}

func guardDutyMemberResourceID(detectorID, accountID string) string {
	return guardDutyChildResourceID(detectorID, accountID)
}

func guardDutyOrganizationAdminAccountResourceID(accountID string) string {
	return accountID
}

func guardDutyOrganizationConfigurationResourceID(detectorID string) string {
	return detectorID
}

func guardDutyPublishingDestinationResourceID(detectorID, destinationID string) string {
	return guardDutyChildResourceID(detectorID, destinationID)
}

func guardDutyResourceName(parts ...string) string {
	return strings.Join(parts, guardDutyResourceNameSeparator)
}

func guardDutyMalwareProtectionPlanBucket(plan *guardduty.GetMalwareProtectionPlanOutput) *guarddutytypes.CreateS3BucketResource {
	if plan == nil || plan.ProtectedResource == nil {
		return nil
	}
	return plan.ProtectedResource.S3Bucket
}

func guardDutyMalwareProtectionPlanStatusImportable(status guarddutytypes.MalwareProtectionPlanStatus) bool {
	return status == guarddutytypes.MalwareProtectionPlanStatusActive ||
		status == guarddutytypes.MalwareProtectionPlanStatusWarning ||
		status == guarddutytypes.MalwareProtectionPlanStatusError
}

func guardDutyMemberImportable(member *guarddutytypes.Member) bool {
	if member == nil || StringValue(member.AccountId) == "" || StringValue(member.Email) == "" {
		return false
	}
	return guardDutyMemberRelationshipImportable(StringValue(member.RelationshipStatus))
}

func guardDutyMemberRelationshipImportable(status string) bool {
	switch status {
	case guardDutyMemberRelationshipStatusCreated,
		guardDutyMemberRelationshipStatusDisabled,
		guardDutyMemberRelationshipStatusEnabled,
		guardDutyMemberRelationshipStatusEmailVerificationInProgress,
		guardDutyMemberRelationshipStatusInvited:
		return true
	default:
		return false
	}
}

func guardDutyMemberInviteEnabled(status string) bool {
	switch status {
	case guardDutyMemberRelationshipStatusDisabled,
		guardDutyMemberRelationshipStatusEnabled,
		guardDutyMemberRelationshipStatusEmailVerificationInProgress,
		guardDutyMemberRelationshipStatusInvited:
		return true
	default:
		return false
	}
}

func guardDutyOrganizationAdminAccountImportable(adminAccount guarddutytypes.AdminAccount) bool {
	return StringValue(adminAccount.AdminAccountId) != "" && adminAccount.AdminStatus == guarddutytypes.AdminStatusEnabled
}

func guardDutyMalwareProtectionPlansUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var accessDenied *guarddutytypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return true
	}
	var badRequest *guarddutytypes.BadRequestException
	if !errors.As(err, &badRequest) {
		return false
	}
	message := strings.ToLower(badRequest.ErrorMessage())
	return strings.Contains(message, "malware") ||
		strings.Contains(message, "protection plan") ||
		strings.Contains(message, "permission") ||
		strings.Contains(message, "not authorized") ||
		strings.Contains(message, "not supported") ||
		strings.Contains(message, "not enabled") ||
		strings.Contains(message, "not available")
}

func guardDutyMemberListingUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var accessDenied *guarddutytypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return true
	}
	var badRequest *guarddutytypes.BadRequestException
	if !errors.As(err, &badRequest) {
		return false
	}
	message := strings.ToLower(badRequest.ErrorMessage())
	if strings.Contains(message, "member account") && strings.Contains(message, "cannot") {
		return true
	}
	if !strings.Contains(message, "administrator") && !strings.Contains(message, "master") {
		return false
	}
	return strings.Contains(message, "not") ||
		strings.Contains(message, "only") ||
		strings.Contains(message, "must")
}

func guardDutyResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *guarddutytypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var badRequest *guarddutytypes.BadRequestException
	if !errors.As(err, &badRequest) {
		return false
	}
	message := strings.ToLower(badRequest.ErrorMessage())
	return strings.Contains(message, "no such resource found") ||
		strings.Contains(message, "input detectorid is not owned by the current account") ||
		strings.Contains(message, "input detectorid is not owned by current account")
}

func guardDutyOrganizationResourceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	var accessDenied *guarddutytypes.AccessDeniedException
	if errors.As(err, &accessDenied) {
		return true
	}
	var badRequest *guarddutytypes.BadRequestException
	if !errors.As(err, &badRequest) {
		return false
	}
	message := strings.ToLower(badRequest.ErrorMessage())
	return strings.Contains(message, "organization") &&
		(strings.Contains(message, "administrator") ||
			strings.Contains(message, "delegated") ||
			strings.Contains(message, "management") ||
			strings.Contains(message, "master") ||
			strings.Contains(message, "member"))
}
