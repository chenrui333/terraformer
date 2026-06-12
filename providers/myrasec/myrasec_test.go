package myrasec

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	mgo "github.com/Myra-Security-GmbH/myrasec-go/v2"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestMyrasecProviderRegistration(t *testing.T) {
	provider := &MyrasecProvider{}

	if err := provider.Init(nil); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if got := provider.GetName(); got != "myrasec" {
		t.Fatalf("GetName() = %q, want %q", got, "myrasec")
	}
	if providerData := provider.GetProviderData(); len(providerData) != 0 {
		t.Fatalf("GetProviderData() = %#v, want empty map", providerData)
	}
	if connections := provider.GetResourceConnections(); len(connections) != 0 {
		t.Fatalf("GetResourceConnections() = %#v, want empty map", connections)
	}

	wantServices := map[string]reflect.Type{
		"cache_setting": reflect.TypeOf(&CacheSettingGenerator{}),
		"dns_record":    reflect.TypeOf(&DNSGenerator{}),
		"domain":        reflect.TypeOf(&DomainGenerator{}),
		"error_page":    reflect.TypeOf(&ErrorPageGenerator{}),
		"ip_filter":     reflect.TypeOf(&IPFilterGenerator{}),
		"maintenance":   reflect.TypeOf(&MaintenanceGenerator{}),
		"redirect":      reflect.TypeOf(&RedirectGenerator{}),
		"settings":      reflect.TypeOf(&SettingsGenerator{}),
		"waf_rule":      reflect.TypeOf(&WafRuleGenerator{}),
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

	if err := provider.InitService("domain", true); err != nil {
		t.Fatalf("InitService(domain) returned error: %v", err)
	}
	service := provider.GetService()
	if service == nil {
		t.Fatal("InitService(domain) left provider service nil")
	}
	if got := service.GetName(); got != "domain" {
		t.Fatalf("selected service name = %q, want %q", got, "domain")
	}
	if got := service.GetProviderName(); got != "myrasec" {
		t.Fatalf("selected service provider = %q, want %q", got, "myrasec")
	}

	if err := provider.InitService("missing", false); err == nil {
		t.Fatal("InitService(missing) returned nil error")
	}
}

func TestNormalizeMyrasecAPIBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain", in: "https://api.example.test", want: "https://api.example.test/%s"},
		{name: "plain with slash", in: "https://api.example.test/", want: "https://api.example.test/%s"},
		{name: "path", in: "https://api.example.test/custom", want: "https://api.example.test/custom/%s"},
		{name: "sdk format", in: "https://api.example.test/%s", want: "https://api.example.test/%s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMyrasecAPIBaseURL(tt.in); got != tt.want {
				t.Fatalf("normalizeMyrasecAPIBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInitializeAPI(t *testing.T) {
	t.Run("missing credentials", func(t *testing.T) {
		t.Setenv("MYRASEC_API_KEY", "")
		t.Setenv("MYRASEC_API_SECRET", "")
		t.Setenv("MYRASEC_API_BASE_URL", "")

		api, err := (&MyrasecService{}).initializeAPI()
		if err == nil {
			t.Fatal("initializeAPI() returned nil error")
		}
		if api != nil {
			t.Fatalf("initializeAPI() api = %#v, want nil", api)
		}
	})

	t.Run("plain base URL", func(t *testing.T) {
		t.Setenv("MYRASEC_API_KEY", "key")
		t.Setenv("MYRASEC_API_SECRET", "secret")
		t.Setenv("MYRASEC_API_BASE_URL", "https://myra.example.test/api")

		api, err := (&MyrasecService{}).initializeAPI()
		if err != nil {
			t.Fatalf("initializeAPI() returned error: %v", err)
		}
		if got, want := api.BaseURL, "https://myra.example.test/api/%s"; got != want {
			t.Fatalf("api.BaseURL = %q, want %q", got, want)
		}
	})

	t.Run("empty base URL keeps SDK default", func(t *testing.T) {
		t.Setenv("MYRASEC_API_KEY", "key")
		t.Setenv("MYRASEC_API_SECRET", "secret")
		t.Setenv("MYRASEC_API_BASE_URL", "")

		defaultAPI, err := mgo.New("key", "secret")
		if err != nil {
			t.Fatalf("mgo.New() returned error: %v", err)
		}
		api, err := (&MyrasecService{}).initializeAPI()
		if err != nil {
			t.Fatalf("initializeAPI() returned error: %v", err)
		}
		if got, want := api.BaseURL, defaultAPI.BaseURL; got != want {
			t.Fatalf("api.BaseURL = %q, want SDK default %q", got, want)
		}
	})
}

func TestDomainGeneratorInitResourcesUsesPlainBaseURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains" {
			t.Fatalf("request path = %q, want %q", r.URL.Path, "/domains")
		}
		assertQueryValue(t, r, "page", "1")
		assertQueryValue(t, r, "pageSize", "250")
		writeMyrasecData(t, w, `[{"id":42,"name":"example.com"}]`)
	}))
	t.Cleanup(server.Close)

	t.Setenv("MYRASEC_API_KEY", "key")
	t.Setenv("MYRASEC_API_SECRET", "secret")
	t.Setenv("MYRASEC_API_BASE_URL", server.URL)

	generator := &DomainGenerator{}
	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() returned error: %v", err)
	}

	resources := generator.GetResources()
	if len(resources) != 1 {
		t.Fatalf("InitResources() produced %d resources, want 1", len(resources))
	}
	assertResource(t, resources[0], "42", terraformutils.TfSanitize("example.com_42"), "myrasec_domain", map[string]string{}, []string{"^metadata"})
}

