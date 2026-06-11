// SPDX-License-Identifier: Apache-2.0

package cloudflarev7

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	cfsdk "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/option"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

type apiConfig struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

type Option func(*apiConfig)

func BaseURL(baseURL string) Option {
	return func(config *apiConfig) {
		config.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func UsingRateLimit(_ int) Option {
	return func(*apiConfig) {}
}

func UsingRetryPolicy(maxRetries int, _, _ time.Duration) Option {
	return func(config *apiConfig) {
		config.maxRetries = maxRetries
	}
}

type API struct {
	client *cfsdk.Client
}

func NewWithAPIToken(apiToken string, opts ...Option) (*API, error) {
	config := newAPIConfig(opts...)
	v7Options := []option.RequestOption{
		option.WithAPIToken(apiToken),
		option.WithBaseURL(config.baseURL),
		option.WithHTTPClient(config.httpClient),
		option.WithMaxRetries(config.maxRetries),
	}
	return &API{
		client: cfsdk.NewClient(v7Options...),
	}, nil
}

func New(apiKey, apiEmail string, opts ...Option) (*API, error) {
	config := newAPIConfig(opts...)
	v7Options := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithAPIEmail(apiEmail),
		option.WithBaseURL(config.baseURL),
		option.WithHTTPClient(config.httpClient),
		option.WithMaxRetries(config.maxRetries),
	}
	return &API{
		client: cfsdk.NewClient(v7Options...),
	}, nil
}

func newAPIConfig(opts ...Option) apiConfig {
	config := apiConfig{
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
		maxRetries: 4,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return config
}

type ResponseInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Success       bool           `json:"success"`
	Errors        []ResponseInfo `json:"errors"`
	Messages      []ResponseInfo `json:"messages"`
	ErrorMessages []string       `json:"-"`
}

type ResultInfoCursors struct {
	Before string `json:"before" url:"before,omitempty"`
	After  string `json:"after" url:"after,omitempty"`
}

type ResultInfo struct {
	Page       int               `json:"page" url:"page,omitempty"`
	PerPage    int               `json:"per_page" url:"per_page,omitempty"`
	TotalPages int               `json:"total_pages" url:"-"`
	Count      int               `json:"count" url:"-"`
	Total      int               `json:"total_count" url:"-"`
	Cursor     string            `json:"cursor" url:"cursor,omitempty"`
	Cursors    ResultInfoCursors `json:"cursors" url:"cursors,omitempty"`
}

func (p ResultInfo) HasMorePages() bool {
	if totalPages := p.totalPages(); totalPages > 0 {
		return p.Page >= 1 && p.Page < totalPages
	}
	if p.Cursor != "" || p.Cursors.After != "" {
		return true
	}
	return false
}

func (p ResultInfo) totalPages() int {
	if p.TotalPages > 0 {
		return p.TotalPages
	}
	if p.Total > 0 && p.PerPage > 0 {
		return (p.Total + p.PerPage - 1) / p.PerPage
	}
	return 0
}

func (p ResultInfo) Done() bool {
	return !p.HasMorePages()
}

func (p ResultInfo) Next() ResultInfo {
	if p.Cursors.After != "" {
		p.Cursor = p.Cursors.After
		return p
	}
	if p.Cursor != "" {
		return p
	}
	if p.Page == 0 {
		p.Page = 1
	}
	p.Page++
	return p
}

type PaginationOptions struct {
	Page    int    `url:"page,omitempty"`
	PerPage int    `url:"per_page,omitempty"`
	Cursor  string `url:"cursor,omitempty"`
}

type RawResponse struct {
	Response
	Result     json.RawMessage `json:"result"`
	ResultInfo *ResultInfo     `json:"result_info,omitempty"`
}

type Error struct {
	Success       bool           `json:"success"`
	Errors        []ResponseInfo `json:"errors"`
	ErrorMessages []string       `json:"-"`
}

func (e *Error) Error() string {
	messages := e.messages()
	if len(messages) == 0 {
		return "cloudflare API error"
	}
	return strings.Join(messages, ", ")
}

func (e *Error) messages() []string {
	if e == nil {
		return nil
	}
	if len(e.ErrorMessages) > 0 {
		return e.ErrorMessages
	}
	messages := make([]string, 0, len(e.Errors))
	for _, info := range e.Errors {
		if info.Message != "" {
			messages = append(messages, info.Message)
		}
	}
	return messages
}

type RequestError struct {
	error *Error
}

func NewRequestError(err *Error) RequestError {
	return RequestError{error: normalizeError(err)}
}

func (e *RequestError) Error() string {
	return e.error.Error()
}

func (e *RequestError) Messages() []string {
	return e.error.messages()
}

func (e *RequestError) ErrorMessages() []string {
	return e.error.messages()
}

type AuthenticationError struct {
	RequestError
}

func NewAuthenticationError(err *Error) AuthenticationError {
	return AuthenticationError{RequestError: NewRequestError(err)}
}

type AuthorizationError struct {
	RequestError
}

func NewAuthorizationError(err *Error) AuthorizationError {
	return AuthorizationError{RequestError: NewRequestError(err)}
}

type NotFoundError struct {
	RequestError
}

func NewNotFoundError(err *Error) NotFoundError {
	return NotFoundError{RequestError: NewRequestError(err)}
}

func normalizeError(err *Error) *Error {
	if err == nil {
		return &Error{}
	}
	if len(err.ErrorMessages) == 0 {
		err.ErrorMessages = make([]string, 0, len(err.Errors))
		for _, info := range err.Errors {
			if info.Message != "" {
				err.ErrorMessages = append(err.ErrorMessages, info.Message)
			}
		}
	}
	return err
}

func (api *API) Raw(ctx context.Context, method, endpoint string, data interface{}, headers http.Header) (RawResponse, error) {
	var raw RawResponse
	options := make([]option.RequestOption, 0, len(headers))
	for key, values := range headers {
		for _, value := range values {
			options = append(options, option.WithHeaderAdd(key, value))
		}
	}
	if data != nil {
		options = append(options, option.WithRequestBody("application/json", data))
	}
	if err := api.client.Execute(ctx, method, endpoint, nil, &raw, options...); err != nil {
		var apiErr *cfsdk.Error
		if errors.As(err, &apiErr) {
			return raw, classifyV7Error(apiErr)
		}
		return raw, err
	}
	populateErrorMessages(&raw.Response)
	if !raw.Success {
		apiErr := &Error{
			Success:       raw.Success,
			Errors:        raw.Errors,
			ErrorMessages: raw.ErrorMessages,
		}
		return raw, classifyError(http.StatusBadRequest, apiErr)
	}
	return raw, nil
}

func classifyError(statusCode int, err *Error) error {
	normalized := normalizeError(err)
	switch statusCode {
	case http.StatusUnauthorized:
		err := NewAuthenticationError(normalized)
		return &err
	case http.StatusForbidden:
		err := NewAuthorizationError(normalized)
		return &err
	case http.StatusNotFound:
		err := NewNotFoundError(normalized)
		return &err
	default:
		err := NewRequestError(normalized)
		return &err
	}
}

func classifyV7Error(err *cfsdk.Error) error {
	apiErr := &Error{Errors: make([]ResponseInfo, 0, len(err.Errors))}
	for _, errorData := range err.Errors {
		apiErr.Errors = append(apiErr.Errors, ResponseInfo{
			Code:    int(errorData.Code),
			Message: errorData.Message,
		})
	}
	return classifyError(err.StatusCode, apiErr)
}

func populateErrorMessages(response *Response) {
	if len(response.ErrorMessages) > 0 {
		return
	}
	for _, info := range response.Errors {
		if info.Message != "" {
			response.ErrorMessages = append(response.ErrorMessages, info.Message)
		}
	}
}

type ResourceContainer struct {
	Level      string
	Identifier string
	Type       string
}

func AccountIdentifier(id string) *ResourceContainer {
	return &ResourceContainer{Level: "accounts", Identifier: id, Type: "accounts"}
}

func ZoneIdentifier(id string) *ResourceContainer {
	return &ResourceContainer{Level: "zones", Identifier: id, Type: "zones"}
}

func (rc *ResourceContainer) URLFragment() string {
	if rc == nil {
		return ""
	}
	return "/" + strings.Trim(rc.Type, "/") + "/" + url.PathEscape(rc.Identifier)
}

func (api *API) get(ctx context.Context, endpoint string, params interface{}, result interface{}) (*ResultInfo, error) {
	endpoint = endpointWithQuery(endpoint, queryValues(params))
	raw, err := api.Raw(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}
	if result != nil && len(raw.Result) > 0 {
		if err := json.Unmarshal(raw.Result, result); err != nil {
			return nil, err
		}
	}
	if raw.ResultInfo == nil {
		raw.ResultInfo = &ResultInfo{}
	}
	return raw.ResultInfo, nil
}

func listWithPaginationOptions[T any](ctx context.Context, api *API, endpoint string, params PaginationOptions, defaultPerPage int) ([]T, *ResultInfo, error) {
	autoPaginate := params.Page < 1 && params.PerPage < 1 && params.Cursor == ""
	normalizePaginationOptions(&params, defaultPerPage)

	var results []T
	var lastInfo *ResultInfo
	for {
		var pageResults []T
		info, err := api.get(ctx, endpoint, params, &pageResults)
		if err != nil {
			return []T{}, info, err
		}
		results = append(results, pageResults...)
		lastInfo = info
		if !autoPaginate || info == nil || !info.HasMorePages() {
			break
		}
		next := info.Next()
		params.Page = next.Page
		params.Cursor = next.Cursor
	}
	return results, lastInfo, nil
}

func normalizePaginationOptions(params *PaginationOptions, defaultPerPage int) {
	if params.PerPage < 1 {
		params.PerPage = defaultPerPage
	}
	if params.Page < 1 && params.Cursor == "" {
		params.Page = 1
	}
}

func endpointWithQuery(endpoint string, values url.Values) string {
	if len(values) == 0 {
		return endpoint
	}
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	return endpoint + separator + values.Encode()
}

func queryValues(params interface{}) url.Values {
	values := url.Values{}
	addQueryValues(values, reflect.ValueOf(params))
	return values
}

func addQueryValues(values url.Values, value reflect.Value) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return
	}
	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}
		if structField.Anonymous || field.Kind() == reflect.Struct {
			addQueryValues(values, field)
			continue
		}
		name := structField.Tag.Get("url")
		if name == "-" {
			continue
		}
		if comma := strings.IndexByte(name, ','); comma >= 0 {
			name = name[:comma]
		}
		if name == "" {
			name = strings.ToLower(structField.Name)
		}
		if isZero(field) {
			continue
		}
		for _, v := range fieldValues(field) {
			values.Add(name, v)
		}
	}
}

