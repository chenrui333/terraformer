// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	defaultSchedulerScheduleGroupName  = "default"
	schedulerScheduleResourceType      = "scheduler_schedule"
	schedulerScheduleGroupResourceType = "scheduler_schedule_group"
)

var (
	schedulerAllowEmptyValues = []string{"tags."}
	schedulerResourceTypes    = []string{schedulerScheduleResourceType, schedulerScheduleGroupResourceType}
)

type SchedulerGenerator struct {
	AWSService
}

func (g *SchedulerGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")
		if g.hasTypedSchedulerFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedIDFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			allPredicatesTrue = allPredicatesTrue && filter.Filter(resource)
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *SchedulerGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := scheduler.NewFromConfig(config)

	if err := g.loadScheduleGroups(svc); err != nil {
		return err
	}
	if err := g.loadSchedules(svc); err != nil {
		return err
	}

	return nil
}

func (g *SchedulerGenerator) loadScheduleGroups(svc *scheduler.Client) error {
	p := scheduler.NewListScheduleGroupsPaginator(svc, &scheduler.ListScheduleGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, group := range page.ScheduleGroups {
			if resource, ok := newSchedulerScheduleGroupResource(StringValue(group.Name)); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *SchedulerGenerator) loadSchedules(svc *scheduler.Client) error {
	p := scheduler.NewListSchedulesPaginator(svc, &scheduler.ListSchedulesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, schedule := range page.Schedules {
			if resource, ok := newSchedulerScheduleResource(StringValue(schedule.GroupName), StringValue(schedule.Name)); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func newSchedulerScheduleGroupResource(groupName string) (terraformutils.Resource, bool) {
	if groupName == "" || groupName == defaultSchedulerScheduleGroupName {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		groupName,
		groupName,
		"aws_scheduler_schedule_group",
		"aws",
		schedulerAllowEmptyValues), true
}

func newSchedulerScheduleResource(groupName, scheduleName string) (terraformutils.Resource, bool) {
	if groupName == "" || scheduleName == "" {
		return terraformutils.Resource{}, false
	}
	resourceID := schedulerScheduleImportID(groupName, scheduleName)
	return terraformutils.NewSimpleResource(
		resourceID,
		resourceID,
		"aws_scheduler_schedule",
		"aws",
		schedulerAllowEmptyValues), true
}

func schedulerScheduleImportID(groupName, scheduleName string) string {
	return groupName + "/" + scheduleName
}

func (g *SchedulerGenerator) hasTypedSchedulerFilter() bool {
	for _, serviceName := range schedulerResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *SchedulerGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *SchedulerGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}
