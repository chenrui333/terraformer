// SPDX-License-Identifier: Apache-2.0

package linode

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/linode/linodego/v2"
)

func TestLinodeProviderRegistration(t *testing.T) {
	t.Setenv("LINODE_TOKEN", "token")

	provider := &LinodeProvider{}
	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if got := provider.GetName(); got != "linode" {
		t.Fatalf("GetName() = %q, want %q", got, "linode")
	}
	if providerData := provider.GetProviderData(); len(providerData) != 0 {
		t.Fatalf("GetProviderData() = %#v, want empty map", providerData)
	}
	if connections := provider.GetResourceConnections(); len(connections) != 0 {
		t.Fatalf("GetResourceConnections() = %#v, want empty map", connections)
	}

	wantServices := map[string]reflect.Type{
		"domain":       reflect.TypeOf(&DomainGenerator{}),
		"image":        reflect.TypeOf(&ImageGenerator{}),
		"instance":     reflect.TypeOf(&InstanceGenerator{}),
		"nodebalancer": reflect.TypeOf(&NodeBalancerGenerator{}),
		"rdns":         reflect.TypeOf(&RDNSGenerator{}),
		"sshkey":       reflect.TypeOf(&SSHKeyGenerator{}),
		"stackscript":  reflect.TypeOf(&StackScriptGenerator{}),
		"token":        reflect.TypeOf(&TokenGenerator{}),
		"volume":       reflect.TypeOf(&VolumeGenerator{}),
	}
	services := provider.GetSupportedService()
	if len(services) != len(wantServices) {
		t.Fatalf("GetSupportedService() returned %d services, want %d", len(services), len(wantServices))
	}
	for name, wantType := range wantServices {
		service, ok := services[name]
		if !ok {
			t.Fatalf("supported services missing %q", name)
		}
		if gotType := reflect.TypeOf(service); gotType != wantType {
			t.Fatalf("service %q type = %v, want %v", name, gotType, wantType)
		}
	}

	if err := provider.InitService("nodebalancer", true); err != nil {
		t.Fatalf("InitService(nodebalancer) returned error: %v", err)
	}
	service := provider.GetService()
	if service == nil {
		t.Fatal("InitService(nodebalancer) left provider service nil")
	}
	if got := service.GetName(); got != "nodebalancer" {
		t.Fatalf("selected service name = %q, want %q", got, "nodebalancer")
	}
	if got := service.GetProviderName(); got != "linode" {
		t.Fatalf("selected service provider = %q, want %q", got, "linode")
	}
	if got := service.GetArgs()["token"]; got != "token" {
		t.Fatalf("selected service token arg = %#v, want token", got)
	}

	if err := provider.InitService("missing", false); err == nil {
		t.Fatal("InitService(missing) returned nil error")
	}
}