func isZero(value reflect.Value) bool {
	return value.IsZero()
}

func fieldValues(value reflect.Value) []string {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.String:
		return []string{value.String()}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{strconv.FormatInt(value.Int(), 10)}
	case reflect.Bool:
		return []string{strconv.FormatBool(value.Bool())}
	case reflect.Slice:
		values := make([]string, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			values = append(values, fieldValues(value.Index(i))...)
		}
		return values
	default:
		return []string{fmt.Sprint(value.Interface())}
	}
}

func unmarshalRaw(raw json.RawMessage, result interface{}) error {
	if len(raw) == 0 || result == nil {
		return nil
	}
	return json.Unmarshal(raw, result)
}

type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const listZonesDefaultPageSize = 50

func (api *API) ListZones(ctx context.Context, names ...string) ([]Zone, error) {
	params := struct {
		Name string `url:"name,omitempty"`
	}{}
	if len(names) > 0 {
		params.Name = names[0]
		var zones []Zone
		_, err := api.get(ctx, "/zones", params, &zones)
		return zones, err
	}
	zones, _, err := listWithPaginationOptions[Zone](ctx, api, "/zones", PaginationOptions{}, listZonesDefaultPageSize)
	return zones, err
}

func (api *API) ZoneDetails(ctx context.Context, zoneID string) (Zone, error) {
	var zone Zone
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID), nil, &zone)
	return zone, err
}

type DNSRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied *bool  `json:"proxied,omitempty"`
}

type ListDNSRecordsParams struct {
	ResultInfo
	Type    string `url:"type,omitempty"`
	Name    string `url:"name,omitempty"`
	Content string `url:"content,omitempty"`
}

const listDNSRecordsDefaultPageSize = 100

func (api *API) ListDNSRecords(ctx context.Context, rc *ResourceContainer, params ListDNSRecordsParams) ([]DNSRecord, *ResultInfo, error) {
	hasCursor := params.Cursor != "" || params.Cursors.After != "" || params.Cursors.Before != ""
	autoPaginate := params.Page < 1 && params.PerPage < 1 && !hasCursor
	if params.PerPage < 1 {
		params.PerPage = listDNSRecordsDefaultPageSize
	}
	if params.Page < 1 && !hasCursor {
		params.Page = 1
	}

	var records []DNSRecord
	var lastInfo *ResultInfo
	for {
		var pageRecords []DNSRecord
		info, err := api.get(ctx, rc.URLFragment()+"/dns_records", params, &pageRecords)
		if err != nil {
			return []DNSRecord{}, info, err
		}
		records = append(records, pageRecords...)
		lastInfo = info
		if !autoPaginate || info == nil || !info.HasMorePages() {
			break
		}
		params.ResultInfo = info.Next()
	}
	return records, lastInfo, nil
}

type LockdownListParams struct {
	PaginationOptions
}

const firewallListDefaultPageSize = 50

type AccessRule struct {
	ID    string          `json:"id"`
	Notes string          `json:"notes"`
	Scope AccessRuleScope `json:"scope"`
}

type AccessRuleScope struct {
	Type string `json:"type"`
}

type AccessRulesResponse struct {
	Result     []AccessRule `json:"result"`
	TotalPages int          `json:"total_pages"`
}

func (api *API) ListZoneLockdowns(ctx context.Context, rc *ResourceContainer, params LockdownListParams) ([]AccessRule, *ResultInfo, error) {
	return listWithPaginationOptions[AccessRule](ctx, api, rc.URLFragment()+"/firewall/lockdowns", params.PaginationOptions, firewallListDefaultPageSize)
}

func (api *API) ListAccountAccessRules(ctx context.Context, accountID string, _ AccessRule, page int) (AccessRulesResponse, error) {
	var rules []AccessRule
	info, err := api.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/firewall/access_rules/rules", PaginationOptions{Page: page, PerPage: 50}, &rules)
	response := AccessRulesResponse{Result: rules}
	if info != nil {
		response.TotalPages = info.TotalPages
	}
	return response, err
}

func (api *API) ListZoneAccessRules(ctx context.Context, zoneID string, _ AccessRule, page int) (AccessRulesResponse, error) {
	var rules []AccessRule
	info, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/firewall/access_rules/rules", PaginationOptions{Page: page, PerPage: 50}, &rules)
	response := AccessRulesResponse{Result: rules}
	if info != nil {
		response.TotalPages = info.TotalPages
	}
	return response, err
}

type FilterListParams struct {
	PaginationOptions
}

type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type ListItem struct {
	IP       *string           `json:"ip"`
	ASN      *uint32           `json:"asn"`
	Comment  string            `json:"comment"`
	Hostname *ListItemHostname `json:"hostname"`
	Redirect *ListItemRedirect `json:"redirect"`
}

type ListItemHostname struct {
	URLHostname string `json:"url_hostname"`
}

type ListItemRedirect struct {
	SourceURL           string `json:"source_url"`
	TargetURL           string `json:"target_url"`
	IncludeSubdomains   *bool  `json:"include_subdomains"`
	PreservePathSuffix  *bool  `json:"preserve_path_suffix"`
	PreserveQueryString *bool  `json:"preserve_query_string"`
	StatusCode          *int   `json:"status_code"`
	SubpathMatching     *bool  `json:"subpath_matching"`
}

