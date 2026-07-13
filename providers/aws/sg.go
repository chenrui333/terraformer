// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
	simplegraph "gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

var SgAllowEmptyValues = []string{"tags."}

type void struct{}

var member void

type SecurityGenerator struct {
	AWSService
}

func (SecurityGenerator) createResources(securityGroups []types.SecurityGroup) []terraformutils.Resource {
	var sgIDsToMoveOut []string
	_, shouldSplitRules := os.LookupEnv("SPLIT_SG_RULES")
	if shouldSplitRules {
		ids := make(map[string]struct{})
		for _, sg := range securityGroups {
			if groupID := StringValue(sg.GroupId); groupID != "" {
				ids[groupID] = struct{}{}
			}
		}
		for groupID := range ids {
			sgIDsToMoveOut = append(sgIDsToMoveOut, groupID)
		}
		sort.Strings(sgIDsToMoveOut)
	} else {
		sgIDsToMoveOut = findSgsToMoveOut(securityGroups)
	}
	moveRulesOut := make(map[string]struct{}, len(sgIDsToMoveOut))
	for _, groupID := range sgIDsToMoveOut {
		moveRulesOut[groupID] = struct{}{}
	}

	var resources []terraformutils.Resource
	for _, sg := range securityGroups {
		groupID := StringValue(sg.GroupId)
		if groupID == "" || sg.VpcId == nil {
			continue
		}
		ruleAttributes := map[string]interface{}{}
		// we must move out all of the rules - https://github.com/hashicorp/terraform/issues/11011#issuecomment-283076580
		if _, ok := moveRulesOut[groupID]; ok {
			ruleAttributes["clearRules"] = true
			for _, rule := range sg.IpPermissions {
				resources = processRule(rule, "ingress", sg, resources)
			}
			for _, rule := range sg.IpPermissionsEgress {
				resources = processRule(rule, "egress", sg, resources)
			}
		}

		resources = append(resources, terraformutils.NewResource(
			groupID,
			strings.Trim(StringValue(sg.GroupName)+"_"+groupID, " "),
			"aws_security_group",
			"aws",
			map[string]string{},
			SgAllowEmptyValues,
			ruleAttributes))
	}
	return resources
}

func processRule(rule types.IpPermission, ruleType string, sg types.SecurityGroup, resources []terraformutils.Resource) []terraformutils.Resource {
	securityGroupID := StringValue(sg.GroupId)
	if securityGroupID == "" {
		return resources
	}
	if len(rule.UserIdGroupPairs) > 0 {
		if len(rule.IpRanges) > 0 { // we must unwind coupled CIDR IPv4 range + security group rules
			attributes := baseRuleAttributes(ruleType, rule, sg)
			resources = append(resources, terraformutils.NewResource(
				permissionID(*sg.GroupId, ruleType, "", rule),
				permissionID(*sg.GroupId, ruleType, "", rule),
				"aws_security_group_rule",
				"aws",
				terraformutils.Flatten(attributes),
				SgAllowEmptyValues,
				map[string]interface{}{}))
		}
		if len(rule.Ipv6Ranges) > 0 { // we must unwind coupled CIDR IPv6 range + security group rules
			attributes := baseRuleAttributes(ruleType, rule, sg)
			resources = append(resources, terraformutils.NewResource(
				permissionID(*sg.GroupId, ruleType, "", rule),
				permissionID(*sg.GroupId, ruleType, "", rule),
				"aws_security_group_rule",
				"aws",
				terraformutils.Flatten(attributes),
				SgAllowEmptyValues,
				map[string]interface{}{}))
		}
		for _, groupPair := range rule.UserIdGroupPairs {
			referencedGroupID := StringValue(groupPair.GroupId)
			if referencedGroupID == "" {
				continue
			}
			attributes := baseRuleAttributes(ruleType, rule, sg)
			delete(attributes, "cidr_blocks")
			delete(attributes, "ipv6_cidr_blocks")
			if referencedGroupID == securityGroupID { // Solution to C1
				attributes["self"] = true
			} else {
				attributes["source_security_group_id"] = sourceSecurityGroupID(groupPair)
			}

			resources = append(resources, terraformutils.NewResource(
				permissionID(securityGroupID, ruleType, referencedGroupID, rule),
				permissionID(securityGroupID, ruleType, referencedGroupID, rule),
				"aws_security_group_rule",
				"aws",
				terraformutils.Flatten(attributes),
				SgAllowEmptyValues,
				map[string]interface{}{}))
		}
	} else {
		attributes := baseRuleAttributes(ruleType, rule, sg)
		resources = append(resources, terraformutils.NewResource(
			permissionID(*sg.GroupId, ruleType, "", rule),
			permissionID(*sg.GroupId, ruleType, "", rule),
			"aws_security_group_rule",
			"aws",
			terraformutils.Flatten(attributes),
			SgAllowEmptyValues,
			map[string]interface{}{}))
	}
	return resources
}

