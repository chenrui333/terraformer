// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	accessanalyzertypes "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var accessanalyzerAllowEmptyValues = []string{"tags."}

const (
	accessAnalyzerAnalyzerResourceType    = "aws_accessanalyzer_analyzer"
	accessAnalyzerArchiveRuleResourceType = "aws_accessanalyzer_archive_rule"
	accessAnalyzerArchiveRuleIDSeparator  = "/"
	accessAnalyzerResourceNameSeparator   = ":"
)

type AccessAnalyzerGenerator struct {
	AWSService
}

func (g *AccessAnalyzerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := accessanalyzer.NewFromConfig(config)
	p := accessanalyzer.NewListAnalyzersPaginator(svc, &accessanalyzer.ListAnalyzersInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, e := p.NextPage(context.TODO())
		if e != nil {
			return e
		}
		for _, analyzer := range page.Analyzers {
			analyzerName := StringValue(analyzer.Name)
			if analyzerName == "" {
				continue
			}
			resources = append(resources, newAccessAnalyzerAnalyzerResource(analyzerName))
			archiveRuleResources, err := accessAnalyzerArchiveRuleResources(svc, analyzerName)
			if err != nil {
				if accessAnalyzerResourceNotFound(err) {
					continue
				}
				return err
			}
			resources = append(resources, archiveRuleResources...)
		}
	}
	g.Resources = resources
	return nil
}

func accessAnalyzerArchiveRuleResources(svc *accessanalyzer.Client, analyzerName string) ([]terraformutils.Resource, error) {
	p := accessanalyzer.NewListArchiveRulesPaginator(svc, &accessanalyzer.ListArchiveRulesInput{
		AnalyzerName: &analyzerName,
	})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		for _, rule := range page.ArchiveRules {
			ruleName := StringValue(rule.RuleName)
			if ruleName == "" {
				continue
			}
			resources = append(resources, newAccessAnalyzerArchiveRuleResource(analyzerName, ruleName))
		}
	}
	return resources, nil
}

func newAccessAnalyzerAnalyzerResource(analyzerName string) terraformutils.Resource {
	return terraformutils.NewResource(
		analyzerName,
		accessAnalyzerResourceName(analyzerName),
		accessAnalyzerAnalyzerResourceType,
		"aws",
		map[string]string{
			"analyzer_name": analyzerName,
		},
		accessanalyzerAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAccessAnalyzerArchiveRuleResource(analyzerName, ruleName string) terraformutils.Resource {
	return terraformutils.NewResource(
		accessAnalyzerArchiveRuleResourceID(analyzerName, ruleName),
		accessAnalyzerResourceName(analyzerName, ruleName),
		accessAnalyzerArchiveRuleResourceType,
		"aws",
		map[string]string{
			"analyzer_name": analyzerName,
			"rule_name":     ruleName,
		},
		accessanalyzerAllowEmptyValues,
		map[string]interface{}{},
	)
}

func accessAnalyzerArchiveRuleResourceID(analyzerName, ruleName string) string {
	return strings.Join([]string{analyzerName, ruleName}, accessAnalyzerArchiveRuleIDSeparator)
}

func accessAnalyzerResourceName(parts ...string) string {
	return strings.Join(parts, accessAnalyzerResourceNameSeparator)
}

func accessAnalyzerResourceNotFound(err error) bool {
	var notFound *accessanalyzertypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
