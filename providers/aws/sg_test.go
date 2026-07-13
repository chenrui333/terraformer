// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestEmptySgs(t *testing.T) {
	var securityGroups []types.SecurityGroup

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if !reflect.DeepEqual(rulesToMoveOut, []string{}) {
		t.Errorf("failed to calculate rules to move out %v", rulesToMoveOut)
	}
}

func Test1CycleReference(t *testing.T) {
	sgA := types.SecurityGroup{
		GroupId: aws.String("aaaa"),
		IpPermissions: []types.IpPermission{
			{
				UserIdGroupPairs: []types.UserIdGroupPair{
					{
						GroupId: aws.String("aaaa"),
					},
				},
			},
			{},
		},
	}
	securityGroups := []types.SecurityGroup{
		sgA,
	}

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if !reflect.DeepEqual(rulesToMoveOut, []string{}) {
		t.Errorf("failed to calculate rules to move out %v", rulesToMoveOut)
	}
}

func Test2CycleReference(t *testing.T) {
	sgA := types.SecurityGroup{
		GroupId: aws.String("aaaa"),
		IpPermissions: []types.IpPermission{
			{
				UserIdGroupPairs: []types.UserIdGroupPair{
					{
						GroupId: aws.String("bbbb"),
					},
				},
			},
		},
	}
	securityGroups := []types.SecurityGroup{
		{
			GroupId: aws.String("bbbb"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("aaaa"),
						},
					},
				},
				{},
			},
		},
		sgA,
	}

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if !reflect.DeepEqual(rulesToMoveOut[0], *sgA.GroupId) {
		t.Errorf("failed to calculate rules to move out %v", rulesToMoveOut)
	}
}

func TestNoCycleReference(t *testing.T) {
	sgA := types.SecurityGroup{
		GroupId: aws.String("aaaa"),
		IpPermissions: []types.IpPermission{
			{
				UserIdGroupPairs: []types.UserIdGroupPair{
					{
						GroupId: aws.String("bbbb"),
					},
				},
			},
		},
	}
	securityGroups := []types.SecurityGroup{
		{
			GroupId: aws.String("bbbb"),
			IpPermissions: []types.IpPermission{
				{},
				{},
			},
		},
		sgA,
	}

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if len(rulesToMoveOut) != 0 {
		t.Errorf("failed to calculate rules to move out %v", rulesToMoveOut)
	}
}

func TestNoCycleReferenceToExternalGroup(t *testing.T) {
	securityGroups := []types.SecurityGroup{
		{
			GroupId: aws.String("aaaa"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("bbbb"),
						},
					},
				},
			},
		},
		{
			GroupId: aws.String("bbbb"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("external"),
						},
					},
				},
			},
		},
	}

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if len(rulesToMoveOut) != 0 {
		t.Errorf("failed to ignore external security group reference: %v", rulesToMoveOut)
	}
}

func Test3Cycle1CycleReference(t *testing.T) {
	sgA := types.SecurityGroup{
		GroupId: aws.String("aaaa"),
		IpPermissions: []types.IpPermission{
			{
				UserIdGroupPairs: []types.UserIdGroupPair{
					{
						GroupId: aws.String("aaaa"),
					},
				},
			},
			{
				UserIdGroupPairs: []types.UserIdGroupPair{
					{
						GroupId: aws.String("bbbb"),
					},
				},
			},
		},
	}
	securityGroups := []types.SecurityGroup{
		sgA,
		{
			GroupId: aws.String("bbbb"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("cccc"),
						},
					},
				},
				{},
			},
		},
		{
			GroupId: aws.String("cccc"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("aaaa"),
						},
					},
				},
				{},
			},
		},
		{
			GroupId: aws.String("dddd"),
			IpPermissions: []types.IpPermission{
				{
					UserIdGroupPairs: []types.UserIdGroupPair{
						{
							GroupId: aws.String("aaaa"),
						},
					},
				},
				{},
			},
		},
	}

	rulesToMoveOut := findSgsToMoveOut(securityGroups)

	if !reflect.DeepEqual(rulesToMoveOut[0], *sgA.GroupId) {
		t.Errorf("failed to calculate rules to move out %v", rulesToMoveOut)
	}
}