type Filter struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func (api *API) Filters(ctx context.Context, rc *ResourceContainer, params FilterListParams) ([]Filter, *ResultInfo, error) {
	return listWithPaginationOptions[Filter](ctx, api, rc.URLFragment()+"/filters", params.PaginationOptions, firewallListDefaultPageSize)
}

type FirewallRuleListParams struct {
	PaginationOptions
}

type FirewallRule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func (api *API) FirewallRules(ctx context.Context, rc *ResourceContainer, params FirewallRuleListParams) ([]FirewallRule, *ResultInfo, error) {
	return listWithPaginationOptions[FirewallRule](ctx, api, rc.URLFragment()+"/firewall/rules", params.PaginationOptions, firewallListDefaultPageSize)
}

type RateLimit struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

const listRateLimitsDefaultPageSize = 100

func (api *API) ListAllRateLimits(ctx context.Context, zoneID string) ([]RateLimit, error) {
	params := PaginationOptions{Page: 1, PerPage: listRateLimitsDefaultPageSize}
	var limits []RateLimit
	for {
		var pageLimits []RateLimit
		info, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/rate_limits", params, &pageLimits)
		if err != nil {
			return []RateLimit{}, err
		}
		limits = append(limits, pageLimits...)
		if info == nil || info.Count < info.PerPage {
			break
		}
		params.Page++
	}
	return limits, nil
}

type PageRule struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
}

func (api *API) ListPageRules(ctx context.Context, zoneID string) ([]PageRule, error) {
	var rules []PageRule
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/pagerules", nil, &rules)
	return rules, err
}

type Policy struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Access           string            `json:"access"`
	PermissionGroups []PermissionGroup `json:"permission_groups"`
	ResourceGroups   []ResourceGroup   `json:"resource_groups"`
}

type PermissionGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResourceGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AccountMemberUser struct {
	Email string `json:"email"`
}

type AccountMember struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	User     AccountMemberUser `json:"user"`
	Roles    []Role            `json:"roles"`
	Policies []Policy          `json:"policies"`
}

func (api *API) AccountMembers(ctx context.Context, accountID string, params PaginationOptions) ([]AccountMember, *ResultInfo, error) {
	var members []AccountMember
	info, err := api.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/members", params, &members)
	return members, info, err
}

type MTLSCertificate struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	CA           bool     `json:"ca"`
	Issuer       string   `json:"issuer"`
	SerialNumber string   `json:"serial_number"`
	Certificates []string `json:"certificates"`
}

type ListMTLSCertificatesParams struct{ PaginationOptions }

func (api *API) ListMTLSCertificates(ctx context.Context, rc *ResourceContainer, params ListMTLSCertificatesParams) ([]MTLSCertificate, *ResultInfo, error) {
	var certificates []MTLSCertificate
	info, err := api.get(ctx, rc.URLFragment()+"/mtls_certificates", params, &certificates)
	return certificates, info, err
}

type HostnameAssociation = string

type ListCertificateAuthoritiesHostnameAssociationsParams struct {
	MTLSCertificateID string `url:"mtls_certificate_id,omitempty"`
}

func (api *API) ListCertificateAuthoritiesHostnameAssociations(ctx context.Context, rc *ResourceContainer, params ListCertificateAuthoritiesHostnameAssociationsParams) ([]HostnameAssociation, error) {
	raw, err := api.Raw(ctx, http.MethodGet, endpointWithQuery(rc.URLFragment()+"/certificate_authorities/hostname_associations", queryValues(params)), nil, nil)
	if err != nil {
		return nil, err
	}
	var associations []HostnameAssociation
	if err := unmarshalRaw(raw.Result, &associations); err == nil {
		return associations, nil
	}
	var wrapped struct {
		Hostnames []HostnameAssociation `json:"hostnames"`
	}
	if err := unmarshalRaw(raw.Result, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Hostnames, nil
}

type MagicTransitGRETunnel struct {
	ID                    string                   `json:"id"`
	Name                  string                   `json:"name"`
	Description           string                   `json:"description"`
	CloudflareGREEndpoint string                   `json:"cloudflare_gre_endpoint"`
	CustomerGREEndpoint   string                   `json:"customer_gre_endpoint"`
	InterfaceAddress      string                   `json:"interface_address"`
	MTU                   uint16                   `json:"mtu"`
	TTL                   uint8                    `json:"ttl"`
	HealthCheck           *MagicTransitHealthCheck `json:"health_check"`
}

type MagicTransitIPsecTunnel struct {
	ID                 string                        `json:"id"`
	Name               string                        `json:"name"`
	Description        string                        `json:"description"`
	CloudflareEndpoint string                        `json:"cloudflare_endpoint"`
	CustomerEndpoint   string                        `json:"customer_endpoint"`
	InterfaceAddress   string                        `json:"interface_address"`
	AllowNullCipher    bool                          `json:"allow_null_cipher"`
	Psk                string                        `json:"psk"`
	ReplayProtection   *bool                         `json:"replay_protection"`
	RemoteIdentities   *MagicTransitRemoteIdentities `json:"remote_identities"`
	HealthCheck        *MagicTransitIPsecHealthCheck `json:"health_check"`
}

type MagicTransitStaticRoute struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Nexthop     string                 `json:"nexthop"`
	Prefix      string                 `json:"prefix"`
	Priority    int                    `json:"priority"`
	Weight      int                    `json:"weight"`
	Scope       MagicTransitRouteScope `json:"scope"`
}