func baseRuleAttributes(ruleType string, rule types.IpPermission, sg types.SecurityGroup) map[string]interface{} {
	attributes := map[string]interface{}{
		"type":              ruleType,
		"cidr_blocks":       ipRange(rule),
		"ipv6_cidr_blocks":  ip6Range(rule),
		"prefix_list_ids":   prefixes(rule),
		"from_port":         fromPort(rule),
		"protocol":          *rule.IpProtocol,
		"security_group_id": *sg.GroupId,
		"to_port":           toPort(rule),
	}
	return attributes
}

func sourceSecurityGroupID(groupPair types.UserIdGroupPair) string {
	groupID := StringValue(groupPair.GroupId)
	if ownerID := StringValue(groupPair.UserId); ownerID != "" {
		return ownerID + "/" + groupID
	}
	return groupID
}

// We cannot build a line graph and move out only rules because of hashicorp/terraform#11011.
func findSgsToMoveOut(securityGroups []types.SecurityGroup) []string {
	// Vertexes are security groups, edges are rules. The task is to find correct set of rule definitions, so that we
	// won't have cycles
	groupIDs := make([]string, 0, len(securityGroups))
	groupIDSet := make(map[string]struct{}, len(securityGroups))
	for _, sg := range securityGroups {
		groupID := StringValue(sg.GroupId)
		if groupID == "" {
			continue
		}
		if _, exists := groupIDSet[groupID]; exists {
			continue
		}
		groupIDSet[groupID] = struct{}{}
		groupIDs = append(groupIDs, groupID)
	}
	sort.Strings(groupIDs)

	sourceGraph := simplegraph.NewDirectedGraph()
	groupIDToNode := make(map[string]int64, len(groupIDs))
	nodeToGroupID := make(map[int64]string, len(groupIDs))
	for _, groupID := range groupIDs {
		node := sourceGraph.NewNode()
		sourceGraph.AddNode(node)
		groupIDToNode[groupID] = node.ID()
		nodeToGroupID[node.ID()] = groupID
	}

	ruleCounts := make(map[string]int, len(groupIDs))
	for _, sg := range securityGroups {
		groupID := StringValue(sg.GroupId)
		fromID, ok := groupIDToNode[groupID]
		if !ok {
			continue
		}

		// Duplicate API entries are collapsed by group ID. Keep the largest
		// observed rule count so exact duplicates do not skew cycle selection.
		ruleCount := len(sg.IpPermissions) + len(sg.IpPermissionsEgress)
		if ruleCount > ruleCounts[groupID] {
			ruleCounts[groupID] = ruleCount
		}

		for _, toID := range internalSecurityGroupReferences(sg, groupIDToNode) {
			if fromID == toID {
				continue
			}
			fromNode := sourceGraph.Node(fromID)
			toNode := sourceGraph.Node(toID)
			sourceGraph.SetEdge(sourceGraph.NewEdge(fromNode, toNode))
		}
	}

	resultingSet := make(map[string]void)
	for {
		var groupsToMoveOut []string
		for _, component := range topo.TarjanSCC(sourceGraph) {
			if len(component) < 2 { // Self references are not graph edges.
				continue
			}

			// Move out the node with the fewest total ingress and egress rules.
			// Use the group ID as a stable tie-breaker.
			groupID := ""
			for _, node := range component {
				candidateID := nodeToGroupID[node.ID()]
				if groupID == "" ||
					ruleCounts[candidateID] < ruleCounts[groupID] ||
					(ruleCounts[candidateID] == ruleCounts[groupID] && candidateID < groupID) {
					groupID = candidateID
				}
			}
			groupsToMoveOut = append(groupsToMoveOut, groupID)
		}
		if len(groupsToMoveOut) == 0 {
			break
		}

		// Components are disjoint, so their selected nodes can be removed together.
		// A selected group has no inline outgoing rules and cannot join a later cycle.
		sort.Strings(groupsToMoveOut)
		for _, groupID := range groupsToMoveOut {
			resultingSet[groupID] = member
			sourceGraph.RemoveNode(groupIDToNode[groupID])
		}
	}

	result := make([]string, 0, len(resultingSet))
	for groupID := range resultingSet {
		result = append(result, groupID)
	}
	sort.Strings(result)

	return result
}

func internalSecurityGroupReferences(sg types.SecurityGroup, importedGroupIndexes map[string]int64) []int64 {
	referenceSet := make(map[int64]struct{})
	permissions := [][]types.IpPermission{sg.IpPermissions, sg.IpPermissionsEgress}
	for _, permissionSet := range permissions {
		for _, permission := range permissionSet {
			for _, pair := range permission.UserIdGroupPairs {
				groupID := StringValue(pair.GroupId)
				if groupID == "" {
					continue
				}
				index, ok := importedGroupIndexes[groupID]
				if !ok {
					continue
				}
				referenceSet[index] = struct{}{}
			}
		}
	}

	references := make([]int64, 0, len(referenceSet))
	for index := range referenceSet {
		references = append(references, index)
	}
	sort.Slice(references, func(i, j int) bool { return references[i] < references[j] })
	return references
}

