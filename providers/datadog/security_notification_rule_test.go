// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestSecurityNotificationRulesFromTypedResponses(t *testing.T) {
	tests := []struct {
		name         string
		response     datadogV2.NotificationRulesListResponse
		wantIDs      []string
		wantErrParts []string
	}{
		{
			name:     "one rule",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{securityNotificationRule("signal-rule")}},
			wantIDs:  []string{"signal-rule"},
		},
		{
			name: "two rules",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("signal-rule"),
				securityNotificationRule("vulnerability-rule"),
			}},
			wantIDs: []string{"signal-rule", "vulnerability-rule"},
		},
		{
			name:     "empty data",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{}},
			wantIDs:  []string{},
		},
		{
			name:         "nil data",
			response:     datadogV2.NotificationRulesListResponse{},
			wantErrParts: []string{"data", "nil"},
		},
		{
			name: "outer unparsed object",
			response: datadogV2.NotificationRulesListResponse{UnparsedObject: map[string]interface{}{
				"data": []interface{}{securityNotificationRuleRaw("raw-rule")},
			}},
			wantIDs: []string{"raw-rule"},
		},
		{
			name: "child unparsed object with valid identity",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{{
				UnparsedObject: securityNotificationRuleRaw("raw-child"),
			}}},
			wantIDs: []string{"raw-child"},
		},
		{
			name: "valid item followed by missing id",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{Type: datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES},
			}},
			wantErrParts: []string{"data[1]", "missing id"},
		},
		{
			name: "valid item followed by missing type",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{Id: "missing-type"},
			}},
			wantErrParts: []string{"data[1]", "missing type"},
		},
		{
			name: "valid item followed by wrong type",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{Id: "wrong-type", Type: datadogV2.NotificationRulesType("other_type")},
			}},
			wantErrParts: []string{"data[1]", "unexpected", "type"},
		},
		{
			name: "child raw object missing id at index zero",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{{
				UnparsedObject: map[string]interface{}{"type": "notification_rules"},
			}}},
			wantErrParts: []string{"data[0]", "missing id"},
		},
		{
			name: "child raw object has wrong id type",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{UnparsedObject: map[string]interface{}{"id": 42, "type": "notification_rules"}},
			}},
			wantErrParts: []string{"data[1]", "id"},
		},
		{
			name: "child raw object missing type",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{UnparsedObject: map[string]interface{}{"id": "missing-type"}},
			}},
			wantErrParts: []string{"data[1]", "missing type"},
		},
		{
			name: "child raw object has wrong type",
			response: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
				securityNotificationRule("valid-rule"),
				{UnparsedObject: map[string]interface{}{"id": "wrong-type", "type": "other_type"}},
			}},
			wantErrParts: []string{"data[1]", "unexpected", "type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := fmt.Sprintf("%#v", tt.response)
			rules, err := securityNotificationRulesFromResponse(tt.response)
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("securityNotificationRulesFromResponse returned error: %v", err)
				}
				if got := notificationRuleIDs(rules); !reflect.DeepEqual(got, tt.wantIDs) {
					t.Fatalf("notification rule IDs = %v, want %v", got, tt.wantIDs)
				}
			} else {
				if err == nil {
					t.Fatalf("securityNotificationRulesFromResponse returned rules %v, want error", notificationRuleIDs(rules))
				}
				if rules != nil {
					t.Fatalf("securityNotificationRulesFromResponse returned partial rules %v with error %v", notificationRuleIDs(rules), err)
				}
				assertErrorContains(t, err, tt.wantErrParts...)
			}
			if after := fmt.Sprintf("%#v", tt.response); after != before {
				t.Fatalf("input response mutated:\nbefore: %s\nafter:  %s", before, after)
			}
		})
	}
}

func TestSecurityNotificationRulesFromTypedResponsesRejectMalformedEntries(t *testing.T) {
	malformedResponse := securityNotificationRuleListResponseWithType(t, "malformed-rule", "other_type")
	malformedRule := malformedResponse.GetData()[0]
	if malformedRule.UnparsedObject == nil {
		t.Fatal("SDK-decoded malformed rule has nil UnparsedObject, want raw item data")
	}

	response := datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{
		securityNotificationRule("valid-rule"),
		malformedRule,
	}}
	rules, err := securityNotificationRulesFromResponse(response)
	if err == nil {
		t.Fatalf("securityNotificationRulesFromResponse returned rules %v, want error", notificationRuleIDs(rules))
	}
	if rules != nil {
		t.Fatalf("securityNotificationRulesFromResponse returned partial rules %v with error %v", notificationRuleIDs(rules), err)
	}
	assertErrorContains(t, err, "data[1]", "type")
}