type MagicTransitHealthCheck struct {
	Enabled bool   `json:"enabled"`
	Target  string `json:"target"`
	Type    string `json:"type"`
}

type MagicTransitIPsecHealthCheck struct {
	Enabled   bool   `json:"enabled"`
	Direction string `json:"direction"`
	Rate      string `json:"rate"`
	Target    string `json:"target"`
	Type      string `json:"type"`
}

type MagicTransitRemoteIdentities struct {
	FQDNID string `json:"fqdn_id"`
}

type MagicTransitRouteScope struct {
	ColoNames   []string `json:"colo_names"`
	ColoRegions []string `json:"colo_regions"`
}

func (api *API) ListMagicTransitGRETunnels(ctx context.Context, accountID string) ([]MagicTransitGRETunnel, error) {
	var result struct {
		Tunnels []MagicTransitGRETunnel `json:"gre_tunnels"`
	}
	_, err := api.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/magic/gre_tunnels", nil, &result)
	return result.Tunnels, err
}

func (api *API) ListMagicTransitIPsecTunnels(ctx context.Context, accountID string) ([]MagicTransitIPsecTunnel, error) {
	var result struct {
		Tunnels []MagicTransitIPsecTunnel `json:"ipsec_tunnels"`
	}
	_, err := api.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/magic/ipsec_tunnels", nil, &result)
	return result.Tunnels, err
}

func (api *API) ListMagicTransitStaticRoutes(ctx context.Context, accountID string) ([]MagicTransitStaticRoute, error) {
	var result struct {
		Routes []MagicTransitStaticRoute `json:"routes"`
	}
	_, err := api.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/magic/routes", nil, &result)
	return result.Routes, err
}

type LocationStrategy struct {
	Mode      string `json:"mode"`
	PreferECS string `json:"prefer_ecs"`
}

type RandomSteering struct {
	DefaultWeight float64            `json:"default_weight"`
	PoolWeights   map[string]float64 `json:"pool_weights"`
}

type LoadBalancerFixedResponseData struct {
	MessageBody string `json:"message_body"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`
	Location    string `json:"location"`
}

type LoadBalancerRuleOverridesSessionAffinityAttrs struct {
	DrainDuration        int      `json:"drain_duration"`
	TTL                  int      `json:"ttl"`
	Headers              []string `json:"headers"`
	RequireAllHeaders    *bool    `json:"require_all_headers"`
	SameSite             string   `json:"samesite"`
	Secure               string   `json:"secure"`
	ZeroDowntimeFailover string   `json:"zero_downtime_failover"`
}

type LoadBalancerAdaptiveRouting struct {
	FailoverAcrossPools *bool `json:"failover_across_pools"`
}

type LoadBalancerRuleOverrides struct {
	AdaptiveRouting      *LoadBalancerAdaptiveRouting                   `json:"adaptive_routing"`
	CountryPools         map[string][]string                            `json:"country_pools"`
	DefaultPools         []string                                       `json:"default_pools"`
	FallbackPool         string                                         `json:"fallback_pool"`
	LocationStrategy     *LocationStrategy                              `json:"location_strategy"`
	PoPPools             map[string][]string                            `json:"pop_pools"`
	RegionPools          map[string][]string                            `json:"region_pools"`
	PopPools             map[string][]string                            `json:"-"`
	RandomSteering       *RandomSteering                                `json:"random_steering"`
	Persistence          string                                         `json:"session_affinity"`
	PersistenceTTL       *uint                                          `json:"session_affinity_ttl"`
	SessionAffinityAttrs *LoadBalancerRuleOverridesSessionAffinityAttrs `json:"session_affinity_attributes"`
	SteeringPolicy       string                                         `json:"steering_policy"`
	TTL                  uint                                           `json:"ttl"`
}

type LoadBalancerRule struct {
	Name          string                         `json:"name"`
	Condition     string                         `json:"condition"`
	Disabled      bool                           `json:"disabled"`
	FixedResponse *LoadBalancerFixedResponseData `json:"fixed_response"`
	Overrides     LoadBalancerRuleOverrides      `json:"overrides"`
	Priority      int                            `json:"priority"`
	Terminates    bool                           `json:"terminates"`
}

type Healthcheck struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type LoadBalancer struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	DefaultPools       []string            `json:"default_pools"`
	FallbackPool       string              `json:"fallback_pool"`
	LocationStrategy   *LocationStrategy   `json:"location_strategy"`
	RandomSteering     *RandomSteering     `json:"random_steering"`
	Rules              []*LoadBalancerRule `json:"rules"`
	SessionAffinity    string              `json:"session_affinity"`
	SessionAffinityTTL int                 `json:"session_affinity_ttl"`
	SteeringPolicy     string              `json:"steering_policy"`
}

type ListLoadBalancerParams struct{ PaginationOptions }

func (api *API) ListLoadBalancers(ctx context.Context, rc *ResourceContainer, params ListLoadBalancerParams) ([]LoadBalancer, error) {
	var balancers []LoadBalancer
	_, err := api.get(ctx, rc.URLFragment()+"/load_balancers", params, &balancers)
	return balancers, err
}

func (api *API) GetLoadBalancer(ctx context.Context, rc *ResourceContainer, id string) (LoadBalancer, error) {
	var balancer LoadBalancer
	_, err := api.get(ctx, rc.URLFragment()+"/load_balancers/"+url.PathEscape(id), nil, &balancer)
	return balancer, err
}

