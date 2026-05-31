// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	helmreleasecommon "helm.sh/helm/v4/pkg/release/common"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
)

const (
	helmReleaseResourceType = "helm_release"
	helmProviderName        = "helm"
)

var helmReleaseUnsafeStateFields = map[string]struct{}{
	"manifest":            {},
	"resources":           {},
	"set":                 {},
	"set_list":            {},
	"set_sensitive":       {},
	"set_wo":              {},
	"set_wo_revision":     {},
	"values":              {},
	"repository_password": {},
	"repository_username": {},
}

var helmReleaseUnsafeMetadataFields = map[string]struct{}{
	"notes":  {},
	"values": {},
}

type ReleaseGenerator struct {
	terraformutils.Service
	discovery releaseDiscovery
}

type releaseDiscovery interface {
	GetRelease(namespace, name string) (*helmrelease.Release, error)
	ListReleases() ([]*helmrelease.Release, error)
}

type helmReleaseDiscovery struct {
	settings            *cli.EnvSettings
	restClientGetter    genericclioptions.RESTClientGetter
	actionConfigFactory func(namespace string) (*action.Configuration, error)
}

func newHelmReleaseDiscovery() *helmReleaseDiscovery {
	configureHelmProviderKubeEnv()
	settings := cli.New()
	discovery := &helmReleaseDiscovery{
		settings:         settings,
		restClientGetter: helmDiscoveryRESTClientGetter(settings),
	}
	discovery.actionConfigFactory = discovery.actionConfig
	return discovery
}

func helmDiscoveryRESTClientGetter(settings *cli.EnvSettings) genericclioptions.RESTClientGetter {
	flags := genericclioptions.NewConfigFlags(false)
	setStringFlag(flags.Context, settings.KubeContext)
	setStringFlag(flags.BearerToken, settings.KubeToken)
	setStringFlag(flags.APIServer, settings.KubeAPIServer)
	setStringFlag(flags.CAFile, settings.KubeCaFile)
	setStringFlag(flags.TLSServerName, settings.KubeTLSServerName)
	setStringFlag(flags.Impersonate, settings.KubeAsUser)
	setStringFlag(flags.Namespace, settings.Namespace())
	if len(helmProviderKubeConfigPaths()) > 0 {
		setStringFlag(flags.AuthInfoName, os.Getenv("KUBE_CTX_AUTH_INFO"))
		setStringFlag(flags.ClusterName, os.Getenv("KUBE_CTX_CLUSTER"))
	}
	if len(settings.KubeAsGroups) > 0 && flags.ImpersonateGroup != nil {
		*flags.ImpersonateGroup = settings.KubeAsGroups
	}
	if flags.Insecure != nil {
		*flags.Insecure = settings.KubeInsecureSkipTLSVerify
	}
	if username := os.Getenv("KUBE_USER"); username != "" {
		flags.Username = stringPointer(username)
	}
	if password := os.Getenv("KUBE_PASSWORD"); password != "" {
		flags.Password = stringPointer(password)
	}

	clusterCAData := os.Getenv("KUBE_CLUSTER_CA_CERT_DATA")
	clientCertData := os.Getenv("KUBE_CLIENT_CERT_DATA")
	clientKeyData := os.Getenv("KUBE_CLIENT_KEY_DATA")
	proxyURL := os.Getenv("KUBE_PROXY_URL")
	flags.WrapConfigFn = func(config *rest.Config) *rest.Config {
		config.Burst = settings.BurstLimit
		config.QPS = settings.QPS
		if clusterCAData != "" {
			config.CAData = []byte(clusterCAData)
			config.CAFile = ""
		}
		if clientCertData != "" {
			config.CertData = []byte(clientCertData)
			config.CertFile = ""
		}
		if clientKeyData != "" {
			config.KeyData = []byte(clientKeyData)
			config.KeyFile = ""
		}
		if proxyURL != "" {
			if parsedProxyURL, err := url.Parse(proxyURL); err == nil {
				config.Proxy = http.ProxyURL(parsedProxyURL)
			}
		}
		return config
	}
	return flags
}

