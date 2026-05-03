// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestOnCallCreateResources(t *testing.T) {
	policyResource, err := (&OnCallEscalationPolicyGenerator{}).createResource(onCallEscalationPolicy(t, "policy-1"))
	if err != nil {
		t.Fatalf("createResource escalation policy returned error: %v", err)
	}
	if policyResource.InstanceState.ID != "policy-1" {
		t.Fatalf("policy resource ID = %q, want policy-1", policyResource.InstanceState.ID)
	}
	if policyResource.InstanceInfo.Type != "datadog_on_call_escalation_policy" {
		t.Fatalf("policy resource type = %q, want datadog_on_call_escalation_policy", policyResource.InstanceInfo.Type)
	}

	scheduleResource, err := (&OnCallScheduleGenerator{}).createResource(onCallSchedule(t, "schedule-1"))
	if err != nil {
		t.Fatalf("createResource schedule returned error: %v", err)
	}
	if scheduleResource.InstanceState.ID != "schedule-1" {
		t.Fatalf("schedule resource ID = %q, want schedule-1", scheduleResource.InstanceState.ID)
	}

	routingResource, err := (&OnCallTeamRoutingRulesGenerator{}).createResource(onCallTeamRoutingRules(t, "team-1"))
	if err != nil {
		t.Fatalf("createResource routing rules returned error: %v", err)
	}
	if routingResource.InstanceState.ID != "team-1" {
		t.Fatalf("routing rules resource ID = %q, want team-1", routingResource.InstanceState.ID)
	}

	channelResource, skipped, err := (&OnCallUserNotificationChannelGenerator{}).createResource("user-1", onCallNotificationChannel(t, onCallEmailNotificationChannelJSON("channel-1")))
	if err != nil {
		t.Fatalf("createResource notification channel returned error: %v", err)
	}
	if skipped {
		t.Fatal("createResource skipped email notification channel")
	}
	if channelResource.InstanceState.ID != "channel-1" {
		t.Fatalf("notification channel resource ID = %q, want channel-1", channelResource.InstanceState.ID)
	}
	if channelResource.InstanceState.Attributes["user_id"] != "user-1" {
		t.Fatalf("notification channel user_id = %q, want user-1", channelResource.InstanceState.Attributes["user_id"])
	}

	_, skipped, err = (&OnCallUserNotificationChannelGenerator{}).createResource("user-1", onCallNotificationChannel(t, onCallPushNotificationChannelJSON("push-1")))
	if err != nil {
		t.Fatalf("createResource push notification channel returned error: %v", err)
	}
	if !skipped {
		t.Fatal("createResource did not skip push notification channel")
	}

	ruleResource, err := (&OnCallUserNotificationRuleGenerator{}).createResource("user-1", onCallNotificationRule(t, onCallNotificationRuleJSON("rule-1", "channel-1")))
	if err != nil {
		t.Fatalf("createResource notification rule returned error: %v", err)
	}
	if ruleResource.InstanceState.ID != "rule-1" {
		t.Fatalf("notification rule resource ID = %q, want rule-1", ruleResource.InstanceState.ID)
	}
	if ruleResource.InstanceState.Attributes["user_id"] != "user-1" {
		t.Fatalf("notification rule user_id = %q, want user-1", ruleResource.InstanceState.Attributes["user_id"])
	}
}

func TestParseOnCallUserChildImportID(t *testing.T) {
	testCases := []struct {
		name      string
		importID  string
		wantUser  string
		wantChild string
		wantErr   bool
	}{
		{
			name:      "colon delimiter",
			importID:  "user-1:channel-1",
			wantUser:  "user-1",
			wantChild: "channel-1",
		},
		{
			name:      "comma delimiter",
			importID:  "user-1,channel-1",
			wantUser:  "user-1",
			wantChild: "channel-1",
		},
		{
			name:     "missing delimiter",
			importID: "user-1",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotUser, gotChild, err := parseOnCallUserChildImportID(tc.importID, "channel")
			if tc.wantErr {
				if err == nil {
					t.Fatal("parseOnCallUserChildImportID returned nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOnCallUserChildImportID returned error: %v", err)
			}
			if gotUser != tc.wantUser {
				t.Fatalf("user id = %q, want %q", gotUser, tc.wantUser)
			}
			if gotChild != tc.wantChild {
				t.Fatalf("child id = %q, want %q", gotChild, tc.wantChild)
			}
		})
	}
}

