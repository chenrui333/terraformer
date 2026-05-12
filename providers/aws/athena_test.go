// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func TestAthenaPreparedStatementImportID(t *testing.T) {
	got := athenaPreparedStatementImportID("analytics", "daily_report")
	want := "analytics/daily_report"
	if got != want {
		t.Fatalf("athenaPreparedStatementImportID() = %q, want %q", got, want)
	}
}

func TestAthenaResourceName(t *testing.T) {
	got := athenaResourceName("primary", "", "orders")
	want := "primary/orders"
	if got != want {
		t.Fatalf("athenaResourceName() = %q, want %q", got, want)
	}
}

func TestAthenaNamedQueriesInputSetsWorkGroup(t *testing.T) {
	got := athenaNamedQueriesInput("analytics")
	if StringValue(got.WorkGroup) != "analytics" {
		t.Fatalf("WorkGroup = %q, want analytics", StringValue(got.WorkGroup))
	}
}

func TestNewAthenaDataCatalogResource(t *testing.T) {
	resource, ok := newAthenaDataCatalogResource(&athenatypes.DataCatalog{
		Name:        aws.String("analytics"),
		Description: aws.String("catalog"),
		Type:        athenatypes.DataCatalogTypeHive,
		Parameters:  map[string]string{"metadata-function": "arn:aws:lambda:us-east-1:123456789012:function:metadata"},
	})
	if !ok {
		t.Fatal("newAthenaDataCatalogResource() ok = false, want true")
	}
	if resource.InstanceInfo.Type != athenaDataCatalogResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, athenaDataCatalogResourceType)
	}
	if resource.InstanceState.ID != "analytics" {
		t.Fatalf("ID = %q, want analytics", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["parameters.metadata-function"]; got == "" {
		t.Fatal("parameters.metadata-function was not seeded")
	}
}

func TestNewAthenaDataCatalogResourceSkipsUnsafeShapes(t *testing.T) {
	tests := []struct {
		name    string
		catalog *athenatypes.DataCatalog
	}{
		{name: "nil", catalog: nil},
		{name: "default", catalog: &athenatypes.DataCatalog{Name: aws.String(athenaDefaultDataCatalogName), Type: athenatypes.DataCatalogTypeGlue, Parameters: map[string]string{"catalog-id": "123456789012"}}},
		{name: "empty parameters", catalog: &athenatypes.DataCatalog{Name: aws.String("empty"), Type: athenatypes.DataCatalogTypeHive}},
		{name: "secret parameters", catalog: &athenatypes.DataCatalog{Name: aws.String("secret"), Type: athenatypes.DataCatalogTypeFederated, Parameters: map[string]string{"connection-properties": "SecretArn=arn"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newAthenaDataCatalogResource(tt.catalog); ok {
				t.Fatal("newAthenaDataCatalogResource() ok = true, want false")
			}
		})
	}
}

func TestAthenaDataCatalogSummaryImportable(t *testing.T) {
	tests := []struct {
		name    string
		summary athenatypes.DataCatalogSummary
		want    bool
	}{
		{name: "create complete", summary: athenatypes.DataCatalogSummary{CatalogName: aws.String("analytics"), Status: athenatypes.DataCatalogStatusCreateComplete}, want: true},
		{name: "sync catalog empty status", summary: athenatypes.DataCatalogSummary{CatalogName: aws.String("analytics")}, want: true},
		{name: "default catalog", summary: athenatypes.DataCatalogSummary{CatalogName: aws.String(athenaDefaultDataCatalogName)}, want: false},
		{name: "creating", summary: athenatypes.DataCatalogSummary{CatalogName: aws.String("analytics"), Status: athenatypes.DataCatalogStatusCreateInProgress}, want: false},
		{name: "failed", summary: athenatypes.DataCatalogSummary{CatalogName: aws.String("analytics"), Status: athenatypes.DataCatalogStatusCreateFailed}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := athenaDataCatalogSummaryImportable(tt.summary)
			if got != tt.want {
				t.Fatalf("athenaDataCatalogSummaryImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAthenaNamedQueryResource(t *testing.T) {
	resource, ok := newAthenaNamedQueryResource(athenatypes.NamedQuery{
		NamedQueryId: aws.String("query-id"),
		Name:         aws.String("daily"),
		Database:     aws.String("analytics"),
		QueryString:  aws.String("select 1"),
		WorkGroup:    aws.String("primary"),
	})
	if !ok {
		t.Fatal("newAthenaNamedQueryResource() ok = false, want true")
	}
	if resource.InstanceState.ID != "query-id" {
		t.Fatalf("ID = %q, want query-id", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != athenaNamedQueryResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, athenaNamedQueryResourceType)
	}
}

func TestNewAthenaCapacityReservationResourcePreservesIDAfterRefresh(t *testing.T) {
	resource, ok := newAthenaCapacityReservationResource(athenatypes.CapacityReservation{
		Name:       aws.String("reserved"),
		Status:     athenatypes.CapacityReservationStatusActive,
		TargetDpus: aws.Int32(24),
	})
	if !ok {
		t.Fatal("newAthenaCapacityReservationResource() ok = false, want true")
	}
	assertAwsFrameworkResourcePreserveIDAfterRefresh(t, resource)
}

func TestAthenaCapacityReservationImportable(t *testing.T) {
	target := int32(24)
	belowMinimumTarget := int32(12)
	tests := []struct {
		name        string
		reservation athenatypes.CapacityReservation
		want        bool
	}{
		{name: "active", reservation: athenatypes.CapacityReservation{Name: aws.String("reserved"), Status: athenatypes.CapacityReservationStatusActive, TargetDpus: aws.Int32(target)}, want: true},
		{name: "pending", reservation: athenatypes.CapacityReservation{Name: aws.String("reserved"), Status: athenatypes.CapacityReservationStatusPending, TargetDpus: aws.Int32(target)}, want: false},
		{name: "below provider minimum", reservation: athenatypes.CapacityReservation{Name: aws.String("reserved"), Status: athenatypes.CapacityReservationStatusActive, TargetDpus: aws.Int32(belowMinimumTarget)}, want: false},
		{name: "zero dpu", reservation: athenatypes.CapacityReservation{Name: aws.String("reserved"), Status: athenatypes.CapacityReservationStatusActive}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := athenaCapacityReservationImportable(tt.reservation)
			if got != tt.want {
				t.Fatalf("athenaCapacityReservationImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAthenaAllowEmptyValuesPreserveRequiredAndDefaultedFields(t *testing.T) {
	resource := terraformutils.NewSimpleResource("analytics", "analytics", athenaWorkGroupResourceType, "aws", athenaAllowEmptyValues)
	assertAllowEmptyValue(t, resource, "^description$")
	assertAllowEmptyValue(t, resource, `^configuration\.\d+\.enforce_workgroup_configuration$`)
}

func TestAthenaPreparedStatementAllowEmptyValuesDropsEmptyDescription(t *testing.T) {
	resource := terraformutils.NewSimpleResource("primary/report", "primary/report", athenaPreparedStatementResourceType, "aws", athenaPreparedStatementAllowEmptyValues)
	assertAllowEmptyValueMissing(t, resource, "^description$")
}

func TestAthenaPostConvertHookWrapsQueries(t *testing.T) {
	resource := terraformutils.NewSimpleResource("query-id", "daily", athenaNamedQueryResourceType, "aws", athenaAllowEmptyValues)
	resource.Item = map[string]interface{}{"query": "select '$" + "{value}'"}
	g := &AthenaGenerator{AWSService: AWSService{}}
	g.Resources = []terraformutils.Resource{resource}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	want := "<<QUERY\nselect '$" + "$" + "{value}'\nQUERY"
	if got := g.Resources[0].Item["query"]; got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}

func assertAllowEmptyValue(t *testing.T, resource terraformutils.Resource, want string) {
	t.Helper()
	for _, value := range resource.AllowEmptyValues {
		if value == want {
			return
		}
	}
	t.Fatalf("AllowEmptyValues = %v, want %q", resource.AllowEmptyValues, want)
}

func assertAllowEmptyValueMissing(t *testing.T, resource terraformutils.Resource, forbidden string) {
	t.Helper()
	for _, value := range resource.AllowEmptyValues {
		if value == forbidden {
			t.Fatalf("AllowEmptyValues = %v, did not want %q", resource.AllowEmptyValues, forbidden)
		}
	}
}

func assertAwsFrameworkResourcePreserveIDAfterRefresh(t *testing.T, resource terraformutils.Resource) {
	t.Helper()
	if resource.InstanceState == nil {
		t.Fatal("InstanceState is nil")
	}
	preserveID, _ := resource.InstanceState.Meta[tfcompat.MetaKeyPreserveIDAfterRefresh].(bool)
	if !preserveID {
		t.Fatalf("InstanceState.Meta[%q] = %v, want true", tfcompat.MetaKeyPreserveIDAfterRefresh, resource.InstanceState.Meta)
	}
}