func setStringFlag(target *string, value string) {
	if target != nil && value != "" {
		*target = value
	}
}

func stringPointer(value string) *string {
	return &value
}

func (d *helmReleaseDiscovery) actionConfig(namespace string) (*action.Configuration, error) {
	configuration := new(action.Configuration)
	restClientGetter := d.restClientGetter
	if restClientGetter == nil {
		restClientGetter = d.settings.RESTClientGetter()
	}
	err := configuration.Init(
		restClientGetter,
		namespace,
		os.Getenv("HELM_DRIVER"),
	)
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func (d *helmReleaseDiscovery) GetRelease(namespace, name string) (*helmrelease.Release, error) {
	configuration, err := d.actionConfigFactory(namespace)
	if err != nil {
		return nil, err
	}
	release, err := action.NewGet(configuration).Run(name)
	if err != nil {
		return nil, err
	}
	typedRelease, ok := release.(*helmrelease.Release)
	if !ok {
		return nil, fmt.Errorf("unexpected helm release type %T", release)
	}
	return typedRelease, nil
}

func (d *helmReleaseDiscovery) newListAction() (*action.List, error) {
	configuration, err := d.actionConfigFactory("")
	if err != nil {
		return nil, err
	}
	list := action.NewList(configuration)
	list.All = true
	list.AllNamespaces = true
	list.StateMask = action.ListAll
	return list, nil
}

func (d *helmReleaseDiscovery) ListReleases() ([]*helmrelease.Release, error) {
	list, err := d.newListAction()
	if err != nil {
		return nil, err
	}
	releases, err := list.Run()
	if err != nil {
		return nil, err
	}
	typedReleases := make([]*helmrelease.Release, 0, len(releases))
	for _, release := range releases {
		typedRelease, ok := release.(*helmrelease.Release)
		if !ok {
			return nil, fmt.Errorf("unexpected helm release type %T", release)
		}
		typedReleases = append(typedReleases, typedRelease)
	}
	return typedReleases, nil
}

type releaseImportID struct {
	Namespace string
	Name      string
}

func (id releaseImportID) String() string {
	return fmt.Sprintf("%s/%s", id.Namespace, id.Name)
}

func parseReleaseImportID(value string) (releaseImportID, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return releaseImportID{}, fmt.Errorf("helm release import ID %q must be namespace/name", value)
	}
	return releaseImportID{Namespace: parts[0], Name: parts[1]}, nil
}

func (g *ReleaseGenerator) releaseDiscovery() releaseDiscovery {
	if g.discovery != nil {
		return g.discovery
	}
	g.discovery = newHelmReleaseDiscovery()
	return g.discovery
}

