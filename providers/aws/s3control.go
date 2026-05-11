// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/s3control"
	s3controltypes "github.com/aws/aws-sdk-go-v2/service/s3control/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	s3ControlAccessPointResourceType                   = "aws_s3_access_point"
	s3ControlAccessPointPolicyResourceType             = "aws_s3control_access_point_policy"
	s3ControlObjectLambdaAccessPointResourceType       = "aws_s3control_object_lambda_access_point"
	s3ControlObjectLambdaAccessPointPolicyResourceType = "aws_s3control_object_lambda_access_point_policy"
	s3ControlAccessPointIDSeparator                    = ":"
)

var s3ControlAllowEmptyValues = []string{"tags."}

type S3ControlGenerator struct {
	AWSService
}

func (g *S3ControlGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	accountID, err := g.getAccountNumber(config)
	if err != nil {
		return err
	}
	if accountID == nil || *accountID == "" {
		return nil
	}

	svc := s3control.NewFromConfig(config)
	if err := g.loadAccessPoints(svc, *accountID); err != nil {
		return err
	}
	return g.loadObjectLambdaAccessPoints(svc, *accountID)
}

func (g *S3ControlGenerator) PostConvertHook() error {
	splitPolicyIDs := s3ControlSplitAccessPointPolicyIDs(g.Resources)
	for i, resource := range g.Resources {
		if resource.InstanceInfo == nil {
			continue
		}
		switch resource.InstanceInfo.Type {
		case s3ControlAccessPointResourceType:
			if resource.InstanceState != nil && splitPolicyIDs[resource.InstanceState.ID] {
				if g.Resources[i].Item != nil {
					delete(g.Resources[i].Item, "policy")
				}
				if g.Resources[i].InstanceState != nil {
					deleteFlatmapAttribute(g.Resources[i].InstanceState.Attributes, "policy")
				}
			}
			wrapS3ControlPolicyHeredoc(g, &g.Resources[i])
		case s3ControlAccessPointPolicyResourceType,
			s3ControlObjectLambdaAccessPointPolicyResourceType:
			wrapS3ControlPolicyHeredoc(g, &g.Resources[i])
		}
	}
	return nil
}

