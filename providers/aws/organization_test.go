// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	organizationtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
)

func TestOrganizationTraverseNodeReturnsAccountErrors(t *testing.T) {
	listErr := errors.New("accounts unavailable")
	generator := &OrganizationGenerator{}

	err := generator.traverseNode(&fakeOrganizationClient{accountErr: listErr}, "r-root")
	if err == nil {
		t.Fatal("expected account listing error")
	}
	if !strings.Contains(err.Error(), "list organization accounts for parent r-root") {
		t.Fatalf("error = %q, want account listing context", err)
	}
	if !errors.Is(err, listErr) {
		t.Fatalf("error does not wrap account listing error: %v", err)
	}
}

func TestOrganizationTraverseNodePaginatesChildren(t *testing.T) {
	generator := &OrganizationGenerator{}
	client := &fakeOrganizationClient{
		accountsByParent: map[string][]*organizations.ListAccountsForParentOutput{
			"r-root": {
				{
					Accounts: []organizationtypes.Account{
						{Id: aws.String("111111111111"), Name: aws.String("dev"), Arn: aws.String("arn:aws:organizations::111111111111:account/o-example/111111111111")},
					},
					NextToken: aws.String("more"),
				},
				{
					Accounts: []organizationtypes.Account{
						{Id: aws.String("222222222222"), Name: aws.String("prod"), Arn: aws.String("arn:aws:organizations::222222222222:account/o-example/222222222222")},
					},
				},
			},
			"ou-root-child": {
				{
					Accounts: []organizationtypes.Account{
						{Id: aws.String("333333333333"), Name: aws.String("shared"), Arn: aws.String("arn:aws:organizations::333333333333:account/o-example/333333333333")},
					},
				},
			},
		},
		unitsByParent: map[string][]*organizations.ListOrganizationalUnitsForParentOutput{
			"r-root": {
				{
					OrganizationalUnits: []organizationtypes.OrganizationalUnit{
						{Id: aws.String("ou-root-child"), Name: aws.String("child"), Arn: aws.String("arn:aws:organizations::123456789012:ou/o-example/ou-root-child")},
					},
				},
			},
		},
	}

	if err := generator.traverseNode(client, "r-root"); err != nil {
		t.Fatalf("traverseNode returned error: %v", err)
	}
	if len(generator.Resources) != 7 {
		t.Fatalf("len(Resources) = %d, want 7", len(generator.Resources))
	}
	if client.accountCalls["r-root"] != 2 {
		t.Fatalf("root account calls = %d, want 2", client.accountCalls["r-root"])
	}
	if _, ok := client.accountCalls["ou-root-child"]; !ok {
		t.Fatal("child organizational unit was not traversed")
	}
}

func TestOrganizationAddPolicyAttachmentsReturnsTargetErrors(t *testing.T) {
	listErr := errors.New("targets unavailable")
	generator := &OrganizationGenerator{}

	err := generator.addPolicyAttachments(&fakeOrganizationClient{targetErr: listErr}, "p-policy", "policy")
	if err == nil {
		t.Fatal("expected policy target listing error")
	}
	if !strings.Contains(err.Error(), "list organization targets for policy p-policy") {
		t.Fatalf("error = %q, want policy target context", err)
	}
	if !errors.Is(err, listErr) {
		t.Fatalf("error does not wrap policy target listing error: %v", err)
	}
}

func TestOrganizationAddPolicyAttachmentsPaginatesTargets(t *testing.T) {
	generator := &OrganizationGenerator{}
	client := &fakeOrganizationClient{
		targetsByPolicy: map[string][]*organizations.ListTargetsForPolicyOutput{
			"p-policy": {
				{
					Targets: []organizationtypes.PolicyTargetSummary{
						{TargetId: aws.String("111111111111")},
					},
					NextToken: aws.String("more"),
				},
				{
					Targets: []organizationtypes.PolicyTargetSummary{
						{TargetId: aws.String("ou-root-child")},
					},
				},
			},
		},
	}

	if err := generator.addPolicyAttachments(client, "p-policy", "policy"); err != nil {
		t.Fatalf("addPolicyAttachments returned error: %v", err)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("len(Resources) = %d, want 2", len(generator.Resources))
	}
	if client.targetCalls["p-policy"] != 2 {
		t.Fatalf("target calls = %d, want 2", client.targetCalls["p-policy"])
	}
}

type fakeOrganizationClient struct {
	accountsByParent map[string][]*organizations.ListAccountsForParentOutput
	unitsByParent    map[string][]*organizations.ListOrganizationalUnitsForParentOutput
	targetsByPolicy  map[string][]*organizations.ListTargetsForPolicyOutput
	accountCalls     map[string]int
	unitCalls        map[string]int
	targetCalls      map[string]int
	accountErr       error
	unitErr          error
	targetErr        error
}

func (c *fakeOrganizationClient) ListAccountsForParent(_ context.Context, input *organizations.ListAccountsForParentInput, _ ...func(*organizations.Options)) (*organizations.ListAccountsForParentOutput, error) {
	if c.accountErr != nil {
		return nil, c.accountErr
	}
	parentID := aws.ToString(input.ParentId)
	call := c.incrementAccountCall(parentID)
	return outputAt(c.accountsByParent[parentID], call), nil
}

func (c *fakeOrganizationClient) ListOrganizationalUnitsForParent(_ context.Context, input *organizations.ListOrganizationalUnitsForParentInput, _ ...func(*organizations.Options)) (*organizations.ListOrganizationalUnitsForParentOutput, error) {
	if c.unitErr != nil {
		return nil, c.unitErr
	}
	parentID := aws.ToString(input.ParentId)
	call := c.incrementUnitCall(parentID)
	return outputAt(c.unitsByParent[parentID], call), nil
}

func (c *fakeOrganizationClient) ListTargetsForPolicy(_ context.Context, input *organizations.ListTargetsForPolicyInput, _ ...func(*organizations.Options)) (*organizations.ListTargetsForPolicyOutput, error) {
	if c.targetErr != nil {
		return nil, c.targetErr
	}
	policyID := aws.ToString(input.PolicyId)
	call := c.incrementTargetCall(policyID)
	return outputAt(c.targetsByPolicy[policyID], call), nil
}

func (c *fakeOrganizationClient) incrementAccountCall(parentID string) int {
	if c.accountCalls == nil {
		c.accountCalls = map[string]int{}
	}
	call := c.accountCalls[parentID]
	c.accountCalls[parentID]++
	return call
}

func (c *fakeOrganizationClient) incrementUnitCall(parentID string) int {
	if c.unitCalls == nil {
		c.unitCalls = map[string]int{}
	}
	call := c.unitCalls[parentID]
	c.unitCalls[parentID]++
	return call
}

func (c *fakeOrganizationClient) incrementTargetCall(policyID string) int {
	if c.targetCalls == nil {
		c.targetCalls = map[string]int{}
	}
	call := c.targetCalls[policyID]
	c.targetCalls[policyID]++
	return call
}

func outputAt[T any](outputs []*T, index int) *T {
	if index >= len(outputs) || outputs[index] == nil {
		return new(T)
	}
	return outputs[index]
}