type ListLoadBalancerPoolParams struct{ PaginationOptions }

type LoadBalancerPool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (api *API) ListLoadBalancerPools(ctx context.Context, rc *ResourceContainer, params ListLoadBalancerPoolParams) ([]LoadBalancerPool, error) {
	var pools []LoadBalancerPool
	_, err := api.get(ctx, rc.URLFragment()+"/load_balancers/pools", params, &pools)
	return pools, err
}

type ListLoadBalancerMonitorParams struct{ PaginationOptions }

type LoadBalancerMonitor struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	ConsecutiveUp   int    `json:"consecutive_up"`
	ConsecutiveDown int    `json:"consecutive_down"`
	Port            uint16 `json:"port"`
}

func (api *API) ListLoadBalancerMonitors(ctx context.Context, rc *ResourceContainer, params ListLoadBalancerMonitorParams) ([]LoadBalancerMonitor, error) {
	var monitors []LoadBalancerMonitor
	_, err := api.get(ctx, rc.URLFragment()+"/load_balancers/monitors", params, &monitors)
	return monitors, err
}

type LogpushJob struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (api *API) ListLogpushJobs(ctx context.Context, rc *ResourceContainer) ([]LogpushJob, error) {
	var jobs []LogpushJob
	_, err := api.get(ctx, rc.URLFragment()+"/logpush/jobs", nil, &jobs)
	return jobs, err
}

type PagesDomain struct {
	Name string `json:"name"`
}

type PagesProject struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Subdomain        string   `json:"subdomain"`
	Domains          []string `json:"domains"`
	ProductionBranch string   `json:"production_branch"`
}

type ListPagesProjectsParams struct{ PaginationOptions }

func (api *API) ListPagesProjects(ctx context.Context, rc *ResourceContainer, params ListPagesProjectsParams) ([]PagesProject, *ResultInfo, error) {
	var projects []PagesProject
	info, err := api.get(ctx, rc.URLFragment()+"/pages/projects", params, &projects)
	return projects, info, err
}

type TunnelListParams struct {
	IsDeleted  *bool      `url:"is_deleted,omitempty"`
	ResultInfo ResultInfo `url:",inline"`
}

type Tunnel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ConfigSource string `json:"config_src"`
	RemoteConfig bool   `json:"remote_config"`
}

func (api *API) ListTunnels(ctx context.Context, rc *ResourceContainer, params TunnelListParams) ([]Tunnel, *ResultInfo, error) {
	var tunnels []Tunnel
	info, err := api.get(ctx, rc.URLFragment()+"/cfd_tunnel", params, &tunnels)
	return tunnels, info, err
}

type TunnelVirtualNetworksListParams struct {
	IsDeleted *bool `url:"is_deleted,omitempty"`
	PaginationOptions
}

type TunnelVirtualNetwork struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default_network"`
}

func (api *API) ListTunnelVirtualNetworks(ctx context.Context, rc *ResourceContainer, params TunnelVirtualNetworksListParams) ([]TunnelVirtualNetwork, error) {
	var networks []TunnelVirtualNetwork
	_, err := api.get(ctx, rc.URLFragment()+"/teamnet/virtual_networks", params, &networks)
	return networks, err
}

type WorkerRoute struct {
	ID      string `json:"id"`
	Pattern string `json:"pattern"`
	Script  string `json:"script"`
	ZoneID  string `json:"zone_id"`
}

type WorkersDomain struct {
	ID          string `json:"id"`
	Hostname    string `json:"hostname"`
	Service     string `json:"service"`
	Environment string `json:"environment"`
	ZoneID      string `json:"zone_id"`
	ZoneName    string `json:"zone_name"`
}

type WorkerMetaData struct {
	ID string `json:"id"`
}

type WorkerCronTrigger struct {
	Cron string `json:"cron"`
}

type ListWorkerCronTriggersParams struct {
	ScriptName string `url:"script_name,omitempty"`
}

func (api *API) ListWorkerCronTriggers(ctx context.Context, rc *ResourceContainer, params ListWorkerCronTriggersParams) ([]WorkerCronTrigger, error) {
	var triggers []WorkerCronTrigger
	_, err := api.get(ctx, rc.URLFragment()+"/workers/scripts/"+url.PathEscape(params.ScriptName)+"/schedules", nil, &triggers)
	return triggers, err
}

type WorkersForPlatformsDispatchNamespace struct {
	Name          string `json:"name"`
	NamespaceName string `json:"namespace_name"`
}

type ListWorkersKVNamespacesParams struct {
	ResultInfo ResultInfo
}

type WorkersKVNamespace struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func (api *API) ListWorkersKVNamespaces(ctx context.Context, rc *ResourceContainer, params ListWorkersKVNamespacesParams) ([]WorkersKVNamespace, *ResultInfo, error) {
	var namespaces []WorkersKVNamespace
	info, err := api.get(ctx, rc.URLFragment()+"/storage/kv/namespaces", params, &namespaces)
	return namespaces, info, err
}

type ListQueuesParams struct {
	ResultInfo ResultInfo
}

type Queue struct {
	ID   string `json:"queue_id"`
	Name string `json:"queue_name"`
}

func (api *API) ListQueues(ctx context.Context, rc *ResourceContainer, params ListQueuesParams) ([]Queue, *ResultInfo, error) {
	var queues []Queue
	info, err := api.get(ctx, rc.URLFragment()+"/queues", params, &queues)
	return queues, info, err
}

