// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mediastore"
	mediastoretypes "github.com/aws/aws-sdk-go-v2/service/mediastore/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const mediaStoreContainerPolicyResourceType = "aws_media_store_container_policy"

var mediastoreAllowEmptyValues = []string{"tags."}

type MediaStoreGenerator struct {
	AWSService
}

func (g *MediaStoreGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := mediastore.NewFromConfig(config)
	p := mediastore.NewListContainersPaginator(svc, &mediastore.ListContainersInput{})
	var resources []terraformutils.Resource
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, container := range page.Containers {
			containerName := StringValue(container.Name)
			resources = append(resources, terraformutils.NewSimpleResource(
				containerName,
				containerName,
				"aws_media_store_container",
				"aws",
				mediastoreAllowEmptyValues))
			if resource, ok := getMediaStoreContainerPolicyResource(svc, containerName); ok {
				resources = append(resources, resource)
			}
		}
	}
	g.Resources = resources
	return nil
}

func (g *MediaStoreGenerator) PostConvertHook() error {
	for i := range g.Resources {
		if g.Resources[i].InstanceInfo.Type == mediaStoreContainerPolicyResourceType {
			wrapMediaStorePolicyHeredoc(g, &g.Resources[i])
		}
	}
	return nil
}

type mediaStoreContainerPolicyGetter interface {
	GetContainerPolicy(context.Context, *mediastore.GetContainerPolicyInput, ...func(*mediastore.Options)) (*mediastore.GetContainerPolicyOutput, error)
}

func getMediaStoreContainerPolicyResource(svc mediaStoreContainerPolicyGetter, containerName string) (terraformutils.Resource, bool) {
	if containerName == "" {
		return terraformutils.Resource{}, false
	}
	output, err := svc.GetContainerPolicy(context.TODO(), &mediastore.GetContainerPolicyInput{
		ContainerName: aws.String(containerName),
	})
	if err != nil {
		if !mediaStoreContainerPolicyNotFound(err) {
			log.Printf("skipping optional MediaStore container policy discovery for %s: %v", containerName, err)
		}
		return terraformutils.Resource{}, false
	}
	if output == nil {
		return terraformutils.Resource{}, false
	}
	return newMediaStoreContainerPolicyResource(containerName, StringValue(output.Policy))
}

func newMediaStoreContainerPolicyResource(containerName, policy string) (terraformutils.Resource, bool) {
	if containerName == "" || policy == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewResource(
		mediaStoreContainerPolicyImportID(containerName),
		mediaStoreResourceName("container-policy", containerName),
		mediaStoreContainerPolicyResourceType,
		"aws",
		map[string]string{
			"container_name": containerName,
			"policy":         policy,
		},
		mediastoreAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func mediaStoreContainerPolicyImportID(containerName string) string {
	return containerName
}

func mediaStoreResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "media-store-resource"
	}
	return strings.Join(cleanParts, "/")
}

func mediaStoreContainerPolicyNotFound(err error) bool {
	var containerNotFound *mediastoretypes.ContainerNotFoundException
	var policyNotFound *mediastoretypes.PolicyNotFoundException
	return errors.As(err, &containerNotFound) || errors.As(err, &policyNotFound)
}

func wrapMediaStorePolicyHeredoc(g *MediaStoreGenerator, resource *terraformutils.Resource) {
	if resource == nil || resource.Item == nil {
		return
	}
	policy, ok := resource.Item["policy"].(string)
	if !ok || policy == "" {
		return
	}
	resource.Item["policy"] = fmt.Sprintf("<<POLICY\n%s\nPOLICY", g.escapeAwsInterpolation(policy))
}