func TestListDatadogUserIDsUsesSupportedPageSize(t *testing.T) {
	pathCh := make(chan string, 1)
	pageSizeCh := make(chan string, 1)
	pageNumberCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		pageSizeCh <- r.URL.Query().Get("page[size]")
		pageNumberCh <- r.URL.Query().Get("page[number]")
		_, _ = fmt.Fprint(w, onCallUserListResponseJSON(1, "user-1"))
	}))
	defer server.Close()

	api := datadogV2.NewUsersApi(newTeamRelationshipTestClient(server))
	userIDs, err := listDatadogUserIDs(context.Background(), api)
	if err != nil {
		t.Fatalf("listDatadogUserIDs returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/users")
	assertObservedQueryValue(t, pageSizeCh, "page[size]", "100")
	assertObservedQueryValue(t, pageNumberCh, "page[number]", "0")
	if len(userIDs) != 1 {
		t.Fatalf("expected 1 user id, got %d", len(userIDs))
	}
	if userIDs[0] != "user-1" {
		t.Fatalf("user id = %q, want user-1", userIDs[0])
	}
}

func TestListDatadogUserIDsPaginates(t *testing.T) {
	pathCh := make(chan string, 2)
	pageSizeCh := make(chan string, 2)
	pageNumberCh := make(chan string, 2)
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		pageSizeCh <- r.URL.Query().Get("page[size]")
		pageNumberCh <- r.URL.Query().Get("page[number]")
		requestCount++
		switch requestCount {
		case 1:
			_, _ = fmt.Fprint(w, onCallUserListResponseJSON(101, "user-1"))
		default:
			_, _ = fmt.Fprint(w, onCallUserListResponseJSON(101, "user-2"))
		}
	}))
	defer server.Close()

	api := datadogV2.NewUsersApi(newTeamRelationshipTestClient(server))
	userIDs, err := listDatadogUserIDs(context.Background(), api)
	if err != nil {
		t.Fatalf("listDatadogUserIDs returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/users")
	assertObservedQueryValue(t, pageSizeCh, "page[size]", "100")
	assertObservedQueryValue(t, pageNumberCh, "page[number]", "0")
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/users")
	assertObservedQueryValue(t, pageSizeCh, "page[size]", "100")
	assertObservedQueryValue(t, pageNumberCh, "page[number]", "1")
	if len(userIDs) != 2 {
		t.Fatalf("expected 2 user ids, got %d", len(userIDs))
	}
	if userIDs[0] != "user-1" {
		t.Fatalf("first user id = %q, want user-1", userIDs[0])
	}
	if userIDs[1] != "user-2" {
		t.Fatalf("second user id = %q, want user-2", userIDs[1])
	}
}

func TestOnCallEscalationPolicyInitResourcesFiltersByID(t *testing.T) {
	pathCh := make(chan string, 1)
	includeCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		includeCh <- r.URL.Query().Get("include")
		_, _ = fmt.Fprint(w, onCallEscalationPolicyResponseJSON("policy-1"))
	}))
	defer server.Close()

	generator := newOnCallEscalationPolicyTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_escalation_policy",
			FieldPath:        "id",
			AcceptableValues: []string{"policy-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/escalation-policies/policy-1")
	assertObservedQueryValue(t, includeCh, "include", "steps.targets")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func TestOnCallScheduleInitResourcesFiltersByID(t *testing.T) {
	pathCh := make(chan string, 1)
	includeCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		includeCh <- r.URL.Query().Get("include")
		_, _ = fmt.Fprint(w, onCallScheduleResponseJSON("schedule-1"))
	}))
	defer server.Close()

	generator := newOnCallScheduleTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_schedule",
			FieldPath:        "id",
			AcceptableValues: []string{"schedule-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/schedules/schedule-1")
	assertObservedQueryValue(t, includeCh, "include", "layers,layers.members.user")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func TestOnCallTeamRoutingRulesInitResourcesFiltersByID(t *testing.T) {
	pathCh := make(chan string, 1)
	includeCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		includeCh <- r.URL.Query().Get("include")
		_, _ = fmt.Fprint(w, onCallTeamRoutingRulesResponseJSON("team-1"))
	}))
	defer server.Close()

	generator := newOnCallTeamRoutingRulesTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_team_routing_rules",
			FieldPath:        "id",
			AcceptableValues: []string{"team-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/teams/team-1/routing-rules")
	assertObservedQueryValue(t, includeCh, "include", "rules")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
}