func (g *S3ControlGenerator) loadAccessPoints(svc *s3control.Client, accountID string) error {
	if accountID == "" {
		return nil
	}
	p := s3control.NewListAccessPointsPaginator(svc, &s3control.ListAccessPointsInput{
		AccountId: aws.String(accountID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if s3ControlResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, listedAccessPoint := range page.AccessPointList {
			apiName := s3ControlAccessPointAPIName(StringValue(listedAccessPoint.Name), StringValue(listedAccessPoint.AccessPointArn))
			if apiName == "" {
				continue
			}
			accessPoint, err := svc.GetAccessPoint(context.TODO(), &s3control.GetAccessPointInput{
				AccountId: aws.String(accountID),
				Name:      aws.String(apiName),
			})
			if s3ControlResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newS3ControlAccessPointResource(accountID, accessPoint); ok {
				g.Resources = append(g.Resources, resource)
			}
			g.addAccessPointPolicy(svc, accountID, apiName, accessPoint)
		}
	}
	return nil
}

func (g *S3ControlGenerator) addAccessPointPolicy(svc *s3control.Client, accountID, apiName string, accessPoint *s3control.GetAccessPointOutput) {
	if accessPoint == nil {
		return
	}
	accessPointName := StringValue(accessPoint.Name)
	accessPointARN := StringValue(accessPoint.AccessPointArn)
	policy, ok := getS3ControlAccessPointPolicy(svc, accountID, apiName)
	if !ok {
		return
	}
	if resource, ok := newS3ControlAccessPointPolicyResource(accountID, accessPointName, accessPointARN, policy); ok {
		g.Resources = append(g.Resources, resource)
	}
}

func (g *S3ControlGenerator) loadObjectLambdaAccessPoints(svc *s3control.Client, accountID string) error {
	if accountID == "" {
		return nil
	}
	p := s3control.NewListAccessPointsForObjectLambdaPaginator(svc, &s3control.ListAccessPointsForObjectLambdaInput{
		AccountId: aws.String(accountID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if s3ControlResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, accessPoint := range page.ObjectLambdaAccessPointList {
			name := StringValue(accessPoint.Name)
			if name == "" {
				continue
			}
			configuration, err := svc.GetAccessPointConfigurationForObjectLambda(context.TODO(), &s3control.GetAccessPointConfigurationForObjectLambdaInput{
				AccountId: aws.String(accountID),
				Name:      aws.String(name),
			})
			if s3ControlResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			details, err := svc.GetAccessPointForObjectLambda(context.TODO(), &s3control.GetAccessPointForObjectLambdaInput{
				AccountId: aws.String(accountID),
				Name:      aws.String(name),
			})
			if s3ControlResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if !s3ControlObjectLambdaAccessPointReadable(details) {
				continue
			}
			if resource, ok := newS3ControlObjectLambdaAccessPointResource(accountID, accessPoint, configuration); ok {
				g.Resources = append(g.Resources, resource)
			}
			g.addObjectLambdaAccessPointPolicy(svc, accountID, name)
		}
	}
	return nil
}

func (g *S3ControlGenerator) addObjectLambdaAccessPointPolicy(svc *s3control.Client, accountID, name string) {
	policy, ok := getS3ControlObjectLambdaAccessPointPolicy(svc, accountID, name)
	if !ok {
		return
	}
	if resource, ok := newS3ControlObjectLambdaAccessPointPolicyResource(accountID, name, policy); ok {
		g.Resources = append(g.Resources, resource)
	}
}

func newS3ControlAccessPointResource(accountID string, accessPoint *s3control.GetAccessPointOutput) (terraformutils.Resource, bool) {
	if accessPoint == nil {
		return terraformutils.Resource{}, false
	}
	name := StringValue(accessPoint.Name)
	bucket := StringValue(accessPoint.Bucket)
	accessPointARN := StringValue(accessPoint.AccessPointArn)
	if accountID == "" || name == "" || bucket == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"account_id": accountID,
		"bucket":     bucket,
		"name":       name,
	}
	if bucketAccountID := StringValue(accessPoint.BucketAccountId); bucketAccountID != "" {
		attributes["bucket_account_id"] = bucketAccountID
	}
	return terraformutils.NewResource(
		s3ControlAccessPointImportID(accountID, name, accessPointARN),
		s3ControlResourceName("access_point", accountID, name, accessPointARN),
		s3ControlAccessPointResourceType,
		"aws",
		attributes,
		s3ControlAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newS3ControlAccessPointPolicyResource(accountID, accessPointName, accessPointARN, policy string) (terraformutils.Resource, bool) {
	if accountID == "" || accessPointName == "" || accessPointARN == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		accessPointARN,
		s3ControlResourceName("access_point_policy", accountID, accessPointName, accessPointARN),
		s3ControlAccessPointPolicyResourceType,
		"aws",
		map[string]string{
			"access_point_arn": accessPointARN,
			"policy":           policy,
		},
		s3ControlAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newS3ControlObjectLambdaAccessPointResource(accountID string, accessPoint s3controltypes.ObjectLambdaAccessPoint, configuration *s3control.GetAccessPointConfigurationForObjectLambdaOutput) (terraformutils.Resource, bool) {
	name := StringValue(accessPoint.Name)
	if accountID == "" || name == "" || !s3ControlObjectLambdaAccessPointImportable(configuration) {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		s3ControlObjectLambdaAccessPointImportID(accountID, name),
		s3ControlResourceName("object_lambda_access_point", accountID, name, StringValue(accessPoint.ObjectLambdaAccessPointArn)),
		s3ControlObjectLambdaAccessPointResourceType,
		"aws",
		map[string]string{
			"account_id": accountID,
			"name":       name,
		},
		s3ControlAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newS3ControlObjectLambdaAccessPointPolicyResource(accountID, name, policy string) (terraformutils.Resource, bool) {
	if accountID == "" || name == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		s3ControlObjectLambdaAccessPointImportID(accountID, name),
		s3ControlResourceName("object_lambda_access_point_policy", accountID, name),
		s3ControlObjectLambdaAccessPointPolicyResourceType,
		"aws",
		map[string]string{
			"account_id": accountID,
			"name":       name,
			"policy":     policy,
		},
		s3ControlAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func getS3ControlAccessPointPolicy(svc *s3control.Client, accountID, apiName string) (string, bool) {
	if accountID == "" || apiName == "" {
		return "", false
	}
	policyOutput, err := svc.GetAccessPointPolicy(context.TODO(), &s3control.GetAccessPointPolicyInput{
		AccountId: aws.String(accountID),
		Name:      aws.String(apiName),
	})
	if s3ControlResourceNotFound(err) {
		return "", false
	}
	if err != nil {
		log.Printf("skipping S3 Control access point policy discovery for %s: %v", apiName, err)
		return "", false
	}
	if policyOutput == nil {
		return "", false
	}
	policy := StringValue(policyOutput.Policy)
	if policy == "" {
		return "", false
	}
	if _, ok := getS3ControlAccessPointPolicyStatus(svc, accountID, apiName); !ok {
		return "", false
	}
	return policy, true
}

func getS3ControlAccessPointPolicyStatus(svc *s3control.Client, accountID, apiName string) (*s3controltypes.PolicyStatus, bool) {
	statusOutput, err := svc.GetAccessPointPolicyStatus(context.TODO(), &s3control.GetAccessPointPolicyStatusInput{
		AccountId: aws.String(accountID),
		Name:      aws.String(apiName),
	})
	if s3ControlResourceNotFound(err) {
		return nil, false
	}
	if s3ControlDirectoryBucketPolicyStatusUnsupported(err) {
		return &s3controltypes.PolicyStatus{IsPublic: false}, true
	}
	if err != nil {
		log.Printf("skipping S3 Control access point policy status discovery for %s: %v", apiName, err)
		return nil, false
	}
	if statusOutput == nil || statusOutput.PolicyStatus == nil {
		return nil, false
	}
	return statusOutput.PolicyStatus, true
}

func getS3ControlObjectLambdaAccessPointPolicy(svc *s3control.Client, accountID, name string) (string, bool) {
	if accountID == "" || name == "" {
		return "", false
	}
	policyOutput, err := svc.GetAccessPointPolicyForObjectLambda(context.TODO(), &s3control.GetAccessPointPolicyForObjectLambdaInput{
		AccountId: aws.String(accountID),
		Name:      aws.String(name),
	})
	if s3ControlResourceNotFound(err) {
		return "", false
	}
	if err != nil {
		log.Printf("skipping S3 Control Object Lambda access point policy discovery for %s: %v", name, err)
		return "", false
	}
	if policyOutput == nil {
		return "", false
	}
	policy := StringValue(policyOutput.Policy)
	if policy == "" {
		return "", false
	}
	statusOutput, err := svc.GetAccessPointPolicyStatusForObjectLambda(context.TODO(), &s3control.GetAccessPointPolicyStatusForObjectLambdaInput{
		AccountId: aws.String(accountID),
		Name:      aws.String(name),
	})
	if s3ControlResourceNotFound(err) {
		return "", false
	}
	if err != nil {
		log.Printf("skipping S3 Control Object Lambda access point policy status discovery for %s: %v", name, err)
		return "", false
	}
	if statusOutput == nil || statusOutput.PolicyStatus == nil {
		return "", false
	}
	return policy, true
}

func s3ControlAccessPointImportID(accountID, accessPointName, accessPointARN string) string {
	if s3ControlARNService(accessPointARN) == "s3-outposts" {
		return accessPointARN
	}
	return strings.Join([]string{accountID, accessPointName}, s3ControlAccessPointIDSeparator)
}

func s3ControlAccessPointImportIDFromARN(accessPointARN string) (string, bool) {
	parsedARN, err := arn.Parse(accessPointARN)
	if err != nil {
		return "", false
	}
	switch parsedARN.Service {
	case "s3", "s3express":
		accessPointName := strings.TrimPrefix(parsedARN.Resource, "accesspoint/")
		if accessPointName == parsedARN.Resource || parsedARN.AccountID == "" || accessPointName == "" {
			return "", false
		}
		return s3ControlAccessPointImportID(parsedARN.AccountID, accessPointName, accessPointARN), true
	case "s3-outposts":
		return accessPointARN, true
	default:
		return "", false
	}
}

func s3ControlObjectLambdaAccessPointImportID(accountID, accessPointName string) string {
	return strings.Join([]string{accountID, accessPointName}, s3ControlAccessPointIDSeparator)
}

func s3ControlAccessPointAPIName(accessPointName, accessPointARN string) string {
	if s3ControlARNService(accessPointARN) == "s3-outposts" {
		return accessPointARN
	}
	return accessPointName
}

func s3ControlARNService(value string) string {
	parsedARN, err := arn.Parse(value)
	if err != nil {
		return ""
	}
	return parsedARN.Service
}

func s3ControlObjectLambdaAccessPointImportable(configuration *s3control.GetAccessPointConfigurationForObjectLambdaOutput) bool {
	if configuration == nil || configuration.Configuration == nil {
		return false
	}
	return StringValue(configuration.Configuration.SupportingAccessPoint) != "" &&
		len(configuration.Configuration.TransformationConfigurations) > 0
}

func s3ControlObjectLambdaAccessPointReadable(accessPoint *s3control.GetAccessPointForObjectLambdaOutput) bool {
	return accessPoint != nil && accessPoint.Alias != nil
}

func s3ControlResourceName(parts ...string) string {
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

func s3ControlResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *s3controltypes.NotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "NoSuchAccessPoint",
		"NoSuchAccessPointPolicy",
		"NoSuchBucket",
		"NotFound",
		"NotFoundException":
		return true
	default:
		return false
	}
}

func s3ControlDirectoryBucketPolicyStatusUnsupported(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "MethodNotAllowed", "UnknownError":
		return true
	default:
		return false
	}
}

func s3ControlSplitAccessPointPolicyIDs(resources []terraformutils.Resource) map[string]bool {
	ids := map[string]bool{}
	for _, resource := range resources {
		if resource.InstanceInfo == nil || resource.InstanceInfo.Type != s3ControlAccessPointPolicyResourceType {
			continue
		}
		if resource.InstanceState == nil || resource.InstanceState.ID == "" {
			continue
		}
		ids[resource.InstanceState.ID] = true
		if resource.InstanceState.Attributes != nil {
			if id, ok := s3ControlAccessPointImportIDFromARN(resource.InstanceState.Attributes["access_point_arn"]); ok {
				ids[id] = true
			}
		}
	}
	return ids
}

func wrapS3ControlPolicyHeredoc(g *S3ControlGenerator, resource *terraformutils.Resource) {
	if resource == nil || resource.Item == nil {
		return
	}
	policy, ok := resource.Item["policy"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
