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
