// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAutoScalingPostConvertHookLinksLaunchConfiguration(t *testing.T) {
	autoscalingGroup := terraformutils.NewResource(
		"app-asg",
		"app-asg",
		"aws_autoscaling_group",
		"aws",
		map[string]string{"launch_configuration": "app-lc"},
		AsgAllowEmptyValues,
		map[string]interface{}{},
	)
	launchConfiguration := terraformutils.NewResource(
		"app-lc",
		"app-lc",
		"aws_launch_configuration",
		"aws",
		map[string]string{"name": "app-lc"},
		AsgAllowEmptyValues,
		map[string]interface{}{},
	)
	generator := AutoScalingGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{autoscalingGroup, launchConfiguration},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}

	want := "${aws_launch_configuration." + launchConfiguration.ResourceName + ".name}"
	if got := generator.Resources[0].Item["launch_configuration"]; got != want {
		t.Fatalf("launch_configuration = %q, want %q", got, want)
	}
}

func TestAutoScalingPostConvertHookSkipsMalformedResources(t *testing.T) {
	autoscalingGroup := terraformutils.NewResource(
		"app-asg",
		"app-asg",
		"aws_autoscaling_group",
		"aws",
		map[string]string{"launch_configuration": "missing-lc"},
		AsgAllowEmptyValues,
		map[string]interface{}{},
	)
	launchConfiguration := terraformutils.NewResource(
		"app-lc",
		"app-lc",
		"aws_launch_configuration",
		"aws",
		map[string]string{},
		AsgAllowEmptyValues,
		map[string]interface{}{},
	)
	generator := AutoScalingGenerator{
		AWSService: AWSService{
			Service: terraformutils.Service{
				Resources: []terraformutils.Resource{
					{},
					autoscalingGroup,
					launchConfiguration,
				},
			},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	if generator.Resources[1].Item != nil {
		t.Fatalf("unmatched autoscaling group item = %#v, want nil", generator.Resources[1].Item)
	}
}
