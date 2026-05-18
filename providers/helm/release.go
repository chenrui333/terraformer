// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

const (
	helmReleaseResourceType = "helm_release"
	helmProviderName        = "helm"
)

type ReleaseGenerator struct {
	terraformutils.Service
	discovery releaseDiscovery
}

type releaseDiscovery interface {
	DefaultNamespace() string
	GetRelease(namespace, name string) (*helmrelease.Release, error)
	ListReleases(namespace string) ([]*helmrelease.Release, error)
}

type helmReleaseDiscovery struct {
	settings *cli.EnvSettings
}

func newHelmReleaseDiscovery() *helmReleaseDiscovery {
	return &helmReleaseDiscovery{settings: cli.New()}
}

func (d *helmReleaseDiscovery) DefaultNamespace() string {
	return d.settings.Namespace()
}

func (d *helmReleaseDiscovery) actionConfig(namespace string) (*action.Configuration, error) {
	configuration := new(action.Configuration)
	err := configuration.Init(
		d.settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) { log.Printf(format, v...) },
	)
	if err != nil {
		return nil, err
	}
	return configuration, nil
}

func (d *helmReleaseDiscovery) GetRelease(namespace, name string) (*helmrelease.Release, error) {
	configuration, err := d.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	return action.NewGet(configuration).Run(name)
}

func (d *helmReleaseDiscovery) ListReleases(namespace string) ([]*helmrelease.Release, error) {
	configuration, err := d.actionConfig(namespace)
	if err != nil {
		return nil, err
	}
	list := action.NewList(configuration)
	list.All = true
	list.AllNamespaces = true
	list.StateMask = action.ListAll
	return list.Run()
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
		namespace := discovery.DefaultNamespace()
		releases, err = discovery.ListReleases(namespace)
		if err != nil {
			return err
		}
	}

	g.Resources = createReleaseResources(selectLatestImportableReleases(releases))
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
			fmt.Sprintf("release_%s_%s", release.Namespace, release.Name),
			helmReleaseResourceType,
			helmProviderName,
			attributes,
			nil,
			nil,
		))
	}
	return resources
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

func isImportableReleaseStatus(status helmrelease.Status) bool {
	return status == helmrelease.StatusDeployed
}

func releaseStatus(release *helmrelease.Release) helmrelease.Status {
	if release == nil || release.Info == nil {
		return helmrelease.StatusUnknown
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
