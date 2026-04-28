// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var sqsAllowEmptyValues = []string{"tags."}

type SqsGenerator struct {
	AWSService
}

func (g *SqsGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := sqs.NewFromConfig(config)

	listQueuesInput := sqs.ListQueuesInput{}

	sqsPrefix, hasPrefix := os.LookupEnv("SQS_PREFIX")
	if hasPrefix {
		listQueuesInput.QueueNamePrefix = aws.String(sqsPrefix)
	}

	queuesList, err := svc.ListQueues(context.TODO(), &listQueuesInput)

	if err != nil {
		return err
	}

	for _, queueURL := range queuesList.QueueUrls {
		queueName := arnLastSegment(queueURL, "/")

		g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
			queueURL,
			queueName,
			"aws_sqs_queue",
			"aws",
			sqsAllowEmptyValues,
		))
	}

	return nil
}

// PostConvertHook for add policy json as heredoc
func (g *SqsGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_sqs_queue" {
			if val, ok := g.Resources[i].Item["policy"]; ok {
				policy := g.escapeAwsInterpolation(val.(string))
				g.Resources[i].Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
			}
		}
	}
	return nil
}
