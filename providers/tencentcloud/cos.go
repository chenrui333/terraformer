// SPDX-License-Identifier: Apache-2.0

package tencentcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentyun/cos-go-sdk-v5"
)

type CosGenerator struct {
	TencentCloudService
}

func (g *CosGenerator) InitResources() error {
	args := g.GetArgs()
	region := args["region"].(string)
	credential := args["credential"].(common.Credential)
	requestURL := fmt.Sprintf("https://cos.%s.myqcloud.com", region)
	u, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("parse Tencent COS service URL: %w", err)
	}
	uri := &cos.BaseURL{ServiceURL: u}
	client := cos.NewClient(uri, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     credential.SecretId,
			SecretKey:    credential.SecretKey,
			SessionToken: credential.Token,
		},
	})

	result, _, err := client.Service.Get(context.Background())
	if err != nil {
		return err
	}

	for _, bucket := range result.Buckets {
		resource := terraformutils.NewResource(
			bucket.Name,
			bucket.Name,
			"tencentcloud_cos_bucket",
			"tencentcloud",
			map[string]string{
				"acl": "private",
			},
			[]string{},
			map[string]interface{}{},
		)
		g.Resources = append(g.Resources, resource)
	}

	return nil
}

func (g *CosGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "tencentcloud_cos_bucket" {
			if _, ok := resource.Item["lifecycle_rules"]; ok {
				lifecycleRules := resource.Item["lifecycle_rules"].([]interface{})
				for i := range lifecycleRules {
					rule := lifecycleRules[i].(map[string]interface{})
					if _, ok := rule["filter_prefix"]; !ok {
						rule["filter_prefix"] = ""
						lifecycleRules[i] = rule
					}
				}
			}
		}
	}
	return nil
}