func (g *SecurityGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := ec2.NewFromConfig(config)
	p := ec2.NewDescribeSecurityGroupsPaginator(svc, &ec2.DescribeSecurityGroupsInput{})
	var resourcesToFilter []types.SecurityGroup
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		resourcesToFilter = append(resourcesToFilter, page.SecurityGroups...)
	}
	sort.Slice(resourcesToFilter, func(i, j int) bool {
		return StringValue(resourcesToFilter[i].GroupId) < StringValue(resourcesToFilter[j].GroupId)
	})
	g.Resources = g.createResources(resourcesToFilter)

	return nil
}

func (g *SecurityGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_security_group_rule" {
			if resource.Item["self"] == "true" {
				delete(resource.Item, "source_security_group_id")
			}
		} else if resource.InstanceInfo.Type == "aws_security_group" {
			if resource.Item["clearRules"] == true {
				delete(resource.Item, "ingress")
				delete(resource.Item, "egress")
				delete(resource.Item, "clearRules")
				continue
			}

			if val, ok := resource.Item["ingress"]; ok {
				g.sortRules(val.([]interface{}))
			}
			if val, ok := resource.Item["egress"]; ok {
				g.sortRules(val.([]interface{}))
			}
		}
	}
	return nil
}

func (g *SecurityGenerator) sortRules(rules []interface{}) {
	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		g.sortIfExist("cidr_blocks", ruleMap)
		g.sortIfExist("ipv6_cidr_blocks", ruleMap)
		g.sortIfExist("security_groups", ruleMap)
	}
	sort.Slice(rules, func(i, j int) bool {
		return fmt.Sprintf("%v", rules[i]) < fmt.Sprintf("%v", rules[j])
	})
}

func (g *SecurityGenerator) sortIfExist(attribute string, ruleMap map[string]interface{}) {
	if val, ok := ruleMap[attribute]; ok {
		sort.Slice(val.([]interface{}), func(i, j int) bool {
			return val.([]interface{})[i].(string) < val.([]interface{})[j].(string)
		})
	}
}

func permissionID(sgID, ruleType, groupID string, ip types.IpPermission) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s_%s_%s_%d_%d_", sgID, ruleType, *ip.IpProtocol, fromPort(ip), toPort(ip))

	if len(ip.IpRanges) > 0 {
		s := make([]string, len(ip.IpRanges))
		for i, r := range ip.IpRanges {
			s[i] = *r.CidrIp
		}
		sort.Strings(s)

		for _, v := range s {
			fmt.Fprintf(&buf, "%s_", v)
		}
	}

	if len(ip.Ipv6Ranges) > 0 {
		s := make([]string, len(ip.Ipv6Ranges))
		for i, r := range ip.Ipv6Ranges {
			s[i] = *r.CidrIpv6
		}
		sort.Strings(s)

		for _, v := range s {
			fmt.Fprintf(&buf, "%s_", v)
		}
	}

	if len(ip.PrefixListIds) > 0 {
		s := make([]string, len(ip.PrefixListIds))
		for i, pl := range ip.PrefixListIds {
			s[i] = *pl.PrefixListId
		}
		sort.Strings(s)

		for _, v := range s {
			fmt.Fprintf(&buf, "%s_", v)
		}
	}

	if groupID != "" {
		fmt.Fprintf(&buf, "%s_", groupID)
	}

	idPreformatted := buf.String()
	return idPreformatted[:len(idPreformatted)-1]
}

func fromPort(ip types.IpPermission) int {
	switch {
	case *ip.IpProtocol == "icmp":
		return -1
	case *ip.IpProtocol == "-1":
		return -1
	case ip.FromPort != nil && *ip.FromPort > 0:
		return int(*ip.FromPort)
	default:
		return 0
	}
}

func toPort(ip types.IpPermission) int {
	switch {
	case *ip.IpProtocol == "icmp":
		return -1
	case *ip.IpProtocol == "-1":
		return -1
	case ip.ToPort != nil && *ip.ToPort > 0:
		return int(*ip.ToPort)
	default:
		return 65536
	}
}

func ipRange(rule types.IpPermission) []string {
	result := make([]string, len(rule.IpRanges))
	for idx, rule := range rule.IpRanges {
		result[idx] = *rule.CidrIp
	}
	return result
}

func ip6Range(rule types.IpPermission) []string {
	result := make([]string, len(rule.Ipv6Ranges))
	for idx, rule := range rule.Ipv6Ranges {
		result[idx] = *rule.CidrIpv6
	}
	return result
}

func prefixes(rule types.IpPermission) []string {
	result := make([]string, len(rule.PrefixListIds))
	for idx, rule := range rule.PrefixListIds {
		result[idx] = *rule.PrefixListId
	}
	return result
}
