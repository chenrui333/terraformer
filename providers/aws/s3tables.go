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
	"github.com/aws/aws-sdk-go-v2/service/s3tables"
	s3tablestypes "github.com/aws/aws-sdk-go-v2/service/s3tables/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

const (
	s3TablesTableBucketResourceType       = "aws_s3tables_table_bucket"
	s3TablesNamespaceResourceType         = "aws_s3tables_namespace"
	s3TablesTableResourceType             = "aws_s3tables_table"
	s3TablesTableBucketPolicyResourceType = "aws_s3tables_table_bucket_policy"
	s3TablesIDSeparator                   = ";"
)

var s3TablesAllowEmptyValues = []string{"tags."}

type S3TablesGenerator struct {
	AWSService
}

func (g *S3TablesGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}

	svc := s3tables.NewFromConfig(config)
	return g.loadTableBuckets(svc)
}

func (g *S3TablesGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo == nil {
			continue
		}
		if g.Resources[i].InstanceInfo.Type == s3TablesTableBucketPolicyResourceType {
			wrapS3TablesPolicyHeredoc(g, &g.Resources[i])
		}
	}
	return nil
}

func (g *S3TablesGenerator) loadTableBuckets(svc *s3tables.Client) error {
	paginator := s3tables.NewListTableBucketsPaginator(svc, &s3tables.ListTableBucketsInput{
		Type: s3tablestypes.TableBucketTypeCustomer,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if s3TablesResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, bucketSummary := range page.TableBuckets {
			tableBucketARN := StringValue(bucketSummary.Arn)
			if tableBucketARN == "" {
				continue
			}
			tableBucket, err := svc.GetTableBucket(context.TODO(), &s3tables.GetTableBucketInput{
				TableBucketARN: aws.String(tableBucketARN),
			})
			if s3TablesResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newS3TablesTableBucketResource(tableBucket); ok {
				g.Resources = append(g.Resources, resource)
			}
			g.addTableBucketPolicy(svc, tableBucketARN)
			if err := g.loadNamespaces(svc, tableBucketARN); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *S3TablesGenerator) addTableBucketPolicy(svc *s3tables.Client, tableBucketARN string) {
	policy, ok := getS3TablesTableBucketPolicy(svc, tableBucketARN)
	if !ok {
		return
	}
	if resource, ok := newS3TablesTableBucketPolicyResource(tableBucketARN, policy); ok {
		g.Resources = append(g.Resources, resource)
	}
}

func (g *S3TablesGenerator) loadNamespaces(svc *s3tables.Client, tableBucketARN string) error {
	if tableBucketARN == "" {
		return nil
	}
	paginator := s3tables.NewListNamespacesPaginator(svc, &s3tables.ListNamespacesInput{
		TableBucketARN: aws.String(tableBucketARN),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if s3TablesResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, namespaceSummary := range page.Namespaces {
			namespace, ok := s3TablesNamespaceName(namespaceSummary.Namespace)
			if !ok {
				continue
			}
			namespaceOutput, err := svc.GetNamespace(context.TODO(), &s3tables.GetNamespaceInput{
				Namespace:      aws.String(namespace),
				TableBucketARN: aws.String(tableBucketARN),
			})
			if s3TablesResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newS3TablesNamespaceResource(tableBucketARN, namespaceOutput); ok {
				g.Resources = append(g.Resources, resource)
			}
			if err := g.loadTables(svc, tableBucketARN, namespace); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *S3TablesGenerator) loadTables(svc *s3tables.Client, tableBucketARN, namespace string) error {
	if tableBucketARN == "" || namespace == "" {
		return nil
	}
	paginator := s3tables.NewListTablesPaginator(svc, &s3tables.ListTablesInput{
		Namespace:      aws.String(namespace),
		TableBucketARN: aws.String(tableBucketARN),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if s3TablesResourceNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, tableSummary := range page.Tables {
			tableName := StringValue(tableSummary.Name)
			if tableName == "" {
				continue
			}
			table, err := svc.GetTable(context.TODO(), &s3tables.GetTableInput{
				Name:           aws.String(tableName),
				Namespace:      aws.String(namespace),
				TableBucketARN: aws.String(tableBucketARN),
			})
			if s3TablesResourceNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if resource, ok := newS3TablesTableResource(tableBucketARN, table); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newS3TablesTableBucketResource(tableBucket *s3tables.GetTableBucketOutput) (terraformutils.Resource, bool) {
	if !s3TablesTableBucketImportable(tableBucket) {
		return terraformutils.Resource{}, false
	}
	tableBucketARN := StringValue(tableBucket.Arn)
	name := StringValue(tableBucket.Name)
	if tableBucketARN == "" || name == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		tableBucketARN,
		s3TablesResourceName("table_bucket", name, tableBucketARN),
		s3TablesTableBucketResourceType,
		"aws",
		map[string]string{
			"arn":           tableBucketARN,
			"force_destroy": "false",
			"name":          name,
		},
		s3TablesAllowEmptyValues,
		map[string]interface{}{},
	)
	setS3TablesPreserveIDAfterRefresh(&resource)
	return resource, true
}

func newS3TablesNamespaceResource(tableBucketARN string, namespaceOutput *s3tables.GetNamespaceOutput) (terraformutils.Resource, bool) {
	if namespaceOutput == nil {
		return terraformutils.Resource{}, false
	}
	namespace, ok := s3TablesNamespaceName(namespaceOutput.Namespace)
	if !ok || tableBucketARN == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		s3TablesNamespaceImportID(tableBucketARN, namespace),
		s3TablesResourceName("namespace", tableBucketARN, namespace),
		s3TablesNamespaceResourceType,
		"aws",
		map[string]string{
			"namespace":        namespace,
			"table_bucket_arn": tableBucketARN,
		},
		s3TablesAllowEmptyValues,
		map[string]interface{}{},
	)
	setS3TablesPreserveIDAfterRefresh(&resource)
	return resource, true
}

func newS3TablesTableResource(tableBucketARN string, table *s3tables.GetTableOutput) (terraformutils.Resource, bool) {
	if !s3TablesTableImportable(table) {
		return terraformutils.Resource{}, false
	}
	name := StringValue(table.Name)
	namespace, ok := s3TablesNamespaceName(table.Namespace)
	tableARN := StringValue(table.TableARN)
	format := string(table.Format)
	if !ok || tableBucketARN == "" || name == "" || tableARN == "" || format == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		s3TablesTableImportID(tableBucketARN, namespace, name),
		s3TablesResourceName("table", tableBucketARN, namespace, name, tableARN),
		s3TablesTableResourceType,
		"aws",
		map[string]string{
			"arn":              tableARN,
			"format":           format,
			"name":             name,
			"namespace":        namespace,
			"table_bucket_arn": tableBucketARN,
		},
		s3TablesAllowEmptyValues,
		map[string]interface{}{},
	)
	setS3TablesPreserveIDAfterRefresh(&resource)
	return resource, true
}

func newS3TablesTableBucketPolicyResource(tableBucketARN, policy string) (terraformutils.Resource, bool) {
	if tableBucketARN == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	resource := terraformutils.NewResource(
		tableBucketARN,
		s3TablesResourceName("table_bucket_policy", tableBucketARN),
		s3TablesTableBucketPolicyResourceType,
		"aws",
		map[string]string{
			"resource_policy":  policy,
			"table_bucket_arn": tableBucketARN,
		},
		s3TablesAllowEmptyValues,
		map[string]interface{}{},
	)
	setS3TablesPreserveIDAfterRefresh(&resource)
	return resource, true
}

func getS3TablesTableBucketPolicy(svc *s3tables.Client, tableBucketARN string) (string, bool) {
	if tableBucketARN == "" {
		return "", false
	}
	policyOutput, err := svc.GetTableBucketPolicy(context.TODO(), &s3tables.GetTableBucketPolicyInput{
		TableBucketARN: aws.String(tableBucketARN),
	})
	if s3TablesResourceNotFound(err) {
		return "", false
	}
	if err != nil {
		log.Printf("skipping S3 Tables table bucket policy discovery for %s: %v", tableBucketARN, err)
		return "", false
	}
	if policyOutput == nil {
		return "", false
	}
	policy := StringValue(policyOutput.ResourcePolicy)
	if policy == "" {
		return "", false
	}
	return policy, true
}

func s3TablesNamespaceImportID(tableBucketARN, namespace string) string {
	return strings.Join([]string{tableBucketARN, namespace}, s3TablesIDSeparator)
}

func s3TablesTableImportID(tableBucketARN, namespace, name string) string {
	return strings.Join([]string{tableBucketARN, namespace, name}, s3TablesIDSeparator)
}

func s3TablesNamespaceName(parts []string) (string, bool) {
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}

func s3TablesTableBucketImportable(tableBucket *s3tables.GetTableBucketOutput) bool {
	if tableBucket == nil {
		return false
	}
	return tableBucket.Type == "" || tableBucket.Type == s3tablestypes.TableBucketTypeCustomer
}

func s3TablesTableImportable(table *s3tables.GetTableOutput) bool {
	if table == nil {
		return false
	}
	if StringValue(table.ManagedByService) != "" {
		return false
	}
	return table.Type == "" || table.Type == s3tablestypes.TableTypeCustomer
}

func s3TablesResourceName(parts ...string) string {
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

func s3TablesResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound *s3tablestypes.NotFoundException
	if errors.As(err, &notFound) {
		return true
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.ErrorCode() {
	case "NotFound",
		"NotFoundException",
		"NoSuchResource",
		"ResourceNotFoundException":
		return true
	default:
		return false
	}
}

func setS3TablesPreserveIDAfterRefresh(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceState == nil {
		return
	}
	if resource.InstanceState.Meta == nil {
		resource.InstanceState.Meta = map[string]interface{}{}
	}
	resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh] = true
}

func wrapS3TablesPolicyHeredoc(g *S3TablesGenerator, resource *terraformutils.Resource) {
	if resource == nil || resource.Item == nil {
		return
	}
	policy, ok := resource.Item["resource_policy"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["resource_policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