func TestSecurityNotificationRuleSDKListDecoding(t *testing.T) {
	t.Run("explicit empty data list", func(t *testing.T) {
		var response datadogV2.NotificationRulesListResponse
		if err := json.Unmarshal([]byte(`{"data":[]}`), &response); err != nil {
			t.Fatalf("decode empty notification rule list through SDK model: %v", err)
		}
		if response.Data == nil {
			t.Fatal("SDK-decoded explicit empty data list is nil")
		}
		rules, err := securityNotificationRulesFromResponse(response)
		if err != nil {
			t.Fatalf("securityNotificationRulesFromResponse returned error: %v", err)
		}
		if len(rules) != 0 {
			t.Fatalf("notification rule count = %d, want 0", len(rules))
		}
	})

	t.Run("null data list", func(t *testing.T) {
		var response datadogV2.NotificationRulesListResponse
		if err := json.Unmarshal([]byte(`{"data":null}`), &response); err != nil {
			t.Fatalf("decode null notification rule list through SDK model: %v", err)
		}
		if response.Data != nil {
			t.Fatalf("SDK-decoded null data list = %#v, want nil", response.Data)
		}
		if _, err := securityNotificationRulesFromResponse(response); err == nil {
			t.Fatal("securityNotificationRulesFromResponse returned nil error, want nil data error")
		} else {
			assertErrorContains(t, err, "data", "nil")
		}
	})

	t.Run("valid typed item", func(t *testing.T) {
		response := securityNotificationRuleListResponseWithMutation(t, "valid-rule", nil)
		if response.UnparsedObject != nil {
			t.Fatal("SDK-decoded valid list has non-nil outer UnparsedObject")
		}
		if len(response.Data) != 1 {
			t.Fatalf("SDK-decoded valid list length = %d, want 1", len(response.Data))
		}
		if response.Data[0].UnparsedObject != nil {
			t.Fatal("SDK-decoded valid rule has non-nil child UnparsedObject")
		}

		rules, err := securityNotificationRulesFromResponse(response)
		if err != nil {
			t.Fatalf("securityNotificationRulesFromResponse returned error: %v", err)
		}
		if got := notificationRuleIDs(rules); !reflect.DeepEqual(got, []string{"valid-rule"}) {
			t.Fatalf("notification rule IDs = %v, want [valid-rule]", got)
		}
	})

	t.Run("unsupported nested selector is recovered from child raw data", func(t *testing.T) {
		response := securityNotificationRuleListResponseWithMutation(t, "forward-compatible-rule", func(rawRule map[string]interface{}) {
			attributes := rawRule["attributes"].(map[string]interface{})
			selectors := attributes["selectors"].(map[string]interface{})
			selectors["trigger_source"] = "future_security_source"
		})
		if response.UnparsedObject != nil {
			t.Fatal("SDK-decoded forward-incompatible list has non-nil outer UnparsedObject")
		}
		if len(response.Data) != 1 {
			t.Fatalf("SDK-decoded forward-incompatible list length = %d, want 1", len(response.Data))
		}
		if response.Data[0].UnparsedObject == nil {
			t.Fatal("SDK-decoded forward-incompatible rule has nil child UnparsedObject")
		}

		rules, err := securityNotificationRulesFromResponse(response)
		if err != nil {
			t.Fatalf("securityNotificationRulesFromResponse returned error: %v", err)
		}
		if got := notificationRuleIDs(rules); !reflect.DeepEqual(got, []string{"forward-compatible-rule"}) {
			t.Fatalf("notification rule IDs = %v, want [forward-compatible-rule]", got)
		}

		generator := &SecurityNotificationRuleGenerator{}
		resource, err := generator.createResource(rules[0])
		if err != nil {
			t.Fatalf("createResource returned error: %v", err)
		}
		if resource.InstanceState.ID != "forward-compatible-rule" || resource.InstanceInfo.Type != "datadog_security_notification_rule" {
			t.Fatalf("resource identity = (%q, %q), want forward-compatible-rule and datadog_security_notification_rule", resource.InstanceState.ID, resource.InstanceInfo.Type)
		}
	})
}