func TestCreateDomainResource(t *testing.T) {
	generator := &DomainGenerator{}
	err := generator.createDomainResource(nil, mgo.Domain{ID: 42, Name: "example.com"})
	if err != nil {
		t.Fatalf("createDomainResource() returned error: %v", err)
	}

	resources := generator.GetResources()
	if len(resources) != 1 {
		t.Fatalf("createDomainResource() produced %d resources, want 1", len(resources))
	}
	assertResource(t, resources[0], "42", terraformutils.TfSanitize("example.com_42"), "myrasec_domain", map[string]string{}, []string{"^metadata"})
}

func TestCreateResourcesPerDomainPaginationEmptyAndError(t *testing.T) {
	t.Run("paginates until short page", func(t *testing.T) {
		var pages []string
		api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/domains" {
				t.Fatalf("request path = %q, want %q", r.URL.Path, "/domains")
			}
			page := r.URL.Query().Get("page")
			pages = append(pages, page)
			assertQueryValue(t, r, "pageSize", "250")
			switch page {
			case "1":
				writeMyrasecData(t, w, domainsJSON(1, 250))
			case "2":
				writeMyrasecData(t, w, domainsJSON(251, 1))
			default:
				t.Fatalf("unexpected page %q", page)
			}
		}))

		var ids []int
		err := createResourcesPerDomain(api, []func(*mgo.API, mgo.Domain) error{
			func(_ *mgo.API, domain mgo.Domain) error {
				ids = append(ids, domain.ID)
				return nil
			},
		})
		if err != nil {
			t.Fatalf("createResourcesPerDomain() returned error: %v", err)
		}
		if got, want := len(ids), 251; got != want {
			t.Fatalf("visited domains = %d, want %d", got, want)
		}
		if got, want := pages, []string{"1", "2"}; !slices.Equal(got, want) {
			t.Fatalf("pages = %v, want %v", got, want)
		}
		if ids[0] != 1 || ids[250] != 251 {
			t.Fatalf("domain ids were not visited in order: first=%d last=%d", ids[0], ids[250])
		}
	})

	t.Run("empty response", func(t *testing.T) {
		api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeMyrasecData(t, w, `[]`)
		}))

		called := false
		err := createResourcesPerDomain(api, []func(*mgo.API, mgo.Domain) error{
			func(_ *mgo.API, _ mgo.Domain) error {
				called = true
				return nil
			},
		})
		if err != nil {
			t.Fatalf("createResourcesPerDomain() returned error: %v", err)
		}
		if called {
			t.Fatal("createResourcesPerDomain() called callback for empty response")
		}
	})

	t.Run("api error", func(t *testing.T) {
		api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeMyrasecError(t, w, "domain list failed")
		}))

		err := createResourcesPerDomain(api, []func(*mgo.API, mgo.Domain) error{
			func(_ *mgo.API, _ mgo.Domain) error {
				t.Fatal("callback must not be called after API error")
				return nil
			},
		})
		if err == nil || !strings.Contains(err.Error(), "domain list failed") {
			t.Fatalf("createResourcesPerDomain() error = %v, want domain list failed", err)
		}
	})
}

