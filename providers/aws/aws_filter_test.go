// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAWSTypedIDFilterValues(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"cloud9_environment_ec2=env-123:env-456",
		"Name=id;Value=global-id",
		"Type=qldb_ledger;Name=id;Value=ledger-a",
		"Type=qldb_ledger;Name=name;Value=ignored",
	})

	cloud9IDs := awsTypedIDFilterValues(service.Filter, cloud9EnvironmentEC2ResourceType)
	if !awsIDFilterAllows(cloud9IDs, "env-123") || !awsIDFilterAllows(cloud9IDs, "env-456") {
		t.Fatalf("Cloud9 typed ID filter did not include expected environment IDs: %#v", cloud9IDs)
	}
	if awsIDFilterAllows(cloud9IDs, "env-789") {
		t.Fatalf("Cloud9 typed ID filter allowed unrelated environment ID: %#v", cloud9IDs)
	}

	qldbIDs := awsTypedIDFilterValues(service.Filter, qldbLedgerResourceType)
	if !awsIDFilterAllows(qldbIDs, "ledger-a") {
		t.Fatalf("QLDB typed ID filter did not include expected ledger: %#v", qldbIDs)
	}
	if awsIDFilterAllows(qldbIDs, "ledger-b") {
		t.Fatalf("QLDB typed ID filter allowed unrelated ledger: %#v", qldbIDs)
	}

	if dataPipelineIDs := awsTypedIDFilterValues(service.Filter, dataPipelinePipelineResourceType); dataPipelineIDs != nil {
		t.Fatalf("unexpected Data Pipeline typed ID filters: %#v", dataPipelineIDs)
	}
	if !awsIDFilterAllows(nil, "anything") {
		t.Fatal("missing typed ID filters should allow discovery")
	}
}

func TestAWSMergeIDFilterValues(t *testing.T) {
	merged := awsMergeIDFilterValues(
		map[string]bool{"first": true},
		nil,
		map[string]bool{"second": true},
	)
	for _, value := range []string{"first", "second"} {
		if !awsIDFilterAllows(merged, value) {
			t.Fatalf("merged ID filter should allow %q: %#v", value, merged)
		}
	}
	if awsIDFilterAllows(merged, "third") {
		t.Fatalf("merged ID filter allowed unrelated value: %#v", merged)
	}
	if got := awsMergeIDFilterValues(nil, map[string]bool{}); got != nil {
		t.Fatalf("empty merged ID filters = %#v, want nil", got)
	}
}

func TestAWSTypedFilterValuesAndApplicability(t *testing.T) {
	service := terraformutils.Service{}
	service.ParseFilters([]string{
		"Name=id;Value=global-id",
		"Name=name;Value=global-name",
		"Type=datapipeline_pipeline_definition;Name=pipeline_id;Value=df-123",
		"Type=cloud9_environment_membership;Name=id;Value=env-123#arn:aws:iam::123456789012:user/alice",
	})

	pipelineIDs := awsTypedFilterValues(service.Filter, dataPipelinePipelineDefinitionResourceType, "pipeline_id")
	if !awsIDFilterAllows(pipelineIDs, "df-123") {
		t.Fatalf("Data Pipeline typed field filter did not include expected pipeline ID: %#v", pipelineIDs)
	}
	if awsIDFilterAllows(pipelineIDs, "df-456") {
		t.Fatalf("Data Pipeline typed field filter allowed unrelated pipeline ID: %#v", pipelineIDs)
	}
	if !awsHasTypedFilter(service.Filter, cloud9EnvironmentMembershipResourceType) {
		t.Fatal("expected Cloud9 membership typed filter")
	}
	if !awsHasTypedNonIDFilter(service.Filter, dataPipelinePipelineDefinitionResourceType) {
		t.Fatal("expected Data Pipeline definition typed non-ID filter")
	}
	if !awsHasApplicableNonIDFilter(service.Filter, qldbLedgerResourceType) {
		t.Fatal("expected global non-ID filter to apply to QLDB ledger")
	}
	if awsHasTypedNonIDFilter(service.Filter, cloud9EnvironmentMembershipResourceType) {
		t.Fatal("unexpected Cloud9 membership typed non-ID filter")
	}
	if awsHasTypedFilter(service.Filter, qldbLedgerResourceType) {
		t.Fatal("unexpected QLDB ledger typed filter")
	}
	if !awsHasApplicableFilter(service.Filter, qldbLedgerResourceType) {
		t.Fatal("global filter should be applicable to QLDB ledger")
	}
	if !awsHasApplicableFilter(service.Filter, dataPipelinePipelineDefinitionResourceType) {
		t.Fatal("typed Data Pipeline definition filter should be applicable to definitions")
	}
}
