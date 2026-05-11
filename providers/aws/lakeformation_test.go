// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lakeformation"
	lakeformationtypes "github.com/aws/aws-sdk-go-v2/service/lakeformation/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestLakeFormationImportIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "lf tag", got: lakeFormationLFTagImportID("123456789012", "pii"), want: "123456789012:pii"},
		{name: "lf tag expression", got: lakeFormationLFTagExpressionImportID("sensitive", "123456789012"), want: "sensitive,123456789012"},
		{name: "data cells filter", got: lakeFormationDataCellsFilterImportID("db", "filter", "123456789012", "table"), want: "db,filter,123456789012,table"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("import ID = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestLakeFormationDataLakeSettingsImportable(t *testing.T) {
	tests := []struct {
		name     string
		settings *lakeformationtypes.DataLakeSettings
		want     bool
	}{
		{name: "nil", settings: nil, want: false},
		{name: "empty", settings: &lakeformationtypes.DataLakeSettings{}, want: false},
		{name: "default parameter", settings: &lakeformationtypes.DataLakeSettings{Parameters: map[string]string{"CROSS_ACCOUNT_VERSION": "1"}}, want: false},
		{name: "default IAM permissions", settings: &lakeformationtypes.DataLakeSettings{CreateDatabaseDefaultPermissions: []lakeformationtypes.PrincipalPermissions{{Principal: &lakeformationtypes.DataLakePrincipal{DataLakePrincipalIdentifier: aws.String("IAM_ALLOWED_PRINCIPALS")}, Permissions: []lakeformationtypes.Permission{lakeformationtypes.PermissionAll}}}}, want: false},
		{name: "configured permissions", settings: &lakeformationtypes.DataLakeSettings{CreateDatabaseDefaultPermissions: []lakeformationtypes.PrincipalPermissions{{Principal: &lakeformationtypes.DataLakePrincipal{DataLakePrincipalIdentifier: aws.String("arn:aws:iam::123456789012:role/admin")}, Permissions: []lakeformationtypes.Permission{lakeformationtypes.PermissionAll}}}}, want: true},
		{name: "configured parameter", settings: &lakeformationtypes.DataLakeSettings{Parameters: map[string]string{"CROSS_ACCOUNT_VERSION": "4"}}, want: true},
		{name: "admin", settings: &lakeformationtypes.DataLakeSettings{DataLakeAdmins: []lakeformationtypes.DataLakePrincipal{{DataLakePrincipalIdentifier: aws.String("arn:aws:iam::123456789012:role/admin")}}}, want: true},
		{name: "external filtering", settings: &lakeformationtypes.DataLakeSettings{AllowExternalDataFiltering: aws.Bool(true)}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lakeFormationDataLakeSettingsImportable(tt.settings)
			if got != tt.want {
				t.Fatalf("lakeFormationDataLakeSettingsImportable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLakeFormationLFTagResource(t *testing.T) {
	resource, ok := newLakeFormationLFTagResource("123456789012", lakeformationtypes.LFTagPair{
		TagKey:    aws.String("classification"),
		TagValues: []string{"public", "private"},
	})
	if !ok {
		t.Fatal("newLakeFormationLFTagResource() ok = false, want true")
	}
	if resource.InstanceState.ID != "123456789012:classification" {
		t.Fatalf("ID = %q, want 123456789012:classification", resource.InstanceState.ID)
	}
	if resource.InstanceInfo.Type != lakeFormationLFTagResourceType {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, lakeFormationLFTagResourceType)
	}
}

func TestNewLakeFormationLFTagExpressionResource(t *testing.T) {
	resource, ok := newLakeFormationLFTagExpressionResource("123456789012", lakeformationtypes.LFTagExpression{
		Name:       aws.String("sensitive"),
		Expression: []lakeformationtypes.LFTag{{TagKey: aws.String("classification"), TagValues: []string{"private"}}},
	})
	if !ok {
		t.Fatal("newLakeFormationLFTagExpressionResource() ok = false, want true")
	}
	if resource.InstanceState.ID != "sensitive,123456789012" {
		t.Fatalf("ID = %q, want sensitive,123456789012", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["catalog_id"]; got != "123456789012" {
		t.Fatalf("catalog_id = %q, want 123456789012", got)
	}
}

func TestNewLakeFormationDataCellsFilterResource(t *testing.T) {
	resource, ok := newLakeFormationDataCellsFilterResource(lakeformationtypes.DataCellsFilter{
		DatabaseName:   aws.String("analytics"),
		Name:           aws.String("orders_filter"),
		TableCatalogId: aws.String("123456789012"),
		TableName:      aws.String("orders"),
	})
	if !ok {
		t.Fatal("newLakeFormationDataCellsFilterResource() ok = false, want true")
	}
	if resource.InstanceState.ID != "analytics,orders_filter,123456789012,orders" {
		t.Fatalf("ID = %q, want analytics,orders_filter,123456789012,orders", resource.InstanceState.ID)
	}
}

func TestLakeFormationPostConvertHookPreservesDataCellsFilterWildcardBlocks(t *testing.T) {
	resource := terraformutils.NewSimpleResource(
		"analytics,orders_filter,123456789012,orders",
		"orders_filter",
		lakeFormationDataCellsFilterResourceType,
		"aws",
		lakeFormationAllowEmptyValues,
	)
	resource.InstanceState.Attributes = map[string]string{
		"table_data.#":                                  "1",
		"table_data.0.column_wildcard.#":                "1",
		"table_data.0.row_filter.#":                     "1",
		"table_data.0.row_filter.0.all_rows_wildcard.#": "1",
	}
	resource.Item = map[string]interface{}{
		"table_data": []interface{}{map[string]interface{}{
			"database_name":    "analytics",
			"name":             "orders_filter",
			"row_filter":       []interface{}{map[string]interface{}{"filter_expression": "TRUE"}},
			"table_catalog_id": "123456789012",
			"table_name":       "orders",
		}},
	}
	g := &LakeFormationGenerator{AWSService: AWSService{}}
	g.Resources = []terraformutils.Resource{resource}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	data, err := terraformutils.HclPrintResource(g.Resources, map[string]interface{}{}, "hcl", true)
	if err != nil {
		t.Fatalf("HclPrintResource() error = %v", err)
	}
	hcl := string(data)
	for _, want := range []string{"column_wildcard {", "all_rows_wildcard {"} {
		if !strings.Contains(hcl, want) {
			t.Fatalf("generated HCL missing %q:\n%s", want, hcl)
		}
	}
	if strings.Contains(hcl, "filter_expression") {
		t.Fatalf("generated HCL retained filter_expression with all_rows_wildcard:\n%s", hcl)
	}
}

func TestNewLakeFormationIdentityCenterConfigurationResource(t *testing.T) {
	resource, ok := newLakeFormationIdentityCenterConfigurationResource("123456789012", &lakeformation.DescribeLakeFormationIdentityCenterConfigurationOutput{
		InstanceArn:    aws.String("arn:aws:sso:::instance/ssoins-123"),
		ApplicationArn: aws.String("arn:aws:sso::123456789012:application/ssoins-123/apl-123"),
	})
	if !ok {
		t.Fatal("newLakeFormationIdentityCenterConfigurationResource() ok = false, want true")
	}
	if resource.InstanceState.ID != "123456789012" {
		t.Fatalf("ID = %q, want 123456789012", resource.InstanceState.ID)
	}
	if got := resource.InstanceState.Attributes["instance_arn"]; got == "" {
		t.Fatal("instance_arn was not seeded")
	}
}