func TestCreateResourcesPerSubDomainIncludesDomainLevelAndSubdomains(t *testing.T) {
	api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/domains":
			writeMyrasecData(t, w, `[{"id":7,"name":"example.com"}]`)
		case "/domain/7/subdomains":
			writeMyrasecData(t, w, `[{"id":8,"label":"www.example.com","value":"www.example.com.","domainName":"example.com"}]`)
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))

	var (
		mu     sync.Mutex
		labels []string
	)
	err := createResourcesPerSubDomain(api, []func(*mgo.API, int, mgo.VHost) error{
		func(_ *mgo.API, domainID int, vhost mgo.VHost) error {
			mu.Lock()
			defer mu.Unlock()
			labels = append(labels, strconv.Itoa(domainID)+":"+vhost.Label)
			return nil
		},
	}, true)
	if err != nil {
		t.Fatalf("createResourcesPerSubDomain() returned error: %v", err)
	}

	slices.Sort(labels)
	want := []string{"7:ALL-7.", "7:www.example.com"}
	if !slices.Equal(labels, want) {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
}

func TestMyrasecResourceGeneratorsCreateResources(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		response   func(t *testing.T, w http.ResponseWriter)
		checkQuery func(t *testing.T, r *http.Request)
		run        func(api *mgo.API) ([]terraformutils.Resource, error)
		wantID     string
		wantName   string
		wantType   string
		wantAttrs  map[string]string
		wantIgnore []string
	}{
		{
			name: "dns record",
			path: "/domain/7/dns-records",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":101,"name":"www.example.com.","value":"127.0.0.1","ttl":300,"recordType":"A"}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &DNSGenerator{}
				err := generator.createDnsResources(api, mgo.Domain{ID: 7, Name: "example.com"})
				return generator.GetResources(), err
			},
			wantID:   "101",
			wantName: terraformutils.TfSanitize("example.com_101"),
			wantType: "myrasec_dns_record",
			wantAttrs: map[string]string{
				"domain_name": "example.com",
			},
			wantIgnore: []string{"^metadata"},
		},
		{
			name: "cache setting",
			path: "/domain/7/www.example.com/cache-settings",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":201}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &CacheSettingGenerator{}
				err := generator.createCacheSettingResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "201",
			wantName: terraformutils.TfSanitize("www.example.com_201"),
			wantType: "myrasec_cache_setting",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
			},
			wantIgnore: []string{"^Metadata"},
		},
		{
			name: "redirect",
			path: "/domain/7/redirects/www.example.com",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":301,"subDomainName":"www.example.com"}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &RedirectGenerator{}
				err := generator.createRedirectResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "301",
			wantName: terraformutils.TfSanitize("www.example.com_301"),
			wantType: "myrasec_redirect",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
			},
		},
		{
			name: "ip filter",
			path: "/domain/7/ip-filters/www.example.com",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":401}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &IPFilterGenerator{}
				err := generator.createIPFilterResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "401",
			wantName: terraformutils.TfSanitize("www.example.com_401"),
			wantType: "myrasec_ip_filter",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
			},
		},
		{
			name: "maintenance",
			path: "/domain/7/www.example.com/maintenances",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":501}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &MaintenanceGenerator{}
				err := generator.createMaintenanceResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "501",
			wantName: terraformutils.TfSanitize("www.example.com_501"),
			wantType: "myrasec_maintenance",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
			},
		},
		{
			name: "error page",
			path: "/domain/7/errorpages",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":601,"subDomainName":"www.example.com","errorCode":404,"content":"not found"}]`)
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &ErrorPageGenerator{}
				err := generator.createErrorPageResources(api, mgo.Domain{ID: 7, Name: "example.com"})
				return generator.GetResources(), err
			},
			wantID:   "601",
			wantName: terraformutils.TfSanitize("www.example.com_601"),
			wantType: "myrasec_error_page",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
				"error_code":     "404",
				"content":        "not found",
			},
			wantIgnore: []string{"^metadata"},
		},
		{
			name: "settings",
			path: "/domain/7/www.example.com/settings",
			response: func(t *testing.T, w http.ResponseWriter) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(`{"only_https":true}`)); err != nil {
					t.Fatalf("write response: %v", err)
				}
			},
			checkQuery: func(t *testing.T, r *http.Request) {
				if _, ok := r.URL.Query()["flat"]; !ok {
					t.Fatalf("query = %q, want flat parameter", r.URL.RawQuery)
				}
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &SettingsGenerator{}
				err := generator.createSettingResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "8",
			wantName: terraformutils.TfSanitize("www.example.com_8"),
			wantType: "myrasec_settings",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
				"only_https":     "true",
			},
		},
		{
			name: "waf rule",
			path: "/domain/7/waf-rules",
			response: func(t *testing.T, w http.ResponseWriter) {
				writeMyrasecData(t, w, `[{"id":701,"subDomainName":"www.example.com"}]`)
			},
			checkQuery: func(t *testing.T, r *http.Request) {
				assertQueryValue(t, r, "subDomain", "www.example.com")
			},
			run: func(api *mgo.API) ([]terraformutils.Resource, error) {
				generator := &WafRuleGenerator{}
				err := generator.createWafRuleResources(api, 7, mgo.VHost{ID: 8, Label: "www.example.com"})
				return generator.GetResources(), err
			},
			wantID:   "701",
			wantName: terraformutils.TfSanitize("www.example.com_701"),
			wantType: "myrasec_waf_rule",
			wantAttrs: map[string]string{
				"subdomain_name": "www.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("request path = %q, want %q", r.URL.Path, tt.path)
				}
				if tt.name != "settings" {
					assertQueryValue(t, r, "page", "1")
					assertQueryValue(t, r, "pageSize", "250")
				}
				if tt.checkQuery != nil {
					tt.checkQuery(t, r)
				}
				tt.response(t, w)
			}))

			resources, err := tt.run(api)
			if err != nil {
				t.Fatalf("generator returned error: %v", err)
			}
			if len(resources) != 1 {
				t.Fatalf("generator produced %d resources, want 1", len(resources))
			}
			assertResource(t, resources[0], tt.wantID, tt.wantName, tt.wantType, tt.wantAttrs, tt.wantIgnore)
		})
	}
}