func TestSecurityNotificationRulesFromRawResponseShapes(t *testing.T) {
	tests := []struct {
		name         string
		raw          interface{}
		wantIDs      []string
		wantErrParts []string
	}{
		{
			name:    "one valid rule",
			raw:     map[string]interface{}{"data": []interface{}{securityNotificationRuleRaw("signal-rule")}},
			wantIDs: []string{"signal-rule"},
		},
		{
			name: "signal and vulnerability rules",
			raw: map[string]interface{}{"data": []interface{}{
				securityNotificationRuleRaw("signal-rule"),
				securityNotificationRuleRaw("vulnerability-rule"),
			}},
			wantIDs: []string{"signal-rule", "vulnerability-rule"},
		},
		{name: "empty data list", raw: map[string]interface{}{"data": []interface{}{}}, wantIDs: []string{}},
		{name: "missing data", raw: map[string]interface{}{}, wantErrParts: []string{"missing data"}},
		{name: "null data", raw: map[string]interface{}{"data": nil}, wantErrParts: []string{"data", "null"}},
		{name: "object data", raw: map[string]interface{}{"data": map[string]interface{}{}}, wantErrParts: []string{"data", "list"}},
		{name: "string data", raw: map[string]interface{}{"data": "rules"}, wantErrParts: []string{"data", "list"}},
		{name: "non-object entry", raw: map[string]interface{}{"data": []interface{}{"rule"}}, wantErrParts: []string{"data[0]", "not an object"}},
		{name: "missing id", raw: map[string]interface{}{"data": []interface{}{map[string]interface{}{"type": "notification_rules"}}}, wantErrParts: []string{"data[0]", "missing id"}},
		{name: "empty id", raw: map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": "", "type": "notification_rules"}}}, wantErrParts: []string{"data[0]", "id"}},
		{name: "numeric id", raw: map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": 42, "type": "notification_rules"}}}, wantErrParts: []string{"data[0]", "id"}},
		{name: "wrong type", raw: map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": "rule", "type": "other_type"}}}, wantErrParts: []string{"data[0]", "type"}},
		{name: "missing type", raw: map[string]interface{}{"data": []interface{}{map[string]interface{}{"id": "rule"}}}, wantErrParts: []string{"data[0]", "missing type"}},
		{
			name: "valid entry followed by malformed entry",
			raw: map[string]interface{}{"data": []interface{}{
				securityNotificationRuleRaw("valid"),
				map[string]interface{}{"id": "invalid", "type": "other_type"},
			}},
			wantErrParts: []string{"data[1]", "type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := securityNotificationRulesFromRawData(tt.raw)
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("securityNotificationRulesFromRawData returned error: %v", err)
				}
				if got := notificationRuleIDs(rules); !reflect.DeepEqual(got, tt.wantIDs) {
					t.Fatalf("notification rule IDs = %v, want %v", got, tt.wantIDs)
				}
				return
			}
			if err == nil {
				t.Fatalf("securityNotificationRulesFromRawData returned rules %v, want error", notificationRuleIDs(rules))
			}
			if rules != nil {
				t.Fatalf("securityNotificationRulesFromRawData returned partial rules %v with error %v", notificationRuleIDs(rules), err)
			}
			assertErrorContains(t, err, tt.wantErrParts...)
		})
	}
}

func TestSecurityNotificationRuleFromRawDataShapes(t *testing.T) {
	tests := []struct {
		name         string
		raw          interface{}
		wantID       string
		wantErrParts []string
	}{
		{name: "valid signal rule", raw: securityNotificationRuleRaw("signal-rule"), wantID: "signal-rule"},
		{name: "valid vulnerability rule", raw: securityNotificationRuleRaw("vulnerability-rule"), wantID: "vulnerability-rule"},
		{name: "data is a list", raw: []interface{}{securityNotificationRuleRaw("rule")}, wantErrParts: []string{"not an object"}},
		{name: "missing id", raw: map[string]interface{}{"type": "notification_rules"}, wantErrParts: []string{"missing id"}},
		{name: "empty id", raw: map[string]interface{}{"id": "", "type": "notification_rules"}, wantErrParts: []string{"id"}},
		{name: "wrong id type", raw: map[string]interface{}{"id": true, "type": "notification_rules"}, wantErrParts: []string{"id"}},
		{name: "wrong resource type", raw: map[string]interface{}{"id": "rule", "type": "other_type"}, wantErrParts: []string{"type"}},
		{name: "missing resource type", raw: map[string]interface{}{"id": "rule"}, wantErrParts: []string{"missing type"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := securityNotificationRuleFromRawData(tt.raw)
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("securityNotificationRuleFromRawData returned error: %v", err)
				}
				if rule.GetId() != tt.wantID {
					t.Fatalf("notification rule ID = %q, want %q", rule.GetId(), tt.wantID)
				}
				if rule.GetType() != datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES {
					t.Fatalf("notification rule type = %q, want notification_rules", rule.GetType())
				}
				return
			}
			if err == nil {
				t.Fatalf("securityNotificationRuleFromRawData returned rule %q, want error", rule.GetId())
			}
			assertErrorContains(t, err, tt.wantErrParts...)
		})
	}
}

func TestSecurityNotificationRuleFromResponseShapes(t *testing.T) {
	tests := []struct {
		name         string
		response     datadogV2.NotificationRuleResponse
		wantID       string
		wantErrParts []string
	}{
		{name: "valid decoded rule", response: securityNotificationRuleResponse("decoded-rule"), wantID: "decoded-rule"},
		{
			name: "valid unparsed rule",
			response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{
				"data": securityNotificationRuleRaw("raw-rule"),
			}},
			wantID: "raw-rule",
		},
		{name: "decoded response missing data", response: datadogV2.NotificationRuleResponse{}, wantErrParts: []string{"decoded", "missing data"}},
		{name: "decoded rule missing id", response: securityNotificationRuleResponseWithRule(datadogV2.NotificationRule{Type: datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES}), wantErrParts: []string{"decoded", "missing id"}},
		{name: "decoded rule missing type", response: securityNotificationRuleResponseWithRule(datadogV2.NotificationRule{Id: "rule"}), wantErrParts: []string{"decoded", "missing type"}},
		{name: "decoded rule has wrong type", response: securityNotificationRuleResponseWithRule(datadogV2.NotificationRule{Id: "rule", Type: datadogV2.NotificationRulesType("other_type")}), wantErrParts: []string{"unexpected", "type"}},
		{name: "unparsed response missing data", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{}}, wantErrParts: []string{"raw", "missing data"}},
		{name: "unparsed data is a list", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": []interface{}{}}}, wantErrParts: []string{"not an object"}},
		{name: "unparsed rule missing id", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": map[string]interface{}{"type": "notification_rules"}}}, wantErrParts: []string{"missing id"}},
		{name: "unparsed rule has empty id", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": map[string]interface{}{"id": "", "type": "notification_rules"}}}, wantErrParts: []string{"id"}},
		{name: "unparsed rule has wrong id type", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": map[string]interface{}{"id": 42, "type": "notification_rules"}}}, wantErrParts: []string{"id"}},
		{name: "unparsed rule missing type", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": map[string]interface{}{"id": "rule"}}}, wantErrParts: []string{"missing type"}},
		{name: "unparsed rule has wrong type", response: datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{"data": map[string]interface{}{"id": "rule", "type": "other_type"}}}, wantErrParts: []string{"unexpected", "type"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := securityNotificationRuleFromResponse(tt.response)
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("securityNotificationRuleFromResponse returned error: %v", err)
				}
				if rule.GetId() != tt.wantID {
					t.Fatalf("notification rule ID = %q, want %q", rule.GetId(), tt.wantID)
				}
				return
			}
			if err == nil {
				t.Fatalf("securityNotificationRuleFromResponse returned rule %q, want error", rule.GetId())
			}
			assertErrorContains(t, err, tt.wantErrParts...)
		})
	}
}