func TestLinodeResourceGeneratorsCreateResources(t *testing.T) {
	tests := []struct {
		name      string
		resources []terraformutils.Resource
		wantIDs   []string
		wantNames []string
		wantTypes []string
		wantAttrs []map[string]string
	}{
		{
			name:      "image",
			resources: (&ImageGenerator{}).createResources([]linodego.Image{{ID: "private/42"}}),
			wantIDs:   []string{"private/42"},
			wantNames: []string{terraformutils.TfSanitize("private/42")},
			wantTypes: []string{"linode_image"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name:      "instance",
			resources: (&InstanceGenerator{}).createResources([]linodego.Instance{{ID: 101}}),
			wantIDs:   []string{"101"},
			wantNames: []string{terraformutils.TfSanitize("101")},
			wantTypes: []string{"linode_instance"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name:      "rdns",
			resources: (&RDNSGenerator{}).createResources([]linodego.InstanceIP{{Address: "192.0.2.10"}}),
			wantIDs:   []string{"192.0.2.10"},
			wantNames: []string{terraformutils.TfSanitize("192.0.2.10")},
			wantTypes: []string{"linode_rdns"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name:      "sshkey",
			resources: (&SSHKeyGenerator{}).createResources([]linodego.SSHKey{{ID: 201}}),
			wantIDs:   []string{"201"},
			wantNames: []string{terraformutils.TfSanitize("201")},
			wantTypes: []string{"linode_sshkey"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name: "stackscript excludes public scripts",
			resources: (&StackScriptGenerator{}).createResources([]linodego.Stackscript{
				{ID: 301, IsPublic: false},
				{ID: 302, IsPublic: true},
			}),
			wantIDs:   []string{"301"},
			wantNames: []string{terraformutils.TfSanitize("301")},
			wantTypes: []string{"linode_stackscript"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name:      "token",
			resources: (&TokenGenerator{}).createResources([]linodego.Token{{ID: 401}}),
			wantIDs:   []string{"401"},
			wantNames: []string{terraformutils.TfSanitize("401")},
			wantTypes: []string{"linode_token"},
			wantAttrs: []map[string]string{{}},
		},
		{
			name:      "volume",
			resources: (&VolumeGenerator{}).createResources([]linodego.Volume{{ID: 501}}),
			wantIDs:   []string{"501"},
			wantNames: []string{terraformutils.TfSanitize("501")},
			wantTypes: []string{"linode_volume"},
			wantAttrs: []map[string]string{{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.resources) != len(tt.wantIDs) {
				t.Fatalf("resources length = %d, want %d", len(tt.resources), len(tt.wantIDs))
			}
			for i, resource := range tt.resources {
				assertLinodeResource(t, resource, tt.wantIDs[i], tt.wantNames[i], tt.wantTypes[i], tt.wantAttrs[i])
			}
		})
	}
}

func TestDomainGeneratorLoadDomainsAndRecords(t *testing.T) {
	var requests []string
	client := newTestLinodeClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		switch r.URL.Path {
		case "/v4/domains":
			page := assertLinodePageQuery(t, r)
			switch page {
			case 1:
				writeLinodePage(t, w, 1, 2, []map[string]any{{"id": 11, "domain": "example.com"}})
			case 2:
				writeLinodePage(t, w, 2, 2, []map[string]any{{"id": 12, "domain": "example.org"}})
			default:
				unexpectedLinodePage(t, w, r, page)
			}
		case "/v4/domains/11/records":
			assertLinodePageQuery(t, r)
			writeLinodePage(t, w, 1, 1, []map[string]any{{"id": 101, "type": "A", "name": "www", "target": "192.0.2.10"}})
		case "/v4/domains/12/records":
			assertLinodePageQuery(t, r)
			writeLinodePage[map[string]any](t, w, 1, 1, nil)
		default:
			unexpectedLinodeRequest(t, w, r)
		}
	})

	generator := &DomainGenerator{}
	domains, err := generator.loadDomains(client)
	if err != nil {
		t.Fatalf("loadDomains() returned error: %v", err)
	}
	if got := len(domains); got != 2 {
		t.Fatalf("loadDomains() returned %d domains, want 2; requests=%v", got, requests)
	}
	if err := generator.loadDomainRecords(client, domains[0].ID); err != nil {
		t.Fatalf("loadDomainRecords(11) returned error: %v", err)
	}
	if err := generator.loadDomainRecords(client, domains[1].ID); err != nil {
		t.Fatalf("loadDomainRecords(12) returned error: %v", err)
	}

	resources := generator.GetResources()
	if len(resources) != 3 {
		t.Fatalf("resources length = %d, want 3", len(resources))
	}
	assertLinodeResource(t, resources[0], "11", terraformutils.TfSanitize("11"), "linode_domain", map[string]string{})
	assertLinodeResource(t, resources[1], "12", terraformutils.TfSanitize("12"), "linode_domain", map[string]string{})
	assertLinodeResource(t, resources[2], "101", terraformutils.TfSanitize("101"), "linode_domain_record", map[string]string{"domain_id": "11"})
}

func TestNodeBalancerGeneratorInitResourcesPaginatesParentsConfigsAndNodes(t *testing.T) {
	seenPages := map[string][]int{}
	client := newTestLinodeClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := assertLinodePageQuery(t, r)
		seenPages[r.URL.Path] = append(seenPages[r.URL.Path], page)

		switch r.URL.Path {
		case "/v4/nodebalancers":
			switch page {
			case 1:
				writeLinodePage(t, w, 1, 2, []map[string]any{{"id": 101, "label": "nb-one", "region": "us-east"}})
			case 2:
				writeLinodePage(t, w, 2, 2, []map[string]any{{"id": 102, "label": "nb-two", "region": "us-east"}})
			default:
				unexpectedLinodePage(t, w, r, page)
			}
		case "/v4/nodebalancers/101/configs":
			writeLinodePage(t, w, 1, 1, []map[string]any{{"id": 201, "port": 80, "protocol": "http"}})
		case "/v4/nodebalancers/101/configs/201/nodes":
			switch page {
			case 1:
				writeLinodePage(t, w, 1, 2, []map[string]any{{"id": 301, "label": "node-one", "address": "192.0.2.10:80"}})
			case 2:
				writeLinodePage(t, w, 2, 2, []map[string]any{{"id": 302, "label": "node-two", "address": "192.0.2.11:80"}})
			default:
				unexpectedLinodePage(t, w, r, page)
			}
		case "/v4/nodebalancers/102/configs":
			writeLinodePage[map[string]any](t, w, 1, 1, nil)
		default:
			unexpectedLinodeRequest(t, w, r)
		}
	})

	generator := &NodeBalancerGenerator{}
	if err := generator.initResources(client); err != nil {
		t.Fatalf("initResources() returned error: %v", err)
	}

	resources := generator.GetResources()
	if len(resources) != 5 {
		t.Fatalf("resources length = %d, want 5", len(resources))
	}
	assertLinodeResource(t, resources[0], "101", terraformutils.TfSanitize("101"), "linode_nodebalancer", map[string]string{})
	assertLinodeResource(t, resources[1], "102", terraformutils.TfSanitize("102"), "linode_nodebalancer", map[string]string{})
	assertLinodeResource(t, resources[2], "201", terraformutils.TfSanitize("201"), "linode_nodebalancer_config", map[string]string{"nodebalancer_id": "101"})
	assertLinodeResource(t, resources[3], "301", terraformutils.TfSanitize("301"), "linode_nodebalancer_node", map[string]string{"nodebalancer_id": "101", "config_id": "201"})
	assertLinodeResource(t, resources[4], "302", terraformutils.TfSanitize("302"), "linode_nodebalancer_node", map[string]string{"nodebalancer_id": "101", "config_id": "201"})

	assertPages(t, seenPages["/v4/nodebalancers"], []int{1, 2})
	assertPages(t, seenPages["/v4/nodebalancers/101/configs"], []int{1})
	assertPages(t, seenPages["/v4/nodebalancers/101/configs/201/nodes"], []int{1, 2})
	assertPages(t, seenPages["/v4/nodebalancers/102/configs"], []int{1})
}

func TestNodeBalancerGeneratorInitResourcesEmptyResponse(t *testing.T) {
	client := newTestLinodeClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/nodebalancers" {
			unexpectedLinodeRequest(t, w, r)
			return
		}
		assertLinodePageQuery(t, r)
		writeLinodePage[map[string]any](t, w, 1, 1, nil)
	})

	generator := &NodeBalancerGenerator{}
	if err := generator.initResources(client); err != nil {
		t.Fatalf("initResources() returned error: %v", err)
	}
	if resources := generator.GetResources(); len(resources) != 0 {
		t.Fatalf("resources length = %d, want 0", len(resources))
	}
}

