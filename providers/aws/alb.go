// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

var AlbAllowEmptyValues = []string{"tags.", "^condition."}

type AlbGenerator struct {
	AWSService
}

func (g *AlbGenerator) loadLB(svc *elasticloadbalancingv2.Client) error {
	p := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(svc, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, lb := range page.LoadBalancers {
			resourceName := StringValue(lb.LoadBalancerName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*lb.LoadBalancerArn,
				resourceName,
				"aws_lb",
				"aws",
				AlbAllowEmptyValues,
			))
			err := g.loadLBListener(svc, lb.LoadBalancerArn)
			if err != nil {
				return fmt.Errorf("load listeners for load balancer %s: %w", StringValue(lb.LoadBalancerArn), err)
			}
		}
	}
	return nil
}

func (g *AlbGenerator) loadLBListener(svc *elasticloadbalancingv2.Client, loadBalancerArn *string) error {
	p := elasticloadbalancingv2.NewDescribeListenersPaginator(svc, &elasticloadbalancingv2.DescribeListenersInput{LoadBalancerArn: loadBalancerArn})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ls := range page.Listeners {
			resourceName := *ls.ListenerArn
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_lb_listener",
				"aws",
				AlbAllowEmptyValues,
			))
			err := g.loadLBListenerRule(svc, ls.ListenerArn)
			if err != nil {
				return fmt.Errorf("load listener rules for listener %s: %w", StringValue(ls.ListenerArn), err)
			}
			err = g.loadLBListenerCertificate(svc, &ls)
			if err != nil {
				return fmt.Errorf("load listener certificates for listener %s: %w", StringValue(ls.ListenerArn), err)
			}
		}
	}
	return nil
}

func (g *AlbGenerator) loadLBListenerRule(svc *elasticloadbalancingv2.Client, listenerArn *string) error {
	var marker *string
	for {
		lsrs, err := svc.DescribeRules(context.TODO(), &elasticloadbalancingv2.DescribeRulesInput{
			ListenerArn: listenerArn,
			Marker:      marker,
			PageSize:    aws.Int32(400)},
		)
		if err != nil {
			return err
		}
		for _, lsr := range lsrs.Rules {
			if !*lsr.IsDefault {
				resourceName := *lsr.RuleArn
				g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
					resourceName,
					resourceName,
					"aws_lb_listener_rule",
					"aws",
					AlbAllowEmptyValues,
				))
			}
		}
		marker = lsrs.NextMarker
		if marker == nil {
			break
		}
	}
	return nil
}

func (g *AlbGenerator) loadLBListenerCertificate(svc *elasticloadbalancingv2.Client, loadBalancer *types.Listener) error {
	if !listenerSupportsCertificates(loadBalancer.Protocol) {
		return nil
	}

	lcs, err := svc.DescribeListenerCertificates(context.TODO(), &elasticloadbalancingv2.DescribeListenerCertificatesInput{
		ListenerArn: loadBalancer.ListenerArn,
	})
	if err != nil {
		return err
	}
	for _, lc := range lcs.Certificates {
		certificateArn := *lc.CertificateArn
		listenerCertificateID := *loadBalancer.ListenerArn + "_" + certificateArn
		if certificateArn == *loadBalancer.Certificates[0].CertificateArn { // discard default certificate
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			listenerCertificateID,
			listenerCertificateID,
			"aws_lb_listener_certificate",
			"aws",
			map[string]string{
				"listener_arn":    *loadBalancer.ListenerArn,
				"certificate_arn": certificateArn,
			},
			AlbAllowEmptyValues,
			map[string]interface{}{},
		))
	}
	return err
}

func listenerSupportsCertificates(protocol types.ProtocolEnum) bool {
	return protocol == types.ProtocolEnumHttps || protocol == types.ProtocolEnumTls
}