func TestGetSecurityNotificationRuleFallbackPolicy(t *testing.T) {
	tests := []struct {
		name         string
		responses    map[string]notificationRuleHTTPFixture
		wantPaths    []string
		wantID       string
		wantErrParts []string
	}{
		{
			name: "decoded signal rule",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: securityNotificationRuleResponseJSON(t, "rule")},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules/rule"},
			wantID:    "rule",
		},
		{
			name: "unparsed signal rule",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: `{"data":{"id":"rule","type":"notification_rules"}}`},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules/rule"},
			wantID:    "rule",
		},
		{
			name: "signal response ID mismatch does not fall back",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: securityNotificationRuleResponseJSON(t, "other-rule")},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"signal", `"other-rule"`, `"rule"`, "does not match"},
		},
		{
			name: "signal 404 then decoded vulnerability rule",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/rule": {body: securityNotificationRuleResponseJSON(t, "rule")},
			},
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/rule",
				"/api/v2/security/vulnerabilities/notification_rules/rule",
			},
			wantID: "rule",
		},
		{
			name: "signal 404 then unparsed vulnerability rule",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/rule": {body: `{"data":{"id":"rule","type":"notification_rules"}}`},
			},
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/rule",
				"/api/v2/security/vulnerabilities/notification_rules/rule",
			},
			wantID: "rule",
		},
		{
			name: "vulnerability response ID mismatch",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/rule": {body: securityNotificationRuleResponseJSON(t, "other-rule")},
			},
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/rule",
				"/api/v2/security/vulnerabilities/notification_rules/rule",
			},
			wantErrParts: []string{"vulnerability", `"other-rule"`, `"rule"`, "does not match"},
		},
		{
			name: "both endpoints return 404",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/rule": {status: http.StatusNotFound},
			},
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/rule",
				"/api/v2/security/vulnerabilities/notification_rules/rule",
			},
			wantErrParts: []string{"vulnerability", "rule", "404"},
		},
		{
			name: "signal 403 does not fall back",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {status: http.StatusForbidden},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"signal", "rule", "403"},
		},
		{
			name: "signal 429 does not fall back",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {status: http.StatusTooManyRequests},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"signal", "rule", "429"},
		},
		{
			name: "signal 500 does not fall back",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {status: http.StatusInternalServerError},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"signal", "rule", "500"},
		},
		{
			name: "malformed signal success does not fall back",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: `{"data":{"id":"rule"}}`},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"parse", "signal", "missing type"},
		},
		{
			name: "malformed vulnerability fallback",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/rule": {body: `{"data":{"id":"rule"}}`},
			},
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/rule",
				"/api/v2/security/vulnerabilities/notification_rules/rule",
			},
			wantErrParts: []string{"parse", "vulnerability", "missing type"},
		},
		{
			name: "empty decoded data",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: `{"data":null}`},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"parse", "signal", "missing data"},
		},
		{
			name: "single response data is a list",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/rule": {body: `{"data":[]}`},
			},
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/rule"},
			wantErrParts: []string{"parse", "signal", "not an object"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, requestedPaths := newSecurityNotificationRuleServer(t, tt.responses)
			defer server.Close()

			api := datadogV2.NewSecurityMonitoringApi(newTeamRelationshipTestClient(server))
			rule, err := getSecurityNotificationRule(context.Background(), api, "rule")
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("getSecurityNotificationRule returned error: %v", err)
				}
				if rule.GetId() != tt.wantID {
					t.Fatalf("notification rule ID = %q, want %q", rule.GetId(), tt.wantID)
				}
			} else {
				if err == nil {
					t.Fatalf("getSecurityNotificationRule returned rule %q, want error", rule.GetId())
				}
				assertErrorContains(t, err, tt.wantErrParts...)
			}
			if !reflect.DeepEqual(*requestedPaths, tt.wantPaths) {
				t.Fatalf("request paths = %v, want %v", *requestedPaths, tt.wantPaths)
			}
		})
	}
}

func TestGetSecurityNotificationRuleDoesNotFallbackOnTransportOrContextErrors(t *testing.T) {
	t.Run("transport error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		api := datadogV2.NewSecurityMonitoringApi(newTeamRelationshipTestClient(server))
		server.Close()

		_, err := getSecurityNotificationRule(context.Background(), api, "rule")
		if err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want transport error")
		}
		assertErrorContains(t, err, "get signal notification rule", "rule")
	})

	t.Run("context cancellation", func(t *testing.T) {
		var requests int
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
		defer server.Close()
		api := datadogV2.NewSecurityMonitoringApi(newTeamRelationshipTestClient(server))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := getSecurityNotificationRule(ctx, api, "rule")
		if err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want context error")
		}
		assertErrorContains(t, err, "get signal notification rule", "rule")
		if requests != 0 {
			t.Fatalf("request count = %d, want 0", requests)
		}
	})
}