type R2Bucket struct {
	Name string `json:"name"`
}

type ListD1DatabasesParams struct {
	ResultInfo ResultInfo
}

type D1Database struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

func (api *API) ListD1Databases(ctx context.Context, rc *ResourceContainer, params ListD1DatabasesParams) ([]D1Database, *ResultInfo, error) {
	var databases []D1Database
	info, err := api.get(ctx, rc.URLFragment()+"/d1/database", params, &databases)
	return databases, info, err
}

type AccessApplication struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListAccessApplicationsParams struct {
	ResultInfo ResultInfo
}

func (api *API) ListAccessApplications(ctx context.Context, rc *ResourceContainer, params ListAccessApplicationsParams) ([]AccessApplication, *ResultInfo, error) {
	var applications []AccessApplication
	info, err := api.get(ctx, rc.URLFragment()+"/access/apps", params, &applications)
	return applications, info, err
}

type AccessGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListAccessGroupsParams struct {
	ResultInfo ResultInfo
}

func (api *API) ListAccessGroups(ctx context.Context, rc *ResourceContainer, params ListAccessGroupsParams) ([]AccessGroup, *ResultInfo, error) {
	var groups []AccessGroup
	info, err := api.get(ctx, rc.URLFragment()+"/access/groups", params, &groups)
	return groups, info, err
}

type AccessIdentityProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListAccessIdentityProvidersParams struct {
	ResultInfo ResultInfo
}

func (api *API) ListAccessIdentityProviders(ctx context.Context, rc *ResourceContainer, params ListAccessIdentityProvidersParams) ([]AccessIdentityProvider, *ResultInfo, error) {
	var providers []AccessIdentityProvider
	info, err := api.get(ctx, rc.URLFragment()+"/access/identity_providers", params, &providers)
	return providers, info, err
}

type AccessMutualTLSCertificate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Certificate string `json:"certificate"`
}

type ListAccessMutualTLSCertificatesParams struct {
	ResultInfo ResultInfo
}

func (api *API) ListAccessMutualTLSCertificates(ctx context.Context, rc *ResourceContainer, params ListAccessMutualTLSCertificatesParams) ([]AccessMutualTLSCertificate, *ResultInfo, error) {
	var certificates []AccessMutualTLSCertificate
	info, err := api.get(ctx, rc.URLFragment()+"/access/certificates", params, &certificates)
	return certificates, info, err
}

func (api *API) GetAccessMutualTLSCertificate(ctx context.Context, rc *ResourceContainer, certificateID string) (AccessMutualTLSCertificate, error) {
	var certificate AccessMutualTLSCertificate
	_, err := api.get(ctx, rc.URLFragment()+"/access/certificates/"+url.PathEscape(certificateID), nil, &certificate)
	return certificate, err
}

type AccessServiceToken struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListAccessPoliciesParams struct {
	ResultInfo ResultInfo
}

func (api *API) ListAccessPolicies(ctx context.Context, rc *ResourceContainer, params ListAccessPoliciesParams) ([]Policy, *ResultInfo, error) {
	var policies []Policy
	info, err := api.get(ctx, rc.URLFragment()+"/access/policies", params, &policies)
	return policies, info, err
}