func (g *ReleaseGenerator) releaseIDFilters() ([]releaseImportID, error) {
	var ids []releaseImportID
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("release") {
			continue
		}
		for _, value := range filter.AcceptableValues {
			id, err := parseReleaseImportID(value)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (g *ReleaseGenerator) InitResources() error {
	discovery := g.releaseDiscovery()
	releaseIDs, err := g.releaseIDFilters()
	if err != nil {
		return err
	}

	var releases []*helmrelease.Release
	if len(releaseIDs) > 0 {
		for _, id := range releaseIDs {
			release, err := discovery.GetRelease(id.Namespace, id.Name)
			if err != nil {
				return err
			}
			releases = append(releases, release)
		}
	} else {
		releases, err = discovery.ListReleases()
		if err != nil {
			return err
		}
	}

	g.Resources = createReleaseResources(selectLatestImportableReleases(releases))
	return nil
}

func (g *ReleaseGenerator) PostRefreshCleanup() {
	if len(g.Filter) == 0 {
		return
	}

	var resources []terraformutils.Resource
	for _, resource := range g.Resources {
		if !g.resourceMatchesPostRefreshFilters(resource) {
			continue
		}
		if !terraformutils.ContainsResource(resources, resource) {
			resources = append(resources, resource)
		}
	}
	g.Resources = resources
}

func (g *ReleaseGenerator) resourceMatchesPostRefreshFilters(resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable("release") && len(filter.AcceptableValues) > 0 {
			if !releaseResourceMatchesIDFilter(resource, filter.AcceptableValues) {
				return false
			}
			continue
		}
		if !filter.Filter(resource) {
			return false
		}
	}
	return true
}

func releaseResourceMatchesIDFilter(resource terraformutils.Resource, acceptableValues []string) bool {
	ids := []string{}
	if resource.InstanceState != nil && resource.InstanceState.ID != "" {
		ids = append(ids, resource.InstanceState.ID)
	}
	if importID := releaseResourceImportID(resource); importID != "" {
		ids = append(ids, importID)
	}
	for _, id := range ids {
		for _, acceptableValue := range acceptableValues {
			if id == acceptableValue {
				return true
			}
		}
	}
	return false
}

func releaseResourceImportID(resource terraformutils.Resource) string {
	name := releaseResourceStringAttribute(resource, "name")
	namespace := releaseResourceStringAttribute(resource, "namespace")
	if name == "" || namespace == "" {
		return ""
	}
	return releaseImportID{Namespace: namespace, Name: name}.String()
}

func releaseResourceStringAttribute(resource terraformutils.Resource, key string) string {
	if resource.InstanceState != nil {
		if value := resource.InstanceState.Attributes[key]; value != "" {
			return value
		}
	}
	if value, ok := resource.Item[key].(string); ok {
		return value
	}
	return ""
}

func (g *ReleaseGenerator) PostConvertHook() error {
	for i := range g.Resources {
		scrubHelmReleaseUnsafeState(&g.Resources[i])
	}
	return nil
}

func createReleaseResources(releases []*helmrelease.Release) []terraformutils.Resource {
	resources := make([]terraformutils.Resource, 0, len(releases))
	for _, release := range releases {
		if release == nil || release.Name == "" || release.Namespace == "" {
			continue
		}
		id := releaseImportID{Namespace: release.Namespace, Name: release.Name}.String()
		attributes := map[string]string{
			"name":      release.Name,
			"namespace": release.Namespace,
		}
		if chart := releaseChartName(release); chart != "" {
			attributes["chart"] = chart
		}
		if version := releaseChartVersion(release); version != "" {
			attributes["version"] = version
		}
		if description := releaseDescription(release); isSafeReleaseDescription(description) {
			attributes["description"] = description
		}

		resources = append(resources, terraformutils.NewResource(
			id,
			releaseResourceName(id, release.Namespace, release.Name),
			helmReleaseResourceType,
			helmProviderName,
			attributes,
			nil,
			nil,
		))
	}
	return resources
}

func releaseResourceName(id, namespace, name string) string {
	sum := sha256.Sum256([]byte(id))
	return fmt.Sprintf("release_%s_%s_%s", namespace, name, hex.EncodeToString(sum[:8]))
}

func selectLatestImportableReleases(releases []*helmrelease.Release) []*helmrelease.Release {
	latestByID := map[string]*helmrelease.Release{}
	for _, release := range releases {
		if release == nil || release.Name == "" || release.Namespace == "" {
			continue
		}
		id := releaseImportID{Namespace: release.Namespace, Name: release.Name}.String()
		if current, ok := latestByID[id]; ok && current.Version > release.Version {
			continue
		}
		latestByID[id] = release
	}

	ids := make([]string, 0, len(latestByID))
	for id := range latestByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	selected := make([]*helmrelease.Release, 0, len(ids))
	for _, id := range ids {
		release := latestByID[id]
		if isImportableReleaseStatus(releaseStatus(release)) {
			selected = append(selected, release)
		}
	}
	return selected
}

func isImportableReleaseStatus(status helmreleasecommon.Status) bool {
	return status == helmreleasecommon.StatusDeployed
}

func releaseStatus(release *helmrelease.Release) helmreleasecommon.Status {
	if release == nil || release.Info == nil {
		return helmreleasecommon.StatusUnknown
	}
	return release.Info.Status
}

func releaseChartName(release *helmrelease.Release) string {
	if release == nil || release.Chart == nil || release.Chart.Metadata == nil {
		return ""
	}
	return release.Chart.Metadata.Name
}

func releaseChartVersion(release *helmrelease.Release) string {
	if release == nil || release.Chart == nil || release.Chart.Metadata == nil {
		return ""
	}
	return release.Chart.Metadata.Version
}

func releaseDescription(release *helmrelease.Release) string {
	if release == nil || release.Info == nil {
		return ""
	}
	return release.Info.Description
}

func isSafeReleaseDescription(description string) bool {
	description = strings.TrimSpace(description)
	if description == "" || strings.ContainsAny(description, "\r\n") {
		return false
	}
	unsafeMarkers := []string{
		"password",
		"passwd",
		"secret",
		"token",
		"credential",
		"private key",
		"kubeconfig",
		"bearer",
		"authorization",
	}
	lowerDescription := strings.ToLower(description)
	for _, marker := range unsafeMarkers {
		if strings.Contains(lowerDescription, marker) {
			return false
		}
	}
	return true
}

func scrubHelmReleaseUnsafeState(resource *terraformutils.Resource) {
	if resource == nil || resource.InstanceInfo == nil || resource.InstanceInfo.Type != helmReleaseResourceType {
		return
	}
	if resource.InstanceState != nil {
		scrubHelmReleaseUnsafeFlatAttributes(resource.InstanceState.Attributes)
		scrubHelmReleaseUnsafeTypedAttributes(resource)
	}
	scrubHelmReleaseUnsafeItem(resource.Item)
}

func scrubHelmReleaseUnsafeFlatAttributes(attributes map[string]string) {
	for key := range attributes {
		if isHelmReleaseUnsafeFlatAttribute(key) {
			delete(attributes, key)
		}
	}
}

func scrubHelmReleaseUnsafeTypedAttributes(resource *terraformutils.Resource) {
	if resource.InstanceState == nil || len(resource.InstanceState.TypedAttributes) == 0 {
		return
	}

	var attributes map[string]interface{}
	if err := json.Unmarshal(resource.InstanceState.TypedAttributes, &attributes); err != nil {
		resource.InstanceState.TypedAttributes = nil
		return
	}
	if attributes == nil {
		resource.InstanceState.TypedAttributes = nil
		return
	}
	scrubHelmReleaseUnsafeItem(attributes)
	rawAttributes, err := json.Marshal(attributes)
	if err != nil {
		resource.InstanceState.TypedAttributes = nil
		return
	}
	resource.InstanceState.SetTypedAttributes(rawAttributes)
}

func scrubHelmReleaseUnsafeItem(item map[string]interface{}) {
	for key, value := range item {
		if isHelmReleaseUnsafeStateField(key) {
			delete(item, key)
			continue
		}
		if key == "metadata" {
			scrubHelmReleaseUnsafeMetadata(value)
		}
	}
}

func scrubHelmReleaseUnsafeMetadata(value interface{}) {
	switch value := value.(type) {
	case []interface{}:
		for _, element := range value {
			scrubHelmReleaseUnsafeMetadata(element)
		}
	case map[string]interface{}:
		for key, nestedValue := range value {
			if isHelmReleaseUnsafeMetadataField(key) {
				delete(value, key)
				continue
			}
			scrubHelmReleaseUnsafeMetadata(nestedValue)
		}
	}
}

func isHelmReleaseUnsafeFlatAttribute(key string) bool {
	if isHelmReleaseUnsafeStateFieldPath(key) {
		return true
	}
	parts := strings.Split(key, ".")
	if len(parts) < 2 || parts[0] != "metadata" {
		return false
	}
	if isHelmReleaseUnsafeMetadataField(parts[1]) {
		return true
	}
	return len(parts) > 2 && isHelmReleaseUnsafeMetadataField(parts[2])
}

func isHelmReleaseUnsafeStateFieldPath(key string) bool {
	for field := range helmReleaseUnsafeStateFields {
		if key == field || strings.HasPrefix(key, field+".") {
			return true
		}
	}
	return false
}

func isHelmReleaseUnsafeStateField(key string) bool {
	_, ok := helmReleaseUnsafeStateFields[key]
	return ok
}

func isHelmReleaseUnsafeMetadataField(key string) bool {
	_, ok := helmReleaseUnsafeMetadataFields[key]
	return ok
}