func TestSecurityNotificationRuleInitResourcesDiscovery(t *testing.T) {
	tests := []struct {
		name         string
		responses    map[string]notificationRuleHTTPFixture
		filter       []terraformutils.ResourceFilter
		initial      []terraformutils.Resource
		wantPaths    []string
		wantIDs      []string
		wantErrParts []string
	}{
		{
			name: "combines decoded endpoint lists",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSONTyped(t, "signal-rule")},
				"/api/v2/security/vulnerabilities/notification_rules": {body: securityNotificationRuleListResponseJSONTyped(t, "vulnerability-rule")},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantIDs:   []string{"signal-rule", "vulnerability-rule"},
		},
		{
			name: "empty signal and valid vulnerability",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSONTyped(t)},
				"/api/v2/security/vulnerabilities/notification_rules": {body: securityNotificationRuleListResponseJSON("vulnerability-rule")},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantIDs:   []string{"vulnerability-rule"},
		},
		{
			name: "valid signal and empty vulnerability",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSON("signal-rule")},
				"/api/v2/security/vulnerabilities/notification_rules": {body: securityNotificationRuleListResponseJSONTyped(t)},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantIDs:   []string{"signal-rule"},
		},
		{
			name: "both endpoints empty",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSONTyped(t)},
				"/api/v2/security/vulnerabilities/notification_rules": {body: securityNotificationRuleListResponseJSONTyped(t)},
			},
			wantPaths: []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantIDs:   []string{},
		},
		{
			name: "signal list failure clears stale resources",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules": {status: http.StatusInternalServerError},
			},
			initial:      staleSecurityNotificationRuleResources(),
			wantPaths:    []string{"/api/v2/security/signals/notification_rules"},
			wantErrParts: []string{"list signal notification rules", "500"},
		},
		{
			name: "vulnerability list failure publishes no signal resources",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSON("signal-rule")},
				"/api/v2/security/vulnerabilities/notification_rules": {status: http.StatusInternalServerError},
			},
			initial:      staleSecurityNotificationRuleResources(),
			wantPaths:    []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantErrParts: []string{"list vulnerability notification rules", "500"},
		},
		{
			name: "parse failure publishes no partial resources",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSON("signal-rule")},
				"/api/v2/security/vulnerabilities/notification_rules": {body: `{"data":[{"id":"invalid"}]}`},
			},
			initial:      staleSecurityNotificationRuleResources(),
			wantPaths:    []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantErrParts: []string{"parse vulnerability notification rules", "missing type"},
		},
		{
			name: "filtered decoded signal lookup",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/signal-rule": {body: securityNotificationRuleResponseJSON(t, "signal-rule")},
			},
			filter:    securityNotificationRuleIDFilter("signal-rule"),
			wantPaths: []string{"/api/v2/security/signals/notification_rules/signal-rule"},
			wantIDs:   []string{"signal-rule"},
		},
		{
			name: "filtered vulnerability lookup",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/vulnerability-rule":         {status: http.StatusNotFound},
				"/api/v2/security/vulnerabilities/notification_rules/vulnerability-rule": {body: securityNotificationRuleResponseJSON(t, "vulnerability-rule")},
			},
			filter: securityNotificationRuleIDFilter("vulnerability-rule"),
			wantPaths: []string{
				"/api/v2/security/signals/notification_rules/vulnerability-rule",
				"/api/v2/security/vulnerabilities/notification_rules/vulnerability-rule",
			},
			wantIDs: []string{"vulnerability-rule"},
		},
		{
			name: "unrelated typed filter uses list discovery",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules":         {body: securityNotificationRuleListResponseJSONTyped(t, "signal-rule")},
				"/api/v2/security/vulnerabilities/notification_rules": {body: securityNotificationRuleListResponseJSONTyped(t)},
			},
			filter: []terraformutils.ResourceFilter{{
				ServiceName:      "security_monitoring_rule",
				FieldPath:        "id",
				AcceptableValues: []string{"unrelated"},
			}},
			wantPaths: []string{"/api/v2/security/signals/notification_rules", "/api/v2/security/vulnerabilities/notification_rules"},
			wantIDs:   []string{"signal-rule"},
		},
		{
			name: "multiple filtered IDs preserve order",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/first":  {body: securityNotificationRuleResponseJSON(t, "first")},
				"/api/v2/security/signals/notification_rules/second": {body: securityNotificationRuleResponseJSON(t, "second")},
			},
			filter:    securityNotificationRuleIDFilter("first", "second"),
			wantPaths: []string{"/api/v2/security/signals/notification_rules/first", "/api/v2/security/signals/notification_rules/second"},
			wantIDs:   []string{"first", "second"},
		},
		{
			name: "later filtered failure publishes no earlier resource",
			responses: map[string]notificationRuleHTTPFixture{
				"/api/v2/security/signals/notification_rules/first":  {body: securityNotificationRuleResponseJSON(t, "first")},
				"/api/v2/security/signals/notification_rules/second": {status: http.StatusInternalServerError},
			},
			filter:       securityNotificationRuleIDFilter("first", "second"),
			initial:      staleSecurityNotificationRuleResources(),
			wantPaths:    []string{"/api/v2/security/signals/notification_rules/first", "/api/v2/security/signals/notification_rules/second"},
			wantErrParts: []string{"get signal notification rule", "second", "500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, requestedPaths := newSecurityNotificationRuleServer(t, tt.responses)
			defer server.Close()
			generator := newSecurityNotificationRuleTestGenerator(server, tt.filter)
			generator.Resources = append([]terraformutils.Resource(nil), tt.initial...)

			err := generator.InitResources()
			if len(tt.wantErrParts) == 0 {
				if err != nil {
					t.Fatalf("InitResources returned error: %v", err)
				}
				if got := resourceIDs(generator.Resources); !reflect.DeepEqual(got, tt.wantIDs) {
					t.Fatalf("resource IDs = %v, want %v", got, tt.wantIDs)
				}
			} else {
				if err == nil {
					t.Fatal("InitResources returned nil error, want error")
				}
				assertErrorContains(t, err, tt.wantErrParts...)
				if len(generator.Resources) != 0 {
					t.Fatalf("resources after failed initialization = %v, want empty", resourceIDs(generator.Resources))
				}
			}
			if !reflect.DeepEqual(*requestedPaths, tt.wantPaths) {
				t.Fatalf("request paths = %v, want %v", *requestedPaths, tt.wantPaths)
			}
		})
	}
}