func TestOnCallTeamRoutingRulesInitResourcesSkipsMissingFilteredID(t *testing.T) {
	pathCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"errors":["not found"]}`)
	}))
	defer server.Close()

	generator := newOnCallTeamRoutingRulesTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_team_routing_rules",
			FieldPath:        "id",
			AcceptableValues: []string{"team-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/teams/team-1/routing-rules")
	if len(generator.Resources) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(generator.Resources))
	}
}

func TestOnCallUserNotificationChannelInitResourcesFiltersByUserID(t *testing.T) {
	pathCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		_, _ = fmt.Fprint(w, onCallNotificationChannelListResponseJSON(
			onCallEmailNotificationChannelJSON("channel-1"),
			onCallPushNotificationChannelJSON("push-1"),
		))
	}))
	defer server.Close()

	generator := newOnCallUserNotificationChannelTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_user_notification_channel",
			FieldPath:        "user_id",
			AcceptableValues: []string{"user-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/users/user-1/notification-channels")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "channel-1" {
		t.Fatalf("resource ID = %q, want channel-1", generator.Resources[0].InstanceState.ID)
	}
}

func TestOnCallUserNotificationRuleInitResourcesFiltersByID(t *testing.T) {
	pathCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pathCh <- r.URL.Path
		_, _ = fmt.Fprint(w, onCallNotificationRuleResponseJSON("rule-1", "channel-1"))
	}))
	defer server.Close()

	generator := newOnCallUserNotificationRuleTestGenerator(server, []terraformutils.ResourceFilter{
		{
			ServiceName:      "on_call_user_notification_rule",
			FieldPath:        "id",
			AcceptableValues: []string{"user-1,rule-1"},
		},
	})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources returned error: %v", err)
	}
	assertObservedQueryValue(t, pathCh, "path", "/api/v2/on-call/users/user-1/notification-rules/rule-1")
	if len(generator.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(generator.Resources))
	}
	if generator.Filter[0].AcceptableValues[0] != "rule-1" {
		t.Fatalf("rewritten id filter = %v, want [rule-1]", generator.Filter[0].AcceptableValues)
	}
}

func TestDatadogProviderOnCallConnections(t *testing.T) {
	connections := DatadogProvider{}.GetResourceConnections()

	assertDatadogConnection(t, connections, "on_call_schedule", "team", "teams", "id")
	assertDatadogConnection(t, connections, "on_call_schedule", "user", "layer.users", "id")
	assertDatadogConnection(t, connections, "on_call_team_routing_rules", "team", "id", "id")
	assertDatadogConnection(t, connections, "on_call_team_routing_rules", "on_call_escalation_policy", "rule.escalation_policy", "id")
	assertDatadogConnection(t, connections, "on_call_user_notification_channel", "user", "user_id", "id")
	assertDatadogConnection(t, connections, "on_call_user_notification_rule", "on_call_user_notification_channel", "channel_id", "id")
	assertDatadogConnection(t, connections, "on_call_user_notification_rule", "user", "user_id", "id")
	assertDatadogConnectionPairs(
		t,
		connections,
		"on_call_escalation_policy",
		"team",
		[]string{
			"teams", "id",
			"step.target.team", "id",
		},
	)
	assertDatadogConnection(t, connections, "on_call_escalation_policy", "user", "step.target.user", "id")
	assertDatadogConnectionPairs(
		t,
		connections,
		"on_call_escalation_policy",
		"on_call_schedule",
		[]string{"step.target.schedule", "id"},
	)
}

func newOnCallEscalationPolicyTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *OnCallEscalationPolicyGenerator {
	return &OnCallEscalationPolicyGenerator{DatadogService: newOnCallTestService(server, filter)}
}

func newOnCallScheduleTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *OnCallScheduleGenerator {
	return &OnCallScheduleGenerator{DatadogService: newOnCallTestService(server, filter)}
}

func newOnCallTeamRoutingRulesTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *OnCallTeamRoutingRulesGenerator {
	return &OnCallTeamRoutingRulesGenerator{DatadogService: newOnCallTestService(server, filter)}
}

func newOnCallUserNotificationChannelTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *OnCallUserNotificationChannelGenerator {
	return &OnCallUserNotificationChannelGenerator{DatadogService: newOnCallTestService(server, filter)}
}

func newOnCallUserNotificationRuleTestGenerator(server *httptest.Server, filter []terraformutils.ResourceFilter) *OnCallUserNotificationRuleGenerator {
	return &OnCallUserNotificationRuleGenerator{DatadogService: newOnCallTestService(server, filter)}
}

func newOnCallTestService(server *httptest.Server, filter []terraformutils.ResourceFilter) DatadogService {
	return DatadogService{
		Service: terraformutils.Service{
			Args: map[string]interface{}{
				"auth":          context.Background(),
				"datadogClient": newTeamRelationshipTestClient(server),
			},
			Filter: filter,
		},
	}
}

func onCallEscalationPolicy(t *testing.T, id string) datadogV2.EscalationPolicy {
	t.Helper()

	var response datadogV2.EscalationPolicy
	mustUnmarshalDatadog(t, onCallEscalationPolicyResponseJSON(id), &response)
	return response
}

func onCallSchedule(t *testing.T, id string) datadogV2.Schedule {
	t.Helper()

	var response datadogV2.Schedule
	mustUnmarshalDatadog(t, onCallScheduleResponseJSON(id), &response)
	return response
}

func onCallTeamRoutingRules(t *testing.T, id string) datadogV2.TeamRoutingRules {
	t.Helper()

	var response datadogV2.TeamRoutingRules
	mustUnmarshalDatadog(t, onCallTeamRoutingRulesResponseJSON(id), &response)
	return response
}

func onCallNotificationChannel(t *testing.T, rawData string) datadogV2.NotificationChannelData {
	t.Helper()

	var response datadogV2.NotificationChannel
	mustUnmarshalDatadog(t, fmt.Sprintf("{\"data\":%s}", rawData), &response)
	return response.GetData()
}

func onCallNotificationRule(t *testing.T, rawData string) datadogV2.OnCallNotificationRuleData {
	t.Helper()

	var response datadogV2.OnCallNotificationRule
	mustUnmarshalDatadog(t, fmt.Sprintf("{\"data\":%s}", rawData), &response)
	return response.GetData()
}

func mustUnmarshalDatadog(t *testing.T, raw string, target interface{}) {
	t.Helper()

	if err := datadog.Unmarshal([]byte(raw), target); err != nil {
		t.Fatalf("failed to unmarshal Datadog response: %v", err)
	}
}

func onCallEscalationPolicyResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":{\"id\":%q,\"type\":\"policies\"}}", id)
}

func onCallScheduleResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":{\"id\":%q,\"type\":\"schedules\"}}", id)
}

func onCallTeamRoutingRulesResponseJSON(id string) string {
	return fmt.Sprintf("{\"data\":{\"id\":%q,\"type\":\"team_routing_rules\"}}", id)
}

func onCallNotificationChannelListResponseJSON(channels ...string) string {
	return fmt.Sprintf("{\"data\":[%s]}", strings.Join(channels, ","))
}

func onCallUserListResponseJSON(totalCount int, ids ...string) string {
	users := []string{}
	for _, id := range ids {
		users = append(users, fmt.Sprintf("{\"id\":%q,\"type\":\"users\"}", id))
	}
	return fmt.Sprintf("{\"data\":[%s],\"meta\":{\"page\":{\"total_count\":%d}}}", strings.Join(users, ","), totalCount)
}

func onCallEmailNotificationChannelJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"notification_channels\",\"attributes\":{\"config\":{\"type\":\"email\",\"address\":\"user@example.com\",\"formats\":[\"html\"]}}}",
		id,
	)
}

func onCallPushNotificationChannelJSON(id string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"notification_channels\",\"attributes\":{\"config\":{\"type\":\"push\",\"name\":\"mobile\"}}}",
		id,
	)
}

func onCallNotificationRuleResponseJSON(id string, channelID string) string {
	return fmt.Sprintf("{\"data\":%s}", onCallNotificationRuleJSON(id, channelID))
}

func onCallNotificationRuleJSON(id string, channelID string) string {
	return fmt.Sprintf(
		"{\"id\":%q,\"type\":\"notification_rules\",\"attributes\":{\"category\":\"high_urgency\",\"delay_minutes\":0},\"relationships\":{\"channel\":{\"data\":{\"id\":%q,\"type\":\"notification_channels\"}}}}",
		id,
		channelID,
	)
}