func TestSecurityGroupEgressAndMixedCycleDetection(t *testing.T) {
	tests := []struct {
		name   string
		groups []types.SecurityGroup
		want   []string
	}{
		{
			name: "egress two group cycle",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{"aaaa"},
		},
		{
			name: "egress three group cycle",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("cccc")}),
				testSecurityGroup("cccc", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{"aaaa"},
		},
		{
			name: "mixed two group cycle",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermission("bbbb")}, nil),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{"aaaa"},
		},
		{
			name: "mixed three group cycle",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermission("bbbb")}, nil),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("cccc")}),
				testSecurityGroup("cccc", []types.IpPermission{securityGroupPermission("aaaa")}, nil),
			},
			want: []string{"aaaa"},
		},
		{
			name: "acyclic egress reference",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
				testSecurityGroup("bbbb", nil, nil),
			},
			want: []string{},
		},
		{
			name: "external egress reference",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("external")}),
			},
			want: []string{},
		},
		{
			name: "egress self reference",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{},
		},
		{
			name: "nil referenced group ID",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermissionPointer(nil)}, nil),
			},
			want: []string{},
		},
		{
			name: "empty referenced group ID",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("")}),
			},
			want: []string{},
		},
		{
			name: "nil security group ID",
			groups: []types.SecurityGroup{
				{IpPermissions: []types.IpPermission{securityGroupPermission("aaaa")}},
				testSecurityGroup("aaaa", nil, nil),
			},
			want: []string{},
		},
		{
			name: "empty security group ID",
			groups: []types.SecurityGroup{
				testSecurityGroup("", []types.IpPermission{securityGroupPermission("aaaa")}, nil),
				testSecurityGroup("aaaa", nil, nil),
			},
			want: []string{},
		},
		{
			name: "groups without rules",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, nil),
				testSecurityGroup("bbbb", nil, nil),
			},
			want: []string{},
		},
		{
			name: "duplicate ingress and egress references",
			groups: []types.SecurityGroup{
				testSecurityGroup(
					"aaaa",
					[]types.IpPermission{securityGroupPermission("bbbb"), securityGroupPermission("bbbb")},
					[]types.IpPermission{securityGroupPermission("bbbb")},
				),
				testSecurityGroup("bbbb", []types.IpPermission{securityGroupPermission("aaaa")}, nil),
			},
			want: []string{"bbbb"},
		},
		{
			name: "internal and external reference in one permission",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb", "external")}),
				testSecurityGroup("bbbb", nil, nil),
			},
			want: []string{},
		},
		{
			name: "fewest total rules wins",
			groups: []types.SecurityGroup{
				testSecurityGroup(
					"aaaa",
					[]types.IpPermission{securityGroupPermission("bbbb")},
					[]types.IpPermission{securityGroupPermission("external")},
				),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{"bbbb"},
		},
		{
			name: "equal rule counts use group ID tie breaker",
			groups: []types.SecurityGroup{
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
				testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermission("bbbb")}, nil),
			},
			want: []string{"aaaa"},
		},
		{
			name: "duplicate security group entries are collapsed",
			groups: []types.SecurityGroup{
				testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermission("bbbb")}, nil),
				testSecurityGroup("aaaa", nil, nil),
				testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
			},
			want: []string{"aaaa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSgsToMoveOut(tt.groups)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("findSgsToMoveOut() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecurityGroupCycleSelectionIsDeterministic(t *testing.T) {
	groups := []types.SecurityGroup{
		testSecurityGroup("dddd", nil, []types.IpPermission{securityGroupPermission("cccc")}),
		testSecurityGroup("bbbb", []types.IpPermission{securityGroupPermission("aaaa")}, nil),
		testSecurityGroup("cccc", []types.IpPermission{securityGroupPermission("dddd")}, nil),
		testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
	}
	want := []string{"aaaa", "cccc"}

	for i := 0; i < 100; i++ {
		got := findSgsToMoveOut(groups)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: findSgsToMoveOut() = %v, want %v", i, got, want)
		}
		if !sort.StringsAreSorted(got) {
			t.Fatalf("run %d: result is not sorted: %v", i, got)
		}
	}
}

func TestSecurityGroupCycleSelectionIsIndependentOfInputOrder(t *testing.T) {
	groupA := testSecurityGroup("aaaa", []types.IpPermission{securityGroupPermission("bbbb")}, nil)
	groupB := testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")})

	for _, groups := range [][]types.SecurityGroup{{groupA, groupB}, {groupB, groupA}} {
		got := findSgsToMoveOut(groups)
		want := []string{"aaaa"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("findSgsToMoveOut(%v) = %v, want %v", securityGroupIDs(groups), got, want)
		}
	}
}