func TestSecurityNotificationRuleTypedAndRawResourcesHaveEquivalentIdentity(t *testing.T) {
	typedRules, err := securityNotificationRulesFromResponse(datadogV2.NotificationRulesListResponse{
		Data: []datadogV2.NotificationRule{securityNotificationRule("rule")},
	})
	if err != nil {
		t.Fatalf("parse typed response: %v", err)
	}
	rawRules, err := securityNotificationRulesFromRawData(map[string]interface{}{
		"data": []interface{}{securityNotificationRuleRaw("rule")},
	})
	if err != nil {
		t.Fatalf("parse raw response: %v", err)
	}
	generator := &SecurityNotificationRuleGenerator{}
	typedResources, err := generator.createResources(typedRules)
	if err != nil {
		t.Fatalf("create typed resource: %v", err)
	}
	rawResources, err := generator.createResources(rawRules)
	if err != nil {
		t.Fatalf("create raw resource: %v", err)
	}
	if !reflect.DeepEqual(typedResources, rawResources) {
		t.Fatalf("typed and raw resources differ:\ntyped: %#v\nraw:   %#v", typedResources, rawResources)
	}
}

func TestGetSecurityNotificationRuleClosesResponseBodies(t *testing.T) {
	t.Run("signal success", func(t *testing.T) {
		signalBody := newTrackingBody(`{"data":{}}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return securityNotificationRuleResponse("rule"), responseWithBody(http.StatusOK, signalBody), nil
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err != nil {
			t.Fatalf("getSecurityNotificationRule returned error: %v", err)
		}
		assertBodyClosedOnce(t, signalBody)
	})

	t.Run("signal non-404 error", func(t *testing.T) {
		signalBody := newTrackingBody(`{"errors":["forbidden"]}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, responseWithBody(http.StatusForbidden, signalBody), errors.New("403 Forbidden")
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want error")
		}
		assertBodyClosedOnce(t, signalBody)
	})

	t.Run("signal parse error", func(t *testing.T) {
		signalBody := newTrackingBody(`{"data":{"id":"rule"}}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{
					"data": map[string]interface{}{"id": "rule"},
				}}, responseWithBody(http.StatusOK, signalBody), nil
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want parse error")
		}
		assertBodyClosedOnce(t, signalBody)
	})

	t.Run("signal 404 closes before vulnerability request", func(t *testing.T) {
		signalBody := newTrackingBody(`{"errors":["not found"]}`)
		vulnerabilityBody := newTrackingBody(`{"data":{}}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, responseWithBody(http.StatusNotFound, signalBody), errors.New("404 Not Found")
			},
			getVulnerability: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				if got := signalBody.closeCount.Load(); got != 1 {
					t.Fatalf("signal body close count before fallback = %d, want 1", got)
				}
				return securityNotificationRuleResponse("rule"), responseWithBody(http.StatusOK, vulnerabilityBody), nil
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err != nil {
			t.Fatalf("getSecurityNotificationRule returned error: %v", err)
		}
		assertBodyClosedOnce(t, signalBody)
		assertBodyClosedOnce(t, vulnerabilityBody)
	})

	t.Run("vulnerability parse error", func(t *testing.T) {
		signalBody := newTrackingBody(`{"errors":["not found"]}`)
		vulnerabilityBody := newTrackingBody(`{"data":{"id":"rule"}}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, responseWithBody(http.StatusNotFound, signalBody), errors.New("404 Not Found")
			},
			getVulnerability: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{UnparsedObject: map[string]interface{}{
					"data": map[string]interface{}{"id": "rule"},
				}}, responseWithBody(http.StatusOK, vulnerabilityBody), nil
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want parse error")
		}
		assertBodyClosedOnce(t, signalBody)
		assertBodyClosedOnce(t, vulnerabilityBody)
	})

	t.Run("vulnerability failure", func(t *testing.T) {
		signalBody := newTrackingBody(`{"errors":["not found"]}`)
		vulnerabilityBody := newTrackingBody(`{"errors":["failed"]}`)
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, responseWithBody(http.StatusNotFound, signalBody), errors.New("404 Not Found")
			},
			getVulnerability: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, responseWithBody(http.StatusInternalServerError, vulnerabilityBody), errors.New("500 Internal Server Error")
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want error")
		}
		assertBodyClosedOnce(t, signalBody)
		assertBodyClosedOnce(t, vulnerabilityBody)
	})

	t.Run("nil response", func(t *testing.T) {
		api := &fakeSecurityNotificationRuleAPI{
			getSignal: func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
				return datadogV2.NotificationRuleResponse{}, nil, errors.New("transport error")
			},
		}

		if _, err := getSecurityNotificationRule(context.Background(), api, "rule"); err == nil {
			t.Fatal("getSecurityNotificationRule returned nil error, want error")
		}
	})
}

func TestListSecurityNotificationRulesClosesResponseBodies(t *testing.T) {
	tests := []struct {
		name                  string
		signalResponse        datadogV2.NotificationRulesListResponse
		signalStatus          int
		signalErr             error
		vulnerabilityResponse datadogV2.NotificationRulesListResponse
		vulnerabilityStatus   int
		vulnerabilityErr      error
		wantVulnerabilityCall bool
		wantErr               bool
	}{
		{
			name:                  "both lists succeed",
			signalResponse:        datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{securityNotificationRule("signal")}},
			vulnerabilityResponse: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{securityNotificationRule("vulnerability")}},
			wantVulnerabilityCall: true,
		},
		{
			name:         "signal list error",
			signalStatus: http.StatusForbidden,
			signalErr:    errors.New("403 Forbidden"),
			wantErr:      true,
		},
		{
			name:                  "vulnerability list error",
			signalResponse:        datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{}},
			vulnerabilityStatus:   http.StatusInternalServerError,
			vulnerabilityErr:      errors.New("500 Internal Server Error"),
			wantVulnerabilityCall: true,
			wantErr:               true,
		},
		{
			name: "signal parse error",
			signalResponse: datadogV2.NotificationRulesListResponse{UnparsedObject: map[string]interface{}{
				"data": []interface{}{map[string]interface{}{"id": "missing-type"}},
			}},
			wantErr: true,
		},
		{
			name:           "vulnerability parse error",
			signalResponse: datadogV2.NotificationRulesListResponse{Data: []datadogV2.NotificationRule{}},
			vulnerabilityResponse: datadogV2.NotificationRulesListResponse{UnparsedObject: map[string]interface{}{
				"data": []interface{}{map[string]interface{}{"id": "missing-type"}},
			}},
			wantVulnerabilityCall: true,
			wantErr:               true,
		},
		{
			name:         "nil signal response on error",
			signalStatus: -1,
			signalErr:    errors.New("transport error"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var signalBody, vulnerabilityBody *trackingBody
			var vulnerabilityCalled bool
			api := &fakeSecurityNotificationRuleAPI{
				listSignal: func(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error) {
					if tt.signalStatus == -1 {
						return tt.signalResponse, nil, tt.signalErr
					}
					signalBody = newTrackingBody(`{"data":[]}`)
					return tt.signalResponse, responseWithBody(defaultHTTPStatus(tt.signalStatus), signalBody), tt.signalErr
				},
				listVulnerability: func(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error) {
					vulnerabilityCalled = true
					vulnerabilityBody = newTrackingBody(`{"data":[]}`)
					return tt.vulnerabilityResponse, responseWithBody(defaultHTTPStatus(tt.vulnerabilityStatus), vulnerabilityBody), tt.vulnerabilityErr
				},
			}

			_, err := listSecurityNotificationRules(context.Background(), api)
			if tt.wantErr && err == nil {
				t.Fatal("listSecurityNotificationRules returned nil error, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("listSecurityNotificationRules returned error: %v", err)
			}
			if signalBody != nil {
				assertBodyClosedOnce(t, signalBody)
			}
			if vulnerabilityCalled != tt.wantVulnerabilityCall {
				t.Fatalf("vulnerability endpoint called = %t, want %t", vulnerabilityCalled, tt.wantVulnerabilityCall)
			}
			if vulnerabilityBody != nil {
				assertBodyClosedOnce(t, vulnerabilityBody)
			}
		})
	}
}

type fakeSecurityNotificationRuleAPI struct {
	getSignal         func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error)
	getVulnerability  func(context.Context, string) (datadogV2.NotificationRuleResponse, *http.Response, error)
	listSignal        func(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error)
	listVulnerability func(context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error)
}

func (f *fakeSecurityNotificationRuleAPI) GetSignalNotificationRule(ctx context.Context, id string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
	return f.getSignal(ctx, id)
}

func (f *fakeSecurityNotificationRuleAPI) GetVulnerabilityNotificationRule(ctx context.Context, id string) (datadogV2.NotificationRuleResponse, *http.Response, error) {
	return f.getVulnerability(ctx, id)
}

func (f *fakeSecurityNotificationRuleAPI) GetSignalNotificationRules(ctx context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error) {
	return f.listSignal(ctx)
}

func (f *fakeSecurityNotificationRuleAPI) GetVulnerabilityNotificationRules(ctx context.Context) (datadogV2.NotificationRulesListResponse, *http.Response, error) {
	return f.listVulnerability(ctx)
}

type trackingBody struct {
	io.Reader
	closeCount atomic.Int32
}

func newTrackingBody(body string) *trackingBody {
	return &trackingBody{Reader: strings.NewReader(body)}
}

func (b *trackingBody) Close() error {
	b.closeCount.Add(1)
	return nil
}

func responseWithBody(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Body: body}
}

func defaultHTTPStatus(status int) int {
	if status == 0 {
		return http.StatusOK
	}
	return status
}

func assertBodyClosedOnce(t *testing.T, body *trackingBody) {
	t.Helper()
	if got := body.closeCount.Load(); got != 1 {
		t.Fatalf("response body close count = %d, want 1", got)
	}
}

func securityNotificationRuleResponse(id string) datadogV2.NotificationRuleResponse {
	response := datadogV2.NotificationRuleResponse{}
	response.SetData(securityNotificationRule(id))
	return response
}

func securityNotificationRuleResponseWithRule(rule datadogV2.NotificationRule) datadogV2.NotificationRuleResponse {
	response := datadogV2.NotificationRuleResponse{}
	response.SetData(rule)
	return response
}

type notificationRuleHTTPFixture struct {
	status int
	body   string
}

func newSecurityNotificationRuleServer(t *testing.T, fixtures map[string]notificationRuleHTTPFixture) (*httptest.Server, *[]string) {
	t.Helper()
	requestedPaths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		fixture, ok := fixtures[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		status := fixture.status
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if fixture.body != "" {
			_, _ = fmt.Fprint(w, fixture.body)
			return
		}
		if status >= http.StatusBadRequest {
			_, _ = fmt.Fprint(w, `{"errors":["request failed"]}`)
		}
	}))
	return server, &requestedPaths
}

func securityNotificationRule(id string) datadogV2.NotificationRule {
	selectors := datadogV2.NewSelectors(datadogV2.TRIGGERSOURCE_SECURITY_SIGNALS)
	attributes := datadogV2.NewNotificationRuleAttributes(
		0,
		*datadogV2.NewRuleUser(),
		true,
		0,
		*datadogV2.NewRuleUser(),
		"notification rule",
		*selectors,
		[]string{},
		1,
	)
	return *datadogV2.NewNotificationRule(*attributes, id, datadogV2.NOTIFICATIONRULESTYPE_NOTIFICATION_RULES)
}

func securityNotificationRuleRaw(id string) map[string]interface{} {
	return map[string]interface{}{
		"id":   id,
		"type": "notification_rules",
	}
}

func securityNotificationRuleResponseJSON(t *testing.T, id string) string {
	t.Helper()
	response := datadogV2.NotificationRuleResponse{}
	response.SetData(securityNotificationRule(id))
	return marshalSecurityNotificationRuleJSON(t, response)
}

func securityNotificationRuleListResponseJSONTyped(t *testing.T, ids ...string) string {
	t.Helper()
	rules := make([]datadogV2.NotificationRule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, securityNotificationRule(id))
	}
	response := datadogV2.NotificationRulesListResponse{Data: rules}
	return marshalSecurityNotificationRuleJSON(t, response)
}

func securityNotificationRuleListResponseWithType(t *testing.T, id string, ruleType string) datadogV2.NotificationRulesListResponse {
	return securityNotificationRuleListResponseWithMutation(t, id, func(rawRule map[string]interface{}) {
		rawRule["type"] = ruleType
	})
}

func securityNotificationRuleListResponseWithMutation(t *testing.T, id string, mutate func(map[string]interface{})) datadogV2.NotificationRulesListResponse {
	t.Helper()

	var rawResponse map[string]interface{}
	if err := json.Unmarshal([]byte(securityNotificationRuleListResponseJSONTyped(t, id)), &rawResponse); err != nil {
		t.Fatalf("unmarshal notification rule list fixture: %v", err)
	}
	rawRules := rawResponse["data"].([]interface{})
	if mutate != nil {
		mutate(rawRules[0].(map[string]interface{}))
	}

	payload, err := json.Marshal(rawResponse)
	if err != nil {
		t.Fatalf("marshal notification rule list fixture: %v", err)
	}
	var response datadogV2.NotificationRulesListResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("decode notification rule list fixture through SDK model: %v", err)
	}
	if response.UnparsedObject != nil {
		t.Fatal("SDK-decoded notification rule list has non-nil outer UnparsedObject")
	}
	return response
}

func marshalSecurityNotificationRuleJSON(t *testing.T, value interface{}) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal notification rule response: %v", err)
	}
	return string(data)
}

func notificationRuleIDs(rules []datadogV2.NotificationRule) []string {
	ids := make([]string, 0, len(rules))
	for _, rule := range rules {
		ids = append(ids, rule.GetId())
	}
	return ids
}

func resourceIDs(resources []terraformutils.Resource) []string {
	ids := make([]string, 0, len(resources))
	for _, resource := range resources {
		ids = append(ids, resource.InstanceState.ID)
	}
	return ids
}

func securityNotificationRuleIDFilter(ids ...string) []terraformutils.ResourceFilter {
	return []terraformutils.ResourceFilter{{
		ServiceName:      "security_notification_rule",
		FieldPath:        "id",
		AcceptableValues: ids,
	}}
}

func staleSecurityNotificationRuleResources() []terraformutils.Resource {
	return []terraformutils.Resource{terraformutils.NewSimpleResource(
		"stale-rule",
		"security_notification_rule_stale-rule",
		"datadog_security_notification_rule",
		"datadog",
		SecurityNotificationRuleAllowEmptyValues,
	)}
}

func assertErrorContains(t *testing.T, err error, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error %q does not contain %q", err, part)
		}
	}
}