func TestNodeBalancerGeneratorInitResourcesUsesGeneratedClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer token"; got != want {
			t.Fatalf("Authorization header = %q, want %q", got, want)
		}
		switch r.URL.Path {
		case "/v4/nodebalancers":
			assertLinodePageQuery(t, r)
			writeLinodePage(t, w, 1, 1, []map[string]any{{"id": 101, "label": "nb-one", "region": "us-east"}})
		case "/v4/nodebalancers/101/configs":
			assertLinodePageQuery(t, r)
			writeLinodePage[map[string]any](t, w, 1, 1, nil)
		default:
			unexpectedLinodeRequest(t, w, r)
		}
	}))
	t.Cleanup(server.Close)
	caPath := writeTestLinodeCA(t, server)
	t.Setenv("LINODE_URL", server.URL)
	t.Setenv("LINODE_API_VERSION", "v4")
	t.Setenv("LINODE_CA", caPath)

	generator := &NodeBalancerGenerator{}
	generator.SetArgs(map[string]interface{}{"token": "token"})
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() returned error: %v", err)
	}

	resources := generator.GetResources()
	if len(resources) != 1 {
		t.Fatalf("resources length = %d, want 1", len(resources))
	}
	assertLinodeResource(t, resources[0], "101", terraformutils.TfSanitize("101"), "linode_nodebalancer", map[string]string{})
}

