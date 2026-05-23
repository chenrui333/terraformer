// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var acmAllowEmptyValues = []string{}

var acmAdditionalFields = map[string]interface{}{}

type ACMGenerator struct {
	AWSService
}

func (g *ACMGenerator) createCertificatesResources(svc *acm.Client) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	p := acm.NewListCertificatesPaginator(svc, &acm.ListCertificatesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("list ACM certificates: %w", err)
		}
		for _, cert := range page.CertificateSummaryList {
			certArn := StringValue(cert.CertificateArn)
			domainName := strings.TrimSuffix(StringValue(cert.DomainName), ".")
			if certArn == "" || domainName == "" {
				continue
			}
			describeOutput, err := svc.DescribeCertificate(context.TODO(), &acm.DescribeCertificateInput{
				CertificateArn: aws.String(certArn),
			})
			if err != nil {
				return nil, fmt.Errorf("describe ACM certificate %s: %w", certArn, err)
			}
			if describeOutput == nil || describeOutput.Certificate == nil {
				return nil, fmt.Errorf("describe ACM certificate %s: empty certificate", certArn)
			}
			if !acmCertificateStatusImportable(describeOutput.Certificate.Status) {
				continue
			}
			certID := extractCertificateUUID(certArn)
			resources = append(resources, terraformutils.NewResource(
				certArn,
				certID+"_"+domainName,
				"aws_acm_certificate",
				"aws",
				map[string]string{
					"domain_name": domainName,
				},
				acmAllowEmptyValues,
				acmAdditionalFields,
			))
		}
	}
	return resources, nil
}

func acmCertificateStatusImportable(status acmtypes.CertificateStatus) bool {
	return status != acmtypes.CertificateStatusValidationTimedOut
}

// Generate TerraformResources from AWS API,
// create terraform resource for each certificates
func (g *ACMGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := acm.NewFromConfig(config)

	resources, err := g.createCertificatesResources(svc)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

// extractCertificateUUID extracts UUID from ARN
func extractCertificateUUID(arn string) string {
	if i := strings.Index(arn, "/"); i != -1 {
		return arn[i+1:]
	}
	return arn
}