func (g *AlbGenerator) loadLBTargetGroup(svc *elasticloadbalancingv2.Client) error {
	p := elasticloadbalancingv2.NewDescribeTargetGroupsPaginator(svc, &elasticloadbalancingv2.DescribeTargetGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, tg := range page.TargetGroups {
			resourceName := StringValue(tg.TargetGroupName)
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*tg.TargetGroupArn,
				resourceName,
				"aws_lb_target_group",
				"aws",
				AlbAllowEmptyValues,
			))
			err := g.loadTargetGroupTargets(svc, tg.TargetGroupArn)
			if err != nil {
				return fmt.Errorf("load target group targets for %s: %w", StringValue(tg.TargetGroupArn), err)
			}
		}
	}
	return nil
}

func (g *AlbGenerator) loadTargetGroupTargets(svc *elasticloadbalancingv2.Client, targetGroupArn *string) error {
	targetHealths, err := svc.DescribeTargetHealth(context.TODO(), &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: targetGroupArn,
	})
	if err != nil {
		return err
	}
	for _, tgh := range targetHealths.TargetHealthDescriptions {
		resource, ok := newALBTargetGroupAttachmentResource(StringValue(targetGroupArn), tgh.Target)
		if ok {
			g.Resources = append(g.Resources, resource)
		}
	}
	return nil
}

func newALBTargetGroupAttachmentResource(targetGroupArn string, target *types.TargetDescription) (terraformutils.Resource, bool) {
	if target == nil {
		return terraformutils.Resource{}, false
	}
	targetID := StringValue(target.Id)
	if targetGroupArn == "" || targetID == "" {
		return terraformutils.Resource{}, false
	}
	attributes := map[string]string{
		"target_id":        targetID,
		"target_group_arn": targetGroupArn,
	}
	idParts := []string{targetGroupArn, targetID}
	if target.Port != nil {
		port := strconv.FormatInt(int64(*target.Port), 10)
		attributes["port"] = port
		idParts = append(idParts, port)
	}
	if availabilityZone := StringValue(target.AvailabilityZone); availabilityZone != "" {
		attributes["availability_zone"] = availabilityZone
		if target.Port == nil {
			idParts = append(idParts, "")
		}
		idParts = append(idParts, availabilityZone)
	}
	resourceName := strings.Join(idParts, ",")
	return terraformutils.NewResource(
		resourceName,
		resourceName,
		"aws_lb_target_group_attachment",
		"aws",
		attributes,
		AlbAllowEmptyValues,
		map[string]interface{}{},
	), true
}

// Generate TerraformResources from AWS API,
func (g *AlbGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := elasticloadbalancingv2.NewFromConfig(config)
	if err := g.loadLB(svc); err != nil {
		return err
	}
	if err := g.loadLBTargetGroup(svc); err != nil {
		return err
	}
	return nil
}

func (g *AlbGenerator) PostConvertHook() error {
	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_lb_listener" {
			continue
		}
		if r.InstanceState.Attributes["default_action.0.order"] == "0" {
			delete(r.Item["default_action"].([]interface{})[0].(map[string]interface{}), "order")
		}
	}

	for i, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_lb_listener_rule" {
			continue
		}
		if r.InstanceState.Attributes["action.0.order"] == "0" {
			delete(r.Item["action"].([]interface{})[0].(map[string]interface{}), "order")
		}
		for _, lb := range g.Resources {
			if lb.InstanceInfo.Type != "aws_lb_listener_certificate" {
				continue
			}
			if r.InstanceState.Attributes["certificate_arn"] == lb.InstanceState.Attributes["arn"] {
				g.Resources[i].Item["certificate_arn"] = "${aws_lb_listener_certificate." + lb.ResourceName + ".arn}"
			}
		}
	}

	for _, r := range g.Resources {
		if r.InstanceInfo.Type != "aws_lb" {
			continue
		}
		if val, ok := r.InstanceState.Attributes["access_logs.0.enabled"]; ok && val == "false" {
			delete(r.Item, "access_logs")
		}
	}
	return nil
}