func TestSecurityGroupDenseCycleDetection(t *testing.T) {
	groups := denseSecurityGroups(10)
	want := []string{
		"sg-00", "sg-01", "sg-02", "sg-03", "sg-04",
		"sg-05", "sg-06", "sg-07", "sg-08",
	}

	got := findSgsToMoveOut(groups)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findSgsToMoveOut() = %v, want %v", got, want)
	}
}

func BenchmarkSecurityGroupDenseCycleDetection(b *testing.B) {
	groups := denseSecurityGroups(10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findSgsToMoveOut(groups)
	}
}

func TestSecurityGroupEgressCycleCreatesStandaloneRules(t *testing.T) {
	disableSplitSecurityGroupRules(t)
	groups := []types.SecurityGroup{
		testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
		testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
	}

	resources := (SecurityGenerator{}).createResources(groups)
	assertSecurityGroupResourceCounts(t, resources, 1)
	standaloneRules := resourcesOfType(resources, "aws_security_group_rule")
	if got := standaloneRules[0].InstanceState.Attributes["type"]; got != "egress" {
		t.Fatalf("standalone rule type = %q, want egress", got)
	}
	if !strings.HasPrefix(standaloneRules[0].InstanceState.ID, "aaaa_egress_") {
		t.Fatalf("standalone rule ID = %q, want selected group aaaa egress rule", standaloneRules[0].InstanceState.ID)
	}
	assertSecurityGroupRulesCleared(t, resources, "aaaa", true)
	assertSecurityGroupRulesCleared(t, resources, "bbbb", false)
}

func TestSecurityGroupAcyclicEgressKeepsInlineRules(t *testing.T) {
	disableSplitSecurityGroupRules(t)
	groups := []types.SecurityGroup{
		testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
		testSecurityGroup("bbbb", nil, nil),
	}

	resources := (SecurityGenerator{}).createResources(groups)
	assertSecurityGroupResourceCounts(t, resources, 0)
	assertSecurityGroupRulesCleared(t, resources, "aaaa", false)
	assertSecurityGroupRulesCleared(t, resources, "bbbb", false)
}

func TestSecurityGroupResourceGenerationSkipsNilReferences(t *testing.T) {
	disableSplitSecurityGroupRules(t)
	groups := []types.SecurityGroup{
		testSecurityGroup("aaaa", []types.IpPermission{
			securityGroupPermissionPointers(nil, aws.String("bbbb")),
		}, nil),
		testSecurityGroup("bbbb", []types.IpPermission{securityGroupPermission("aaaa")}, nil),
	}

	resources := (SecurityGenerator{}).createResources(groups)
	assertSecurityGroupResourceCounts(t, resources, 1)
}

func TestSecurityGroupSplitRulePreservesCrossAccountOwner(t *testing.T) {
	disableSplitSecurityGroupRules(t)
	crossAccountPermission := securityGroupPermission("bbbb")
	crossAccountPermission.UserIdGroupPairs = append(crossAccountPermission.UserIdGroupPairs, types.UserIdGroupPair{
		GroupId: aws.String("external"),
		UserId:  aws.String("test-owner"),
	})
	groups := []types.SecurityGroup{
		testSecurityGroup("aaaa", nil, []types.IpPermission{crossAccountPermission}),
		testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("aaaa")}),
	}

	resources := (SecurityGenerator{}).createResources(groups)
	assertSecurityGroupResourceCounts(t, resources, 2)
	var sources []string
	for _, resource := range resourcesOfType(resources, "aws_security_group_rule") {
		sources = append(sources, resource.InstanceState.Attributes["source_security_group_id"])
	}
	sort.Strings(sources)
	want := []string{"bbbb", "test-owner/external"}
	if !reflect.DeepEqual(sources, want) {
		t.Fatalf("split rule sources = %v, want %v", sources, want)
	}
}

