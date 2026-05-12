// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/vpclattice"
	vpclatticetypes "github.com/aws/aws-sdk-go-v2/service/vpclattice/types"
	"github.com/aws/smithy-go"
)

func TestVPCLatticeResourceConstructors(t *testing.T) {
	tests := []struct {
		name       string
		resource   terraformResourceResult
		wantID     string
		wantType   string
		wantAttr   map[string]string
		wantExists bool
	}{
		{
			name: "service network",
			resource: newTerraformResourceResult(newVPCLatticeServiceNetworkResource(vpclatticetypes.ServiceNetworkSummary{
				Id:   aws.String("sn-123"),
				Name: aws.String("network"),
			})),
			wantID:     "sn-123",
			wantType:   vpclatticeServiceNetworkResourceType,
			wantExists: true,
		},
		{
			name: "service active",
			resource: newTerraformResourceResult(newVPCLatticeServiceResource(vpclatticetypes.ServiceSummary{
				Id:     aws.String("svc-123"),
				Name:   aws.String("service"),
				Status: vpclatticetypes.ServiceStatusActive,
			})),
			wantID:     "svc-123",
			wantType:   vpclatticeServiceResourceType,
			wantExists: true,
		},
		{
			name: "service deleting",
			resource: newTerraformResourceResult(newVPCLatticeServiceResource(vpclatticetypes.ServiceSummary{
				Id:     aws.String("svc-123"),
				Status: vpclatticetypes.ServiceStatusDeleteInProgress,
			})),
			wantExists: false,
		},
		{
			name: "target group active",
			resource: newTerraformResourceResult(newVPCLatticeTargetGroupResource(vpclatticetypes.TargetGroupSummary{
				Id:     aws.String("tg-123"),
				Name:   aws.String("targets"),
				Status: vpclatticetypes.TargetGroupStatusActive,
			})),
			wantID:     "tg-123",
			wantType:   vpclatticeTargetGroupResourceType,
			wantExists: true,
		},
		{
			name: "listener",
			resource: newTerraformResourceResult(newVPCLatticeListenerResource("svc-123", vpclatticetypes.ListenerSummary{
				Id:   aws.String("listener-123"),
				Name: aws.String("https"),
			})),
			wantID:     "svc-123/listener-123",
			wantType:   vpclatticeListenerResourceType,
			wantAttr:   map[string]string{"service_identifier": "svc-123"},
			wantExists: true,
		},
		{
			name: "listener rule",
			resource: newTerraformResourceResult(newVPCLatticeListenerRuleResource("svc-123", "listener-123", vpclatticetypes.RuleSummary{
				Id:   aws.String("rule-123"),
				Name: aws.String("path"),
			})),
			wantID:     "svc-123/listener-123/rule-123",
			wantType:   vpclatticeListenerRuleResourceType,
			wantAttr:   map[string]string{"service_identifier": "svc-123", "listener_identifier": "listener-123"},
			wantExists: true,
		},
		{
			name: "default listener rule skipped",
			resource: newTerraformResourceResult(newVPCLatticeListenerRuleResource("svc-123", "listener-123", vpclatticetypes.RuleSummary{
				Id:        aws.String("rule-123"),
				IsDefault: aws.Bool(true),
			})),
			wantExists: false,
		},
		{
			name: "service network service association",
			resource: newTerraformResourceResult(newVPCLatticeServiceNetworkServiceAssociationResource(vpclatticetypes.ServiceNetworkServiceAssociationSummary{
				Id:               aws.String("snsa-123"),
				CreatedBy:        aws.String("123456789012"),
				ServiceId:        aws.String("svc-123"),
				ServiceNetworkId: aws.String("sn-123"),
				Status:           vpclatticetypes.ServiceNetworkServiceAssociationStatusActive,
			}, "123456789012")),
			wantID:     "snsa-123",
			wantType:   vpclatticeServiceNetworkServiceAssociationResourceType,
			wantAttr:   map[string]string{"service_identifier": "svc-123", "service_network_identifier": "sn-123"},
			wantExists: true,
		},
		{
			name: "service network service association from another account",
			resource: newTerraformResourceResult(newVPCLatticeServiceNetworkServiceAssociationResource(vpclatticetypes.ServiceNetworkServiceAssociationSummary{
				Id:               aws.String("snsa-123"),
				CreatedBy:        aws.String("210987654321"),
				ServiceId:        aws.String("svc-123"),
				ServiceNetworkId: aws.String("sn-123"),
				Status:           vpclatticetypes.ServiceNetworkServiceAssociationStatusActive,
			}, "123456789012")),
			wantExists: false,
		},
		{
			name: "service network vpc association",
			resource: newTerraformResourceResult(newVPCLatticeServiceNetworkVpcAssociationResource(vpclatticetypes.ServiceNetworkVpcAssociationSummary{
				Id:               aws.String("snva-123"),
				CreatedBy:        aws.String("123456789012"),
				ServiceNetworkId: aws.String("sn-123"),
				VpcId:            aws.String("vpc-123"),
				Status:           vpclatticetypes.ServiceNetworkVpcAssociationStatusActive,
			}, "123456789012")),
			wantID:     "snva-123",
			wantType:   vpclatticeServiceNetworkVpcAssociationResourceType,
			wantAttr:   map[string]string{"service_network_identifier": "sn-123", "vpc_identifier": "vpc-123"},
			wantExists: true,
		},
		{
			name: "service network vpc association from another account",
			resource: newTerraformResourceResult(newVPCLatticeServiceNetworkVpcAssociationResource(vpclatticetypes.ServiceNetworkVpcAssociationSummary{
				Id:               aws.String("snva-123"),
				CreatedBy:        aws.String("210987654321"),
				ServiceNetworkId: aws.String("sn-123"),
				VpcId:            aws.String("vpc-123"),
				Status:           vpclatticetypes.ServiceNetworkVpcAssociationStatusActive,
			}, "123456789012")),
			wantExists: false,
		},
		{
			name:       "auth policy",
			resource:   newTerraformResourceResult(newVPCLatticeAuthPolicyResource("arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123", `{"Version":"2012-10-17"}`)),
			wantID:     "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123",
			wantType:   vpclatticeAuthPolicyResourceType,
			wantAttr:   map[string]string{"resource_identifier": "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123", "policy": `{"Version":"2012-10-17"}`},
			wantExists: true,
		},
		{
			name:       "auth policy empty policy",
			resource:   newTerraformResourceResult(newVPCLatticeAuthPolicyResource("svc-123", "")),
			wantExists: false,
		},
		{
			name:       "resource policy",
			resource:   newTerraformResourceResult(newVPCLatticeResourcePolicyResource("arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123", `{"Version":"2012-10-17"}`)),
			wantID:     "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123",
			wantType:   vpclatticeResourcePolicyResourceType,
			wantAttr:   map[string]string{"resource_arn": "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123", "policy": `{"Version":"2012-10-17"}`},
			wantExists: true,
		},
		{
			name: "access log subscription",
			resource: newTerraformResourceResult(newVPCLatticeAccessLogSubscriptionResource(vpclatticetypes.AccessLogSubscriptionSummary{
				Id:             aws.String("als-123"),
				ResourceId:     aws.String("sn-123"),
				DestinationArn: aws.String("arn:aws:s3:::logs"),
			})),
			wantID:     "als-123",
			wantType:   vpclatticeAccessLogSubscriptionResourceType,
			wantAttr:   map[string]string{"resource_identifier": "sn-123", "destination_arn": "arn:aws:s3:::logs"},
			wantExists: true,
		},
		{
			name: "access log subscription missing destination",
			resource: newTerraformResourceResult(newVPCLatticeAccessLogSubscriptionResource(vpclatticetypes.AccessLogSubscriptionSummary{
				Id:         aws.String("als-123"),
				ResourceId: aws.String("sn-123"),
			})),
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resource.ok != tt.wantExists {
				t.Fatalf("resource exists = %t, want %t", tt.resource.ok, tt.wantExists)
			}
			if !tt.wantExists {
				return
			}
			resource := tt.resource.resource
			if resource.InstanceState.ID != tt.wantID {
				t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, tt.wantID)
			}
			if resource.InstanceInfo.Type != tt.wantType {
				t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, tt.wantType)
			}
			for key, want := range tt.wantAttr {
				if got := resource.InstanceState.Attributes[key]; got != want {
					t.Fatalf("attribute %s = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestVPCLatticeImportIDs(t *testing.T) {
	if got, want := vpclatticeListenerImportID("svc-123", "listener-123"), "svc-123/listener-123"; got != want {
		t.Fatalf("vpclatticeListenerImportID() = %q, want %q", got, want)
	}
	if got, want := vpclatticeListenerRuleImportID("svc-123", "listener-123", "rule-123"), "svc-123/listener-123/rule-123"; got != want {
		t.Fatalf("vpclatticeListenerRuleImportID() = %q, want %q", got, want)
	}
}

func TestVPCLatticeResourceNameWithLengthsAvoidsSanitizedCollisions(t *testing.T) {
	first := terraformutils.TfSanitize(vpclatticeResourceName("resource_policy", "ab", "c"))
	second := terraformutils.TfSanitize(vpclatticeResourceName("resource_policy", "a", "bc"))
	if first == second {
		t.Fatalf("vpclatticeResourceName() generated duplicate sanitized names %q", first)
	}
}

func TestVPCLatticeResourceOwnedByAccount(t *testing.T) {
	tests := []struct {
		name      string
		arn       string
		accountID string
		want      bool
	}{
		{
			name:      "owned service network",
			arn:       "arn:aws:vpc-lattice:us-east-1:123456789012:servicenetwork/sn-123",
			accountID: "123456789012",
			want:      true,
		},
		{
			name:      "shared service network",
			arn:       "arn:aws:vpc-lattice:us-east-1:210987654321:servicenetwork/sn-123",
			accountID: "123456789012",
			want:      false,
		},
		{
			name:      "owned service",
			arn:       "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123",
			accountID: "123456789012",
			want:      true,
		},
		{
			name:      "empty account",
			arn:       "arn:aws:vpc-lattice:us-east-1:123456789012:service/svc-123",
			accountID: "",
			want:      false,
		},
		{
			name:      "invalid arn",
			arn:       "svc-123",
			accountID: "123456789012",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vpclatticeResourceOwnedByAccount(tt.arn, tt.accountID); got != tt.want {
				t.Fatalf("vpclatticeResourceOwnedByAccount() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestVPCLatticeOptionalResourceUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "typed resource not found", err: &vpclatticetypes.ResourceNotFoundException{}, want: true},
		{name: "typed access denied", err: &vpclatticetypes.AccessDeniedException{}, want: true},
		{name: "typed validation", err: &vpclatticetypes.ValidationException{}, want: true},
		{name: "api error resource not found", err: &smithy.GenericAPIError{Code: "ResourceNotFoundException"}, want: true},
		{name: "api error throttling", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "wrapped resource not found", err: errors.Join(errors.New("lookup failed"), &vpclatticetypes.ResourceNotFoundException{}), want: true},
		{name: "nil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vpclatticeOptionalResourceUnavailable(tt.err); got != tt.want {
				t.Fatalf("vpclatticeOptionalResourceUnavailable() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestVPCLatticePostConvertHookWrapsPolicies(t *testing.T) {
	resource, ok := newVPCLatticeAuthPolicyResource("svc-123", "{}")
	if !ok {
		t.Fatal("expected auth policy resource")
	}
	resource.Item = map[string]interface{}{
		"policy": `{"Principal":"${aws:PrincipalAccount}"}`,
	}
	generator := &VPCLatticeGenerator{}
	generator.Resources = []terraformutils.Resource{resource}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook returned error: %v", err)
	}
	policy, ok := generator.Resources[0].Item["policy"].(string)
	if !ok {
		t.Fatalf("policy item = %#v, want string", generator.Resources[0].Item["policy"])
	}
	if !strings.HasPrefix(policy, "<<POLICY\n") || !strings.HasSuffix(policy, "\nPOLICY") {
		t.Fatalf("policy was not wrapped in heredoc: %q", policy)
	}
	if !strings.Contains(policy, "$${aws:PrincipalAccount}") {
		t.Fatalf("policy interpolation was not escaped: %q", policy)
	}
}

func TestVPCLatticeAssociationLoadersUseServiceNetworkFilters(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		switch r.URL.Path {
		case "/servicenetworkserviceassociations":
			if got := r.URL.Query().Get("serviceNetworkIdentifier"); got != "sn-123" {
				t.Errorf("serviceNetworkIdentifier query = %q, want sn-123", got)
			}
			writeVPCLatticeJSON(w, http.StatusOK, `{"items":[{"id":"snsa-123","createdBy":"123456789012","serviceId":"svc-123","serviceNetworkId":"sn-123","status":"ACTIVE"},{"id":"snsa-shared","createdBy":"210987654321","serviceId":"svc-shared","serviceNetworkId":"sn-123","status":"ACTIVE"}]}`)
		case "/servicenetworkvpcassociations":
			if got := r.URL.Query().Get("serviceNetworkIdentifier"); got != "sn-123" {
				t.Errorf("serviceNetworkIdentifier query = %q, want sn-123", got)
			}
			writeVPCLatticeJSON(w, http.StatusOK, `{"items":[{"id":"snva-123","createdBy":"123456789012","serviceNetworkId":"sn-123","vpcId":"vpc-123","status":"ACTIVE"},{"id":"snva-shared","createdBy":"210987654321","serviceNetworkId":"sn-123","vpcId":"vpc-shared","status":"ACTIVE"}]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	generator := &VPCLatticeGenerator{}
	client := newTestVPCLatticeClient(server)
	if err := generator.loadVPCLatticeServiceNetworkServiceAssociations(client, "", "123456789012"); err != nil {
		t.Fatalf("loadVPCLatticeServiceNetworkServiceAssociations empty ID returned error: %v", err)
	}
	if err := generator.loadVPCLatticeServiceNetworkVpcAssociations(client, "", "123456789012"); err != nil {
		t.Fatalf("loadVPCLatticeServiceNetworkVpcAssociations empty ID returned error: %v", err)
	}
	if len(generator.Resources) != 0 {
		t.Fatalf("len(Resources) after empty IDs = %d, want 0", len(generator.Resources))
	}

	if err := generator.loadVPCLatticeServiceNetworkServiceAssociations(client, "sn-123", "123456789012"); err != nil {
		t.Fatalf("loadVPCLatticeServiceNetworkServiceAssociations returned error: %v", err)
	}
	if err := generator.loadVPCLatticeServiceNetworkVpcAssociations(client, "sn-123", "123456789012"); err != nil {
		t.Fatalf("loadVPCLatticeServiceNetworkVpcAssociations returned error: %v", err)
	}
	if got := requests["/servicenetworkserviceassociations"]; got != 1 {
		t.Fatalf("service association requests = %d, want 1", got)
	}
	if got := requests["/servicenetworkvpcassociations"]; got != 1 {
		t.Fatalf("vpc association requests = %d, want 1", got)
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("len(Resources) = %d, want 2", len(generator.Resources))
	}
}

func newTestVPCLatticeClient(server *httptest.Server) *vpclattice.Client {
	config := aws.Config{
		Region:           "us-east-1",
		Credentials:      credentials.NewStaticCredentialsProvider("test", "test", ""),
		HTTPClient:       server.Client(),
		RetryMaxAttempts: 1,
	}
	return vpclattice.NewFromConfig(config, func(options *vpclattice.Options) {
		options.BaseEndpoint = aws.String(server.URL)
	})
}

func writeVPCLatticeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