type AccessCustomPage struct {
	ID   string `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
}

type AccessTag struct {
	Name string `json:"name"`
}

type RulesetKind string

const RulesetKindManaged RulesetKind = "managed"

type Ruleset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type UniversalSSLSetting struct {
	Enabled bool `json:"enabled"`
}

type URLNormalizationSettings struct {
	Scope string `json:"scope"`
	Type  string `json:"type"`
}

type ZoneCacheVariantsValues struct {
	Avif []string `json:"avif"`
	Bmp  []string `json:"bmp"`
	Gif  []string `json:"gif"`
	Bjpg []string `json:"bjpg"`
	Jpeg []string `json:"jpeg"`
	Jpg  []string `json:"jpg"`
	Jp2  []string `json:"jp2"`
	Jpg2 []string `json:"jpg2"`
	Jxr  []string `json:"jxr"`
	Png  []string `json:"png"`
	Tif  []string `json:"tif"`
	Tiff []string `json:"tiff"`
	Webp []string `json:"webp"`
}

type ZoneHold struct {
	Hold              *bool      `json:"hold"`
	IncludeSubdomains *bool      `json:"include_subdomains"`
	HoldAfter         *time.Time `json:"hold_after"`
}

type GetZoneHoldParams struct{}
type GetCacheReserveParams struct{}
type GetRegionalTieredCacheParams struct{}

type ValueSetting struct {
	Value string `json:"value"`
}

type EnabledSetting struct {
	Enabled bool `json:"enabled"`
}

type FallbackOriginSetting struct {
	Origin string `json:"origin"`
}

type LogpullRetentionFlag struct {
	Flag bool `json:"flag"`
}

type WaitingRoomSettings struct {
	SearchEngineCrawlerBypass bool `json:"search_engine_crawler_bypass"`
}

type ZoneCacheVariantsSetting struct {
	Value ZoneCacheVariantsValues `json:"value"`
}

func (api *API) ArgoSmartRouting(ctx context.Context, zoneID string) (ValueSetting, error) {
	var setting ValueSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/argo/smart_routing", nil, &setting)
	return setting, err
}

func (api *API) ArgoTieredCaching(ctx context.Context, zoneID string) (ValueSetting, error) {
	var setting ValueSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/argo/tiered_caching", nil, &setting)
	return setting, err
}

func (api *API) GetCacheReserve(ctx context.Context, rc *ResourceContainer, _ GetCacheReserveParams) (ValueSetting, error) {
	var setting ValueSetting
	_, err := api.get(ctx, rc.URLFragment()+"/cache/cache_reserve", nil, &setting)
	return setting, err
}

func (api *API) GetRegionalTieredCache(ctx context.Context, rc *ResourceContainer, _ GetRegionalTieredCacheParams) (ValueSetting, error) {
	var setting ValueSetting
	_, err := api.get(ctx, rc.URLFragment()+"/cache/regional_tiered_cache", nil, &setting)
	return setting, err
}

func (api *API) UniversalSSLSettingDetails(ctx context.Context, zoneID string) (UniversalSSLSetting, error) {
	var setting UniversalSSLSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/ssl/universal/settings", nil, &setting)
	return setting, err
}

func (api *API) URLNormalizationSettings(ctx context.Context, rc *ResourceContainer) (URLNormalizationSettings, error) {
	var setting URLNormalizationSettings
	_, err := api.get(ctx, rc.URLFragment()+"/url_normalization", nil, &setting)
	return setting, err
}

func (api *API) ZoneCacheVariants(ctx context.Context, zoneID string) (ZoneCacheVariantsSetting, error) {
	var setting ZoneCacheVariantsSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/cache/variants", nil, &setting)
	return setting, err
}

func (api *API) GetZoneHold(ctx context.Context, rc *ResourceContainer, _ GetZoneHoldParams) (ZoneHold, error) {
	var setting ZoneHold
	_, err := api.get(ctx, rc.URLFragment()+"/hold", nil, &setting)
	return setting, err
}

func (api *API) GetLogpullRetentionFlag(ctx context.Context, zoneID string) (*LogpullRetentionFlag, error) {
	var setting LogpullRetentionFlag
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/logs/control/retention/flag", nil, &setting)
	return &setting, err
}

func (api *API) GetPerZoneAuthenticatedOriginPullsStatus(ctx context.Context, zoneID string) (EnabledSetting, error) {
	var setting EnabledSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/origin_tls_client_auth/settings", nil, &setting)
	return setting, err
}

func (api *API) CustomHostnameFallbackOrigin(ctx context.Context, zoneID string) (FallbackOriginSetting, error) {
	var setting FallbackOriginSetting
	_, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/custom_hostnames/fallback_origin", nil, &setting)
	return setting, err
}

func (api *API) GetWaitingRoomSettings(ctx context.Context, rc *ResourceContainer) (WaitingRoomSettings, error) {
	var setting WaitingRoomSettings
	_, err := api.get(ctx, rc.URLFragment()+"/waiting_rooms/settings", nil, &setting)
	return setting, err
}

func (api *API) CustomHostnames(ctx context.Context, zoneID string, page int, _ CustomHostname) ([]CustomHostname, *ResultInfo, error) {
	var hostnames []CustomHostname
	info, err := api.get(ctx, "/zones/"+url.PathEscape(zoneID)+"/custom_hostnames", PaginationOptions{Page: page, PerPage: 50}, &hostnames)
	return hostnames, info, err
}

type CustomHostname struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
}

type EmailRoutingSettings struct {
	Enabled bool   `json:"enabled"`
	Tag     string `json:"tag"`
}

type EmailRoutingCatchAllRule struct {
	ID      string `json:"id"`
	Tag     string `json:"tag"`
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
}

func (api *API) GetEmailRoutingSettings(ctx context.Context, rc *ResourceContainer) (EmailRoutingSettings, error) {
	var settings EmailRoutingSettings
	_, err := api.get(ctx, rc.URLFragment()+"/email/routing", nil, &settings)
	return settings, err
}

func (api *API) GetEmailRoutingCatchAllRule(ctx context.Context, rc *ResourceContainer) (EmailRoutingCatchAllRule, error) {
	var rule EmailRoutingCatchAllRule
	_, err := api.get(ctx, rc.URLFragment()+"/email/routing/rules/catch_all", nil, &rule)
	return rule, err
}

type NotificationPolicy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotificationWebhookIntegration struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListWebAnalyticsSitesParams struct {
	ResultInfo ResultInfo
}

type WebAnalyticsSite struct {
	SiteTag string `json:"site_tag"`
	Host    string `json:"host"`
}

func (api *API) ListWebAnalyticsSites(ctx context.Context, rc *ResourceContainer, params ListWebAnalyticsSitesParams) ([]WebAnalyticsSite, *ResultInfo, error) {
	var sites []WebAnalyticsSite
	info, err := api.get(ctx, rc.URLFragment()+"/rum/site_info/list", params, &sites)
	return sites, info, err
}

type WaitingRoom struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListWaitingRoomRuleParams struct {
	WaitingRoomID string `url:"-"`
	PaginationOptions
}

func (api *API) ListWaitingRoomRules(ctx context.Context, rc *ResourceContainer, params ListWaitingRoomRuleParams) ([]WaitingRoom, error) {
	var rules []WaitingRoom
	_, err := api.get(ctx, rc.URLFragment()+"/waiting_rooms/"+url.PathEscape(params.WaitingRoomID)+"/rules", params, &rules)
	return rules, err
}

type WaitingRoomEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