func TestDNSGeneratorPropagatesAPIError(t *testing.T) {
	api := newTestMyrasecAPI(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeMyrasecError(t, w, "dns list failed")
	}))

	generator := &DNSGenerator{}
	err := generator.createDnsResources(api, mgo.Domain{ID: 7, Name: "example.com"})
	if err == nil || !strings.Contains(err.Error(), "dns list failed") {
		t.Fatalf("createDnsResources() error = %v, want dns list failed", err)
	}
	if resources := generator.GetResources(); len(resources) != 0 {
		t.Fatalf("createDnsResources() produced resources after API error: %#v", resources)
	}
}

func newTestMyrasecAPI(t *testing.T, handler http.Handler) *mgo.API {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	api, err := mgo.New("key", "secret")
	if err != nil {
		t.Fatalf("mgo.New() returned error: %v", err)
	}
	api.BaseURL = normalizeMyrasecAPIBaseURL(server.URL)
	api.SetMaxRetries(1)
	api.DisableCaching()
	return api
}

func writeMyrasecData(t *testing.T, w http.ResponseWriter, data string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprintf(w, `{"error":false,"pageSize":250,"page":1,"count":1,"data":%s}`, data); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func writeMyrasecError(t *testing.T, w http.ResponseWriter, message string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprintf(w, `{"error":true,"errorMessage":%q}`, message); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func domainsJSON(start, count int) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i := range count {
		if i > 0 {
			builder.WriteByte(',')
		}
		id := start + i
		fmt.Fprintf(&builder, `{"id":%d,"name":"example-%d.com"}`, id, id)
	}
	builder.WriteByte(']')
	return builder.String()
}

func assertResource(t *testing.T, resource terraformutils.Resource, wantID, wantName, wantType string, wantAttrs map[string]string, wantIgnore []string) {
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
	for _, ignore := range wantIgnore {
		if !slices.Contains(resource.IgnoreKeys, ignore) {
			t.Fatalf("resource ignore keys = %#v, want to contain %q", resource.IgnoreKeys, ignore)
		}
	}
}

func assertQueryValue(t *testing.T, r *http.Request, key, want string) {
	t.Helper()

	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("query %q = %q, want %q; raw query %q", key, got, want, r.URL.RawQuery)
	}
}
