// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