func writeTestLinodeCA(t *testing.T, server *httptest.Server) string {
	t.Helper()

	cert := server.Certificate()
	if cert == nil {
		t.Fatal("test server did not expose a certificate")
	}
	caPath := t.TempDir() + "/linode-test-ca.pem"
	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		t.Fatalf("write test CA: %v", err)
	}
	return caPath
}

func TestNodeBalancerGeneratorInitResourcesWrapsChildErrors(t *testing.T) {
	client := newTestLinodeClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v4/nodebalancers":
			assertLinodePageQuery(t, r)
			writeLinodePage(t, w, 1, 1, []map[string]any{{"id": 101, "label": "nb-one", "region": "us-east"}})
		case "/v4/nodebalancers/101/configs":
			assertLinodePageQuery(t, r)
			writeLinodeError(t, w, http.StatusInternalServerError, "configs unavailable")
		default:
			unexpectedLinodeRequest(t, w, r)
		}
	})

	err := (&NodeBalancerGenerator{}).initResources(client)
	if err == nil {
		t.Fatal("initResources() returned nil error")
	}
	if got := err.Error(); !strings.Contains(got, "list configs for nodebalancer 101") || !strings.Contains(got, "configs unavailable") {
		t.Fatalf("initResources() error = %q, want parent context and API reason", got)
	}
}

func newTestLinodeClient(t *testing.T, handler http.HandlerFunc) linodego.Client {
	t.Helper()
	t.Setenv("LINODE_API_VERSION", "v4")

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := linodego.NewClient(server.Client())
	if err != nil {
		t.Fatalf("linodego.NewClient() returned error: %v", err)
	}
	client.SetBaseURL(server.URL)
	client.SetRetryCount(1)
	client.UseCache(false)
	return client
}

func writeLinodePage[T any](t *testing.T, w http.ResponseWriter, page, pages int, data []T) {
	t.Helper()

	if data == nil {
		data = []T{}
	}
	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"data":    data,
		"page":    page,
		"pages":   pages,
		"results": len(data),
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func writeLinodeError(t *testing.T, w http.ResponseWriter, status int, reason string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]string{{"reason": reason}},
	}); err != nil {
		t.Fatalf("write error response: %v", err)
	}
}

func unexpectedLinodeRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	t.Errorf("unexpected request path %q raw query %q", r.URL.Path, r.URL.RawQuery)
	writeLinodeError(t, w, http.StatusNotFound, "unexpected test request")
}

func unexpectedLinodePage(t *testing.T, w http.ResponseWriter, r *http.Request, page int) {
	t.Helper()

	t.Errorf("unexpected request page %d for path %q raw query %q", page, r.URL.Path, r.URL.RawQuery)
	writeLinodeError(t, w, http.StatusNotFound, "unexpected test page")
}

func assertLinodePageQuery(t *testing.T, r *http.Request) int {
	t.Helper()

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		t.Fatalf("request %s query page = %q, want integer: %v", r.URL.Path, r.URL.Query().Get("page"), err)
	}
	return page
}

func assertPages(t *testing.T, got, want []int) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
}

func assertLinodeResource(t *testing.T, resource terraformutils.Resource, wantID, wantName, wantType string, wantAttrs map[string]string) {
	t.Helper()

	if resource.InstanceState == nil {
		t.Fatal("resource InstanceState is nil")
	}
	if resource.InstanceInfo == nil {
		t.Fatal("resource InstanceInfo is nil")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != wantName {
		t.Fatalf("resource name = %q, want %q", got, wantName)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
	if !reflect.DeepEqual(resource.InstanceState.Attributes, wantAttrs) {
		t.Fatalf("resource attrs = %#v, want %#v", resource.InstanceState.Attributes, wantAttrs)
	}
	if resource.Provider != "linode" {
		t.Fatalf("resource provider = %q, want linode", resource.Provider)
	}
	if !strings.HasPrefix(resource.InstanceInfo.Id, fmt.Sprintf("%s.", wantType)) {
		t.Fatalf("resource InstanceInfo.Id = %q, want %s.<name>", resource.InstanceInfo.Id, wantType)
	}
}