func TestSplitSecurityGroupRulesPreservesOverrideBehavior(t *testing.T) {
	t.Setenv("SPLIT_SG_RULES", "1")
	groups := []types.SecurityGroup{
		{VpcId: aws.String("vpc-test")},
		testSecurityGroup("aaaa", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
		testSecurityGroup("bbbb", nil, []types.IpPermission{securityGroupPermission("bbbb")}),
	}

	resources := (SecurityGenerator{}).createResources(groups)
	assertSecurityGroupResourceCounts(t, resources, 2)
	assertSecurityGroupRulesCleared(t, resources, "aaaa", true)
	assertSecurityGroupRulesCleared(t, resources, "bbbb", true)
}

func testSecurityGroup(id string, ingress, egress []types.IpPermission) types.SecurityGroup {
	return types.SecurityGroup{
		GroupId:             aws.String(id),
		GroupName:           aws.String(id),
		VpcId:               aws.String("vpc-test"),
		IpPermissions:       ingress,
		IpPermissionsEgress: egress,
	}
}

func denseSecurityGroups(count int) []types.SecurityGroup {
	groupIDs := make([]string, count)
	for i := range groupIDs {
		groupIDs[i] = fmt.Sprintf("sg-%02d", i)
	}

	groups := make([]types.SecurityGroup, 0, count)
	for _, groupID := range groupIDs {
		references := make([]string, 0, count-1)
		for _, referencedGroupID := range groupIDs {
			if referencedGroupID != groupID {
				references = append(references, referencedGroupID)
			}
		}
		groups = append(groups, testSecurityGroup(
			groupID,
			nil,
			[]types.IpPermission{securityGroupPermission(references...)},
		))
	}
	return groups
}

func securityGroupPermission(groupIDs ...string) types.IpPermission {
	pointers := make([]*string, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		pointers = append(pointers, aws.String(groupID))
	}
	return securityGroupPermissionPointers(pointers...)
}

func securityGroupPermissionPointer(groupID *string) types.IpPermission {
	return securityGroupPermissionPointers(groupID)
}

func securityGroupPermissionPointers(groupIDs ...*string) types.IpPermission {
	pairs := make([]types.UserIdGroupPair, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		pairs = append(pairs, types.UserIdGroupPair{GroupId: groupID})
	}
	return types.IpPermission{
		IpProtocol:       aws.String("-1"),
		UserIdGroupPairs: pairs,
	}
}

func securityGroupIDs(groups []types.SecurityGroup) []string {
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, StringValue(group.GroupId))
	}
	return ids
}

func disableSplitSecurityGroupRules(t *testing.T) {
	t.Helper()
	value, wasSet := os.LookupEnv("SPLIT_SG_RULES")
	if err := os.Unsetenv("SPLIT_SG_RULES"); err != nil {
		t.Fatalf("unset SPLIT_SG_RULES: %v", err)
	}
	t.Cleanup(func() {
		if wasSet {
			if err := os.Setenv("SPLIT_SG_RULES", value); err != nil {
				t.Errorf("restore SPLIT_SG_RULES: %v", err)
			}
			return
		}
		if err := os.Unsetenv("SPLIT_SG_RULES"); err != nil {
			t.Errorf("unset SPLIT_SG_RULES during cleanup: %v", err)
		}
	})
}

func resourcesOfType(resources []terraformutils.Resource, resourceType string) []terraformutils.Resource {
	var matching []terraformutils.Resource
	for _, resource := range resources {
		if resource.InstanceInfo.Type == resourceType {
			matching = append(matching, resource)
		}
	}
	return matching
}

func assertSecurityGroupResourceCounts(t *testing.T, resources []terraformutils.Resource, rules int) {
	t.Helper()
	if got := len(resourcesOfType(resources, "aws_security_group")); got != 2 {
		t.Fatalf("security group resource count = %d, want 2", got)
	}
	if got := len(resourcesOfType(resources, "aws_security_group_rule")); got != rules {
		t.Fatalf("security group rule resource count = %d, want %d", got, rules)
	}
}

func assertSecurityGroupRulesCleared(t *testing.T, resources []terraformutils.Resource, groupID string, want bool) {
	t.Helper()
	for _, resource := range resourcesOfType(resources, "aws_security_group") {
		if resource.InstanceState.ID != groupID {
			continue
		}
		got, _ := resource.AdditionalFields["clearRules"].(bool)
		if got != want {
			t.Fatalf("security group %s clearRules = %t, want %t", groupID, got, want)
		}
		return
	}
	t.Fatalf("security group resource %s not found", groupID)
}
