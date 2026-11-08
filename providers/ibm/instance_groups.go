// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package ibm

import (
	"fmt"
	"os"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
	"golang.org/x/sync/errgroup"
)

// InstanceGroupGenerator ...
type InstanceGroupGenerator struct {
	IBMService
}

func (g *InstanceGroupGenerator) loadInstanceGroup(instanceGroupID, instanceGroupName string) terraformutils.Resource {
	resources := terraformutils.NewSimpleResource(
		instanceGroupID,
		instanceGroupName,
		"ibm_is_instance_group",
		"ibm",
		[]string{})
	return resources
}

func (g *InstanceGroupGenerator) loadInstanceGroupManger(instanceGroupID, instanceGroupManagerID, managerName string, dependsOn []string) terraformutils.Resource {
	if managerName == "" {
		managerName = instanceGroupManagerID
	}
	resources := terraformutils.NewResource(
		fmt.Sprintf("%s/%s", instanceGroupID, instanceGroupManagerID),
		managerName,
		"ibm_is_instance_group_manager",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{
			"depends_on": dependsOn,
		})
	return resources
}

func (g *InstanceGroupGenerator) loadInstanceGroupMangerPolicy(instanceGroupID, instanceGroupManagerID, policyID, policyName string, dependsOn []string) terraformutils.Resource {
	if policyName == "" {
		policyName = policyID
	}
	resources := terraformutils.NewResource(
		fmt.Sprintf("%s/%s/%s", instanceGroupID, instanceGroupManagerID, policyID),
		policyName,
		"ibm_is_instance_group_manager_policy",
		"ibm",
		map[string]string{},
		[]string{},
		map[string]interface{}{
			"depends_on": dependsOn,
		})
	return resources
}

func (g *InstanceGroupGenerator) handlePolicies(sess *vpcv1.VpcV1, instanceGroupID, instanceGroupManagerID string, policies, dependsOn []string) error {
	for _, instanceGroupManagerPolicyID := range policies {
		getInstanceGroupManagerPolicyOptions := vpcv1.GetInstanceGroupManagerPolicyOptions{
			ID:                     &instanceGroupManagerPolicyID,
			InstanceGroupID:        &instanceGroupID,
			InstanceGroupManagerID: &instanceGroupManagerID,
		}
		data, response, err := sess.GetInstanceGroupManagerPolicy(&getInstanceGroupManagerPolicyOptions)
		if err != nil {
			return fmt.Errorf("error getting InstanceGroup Manager Policy: %w\n%s", err, response)
		}
		instanceGroupManagerPolicy, ok := data.(*vpcv1.InstanceGroupManagerPolicy)
		if !ok {
			return fmt.Errorf("unexpected InstanceGroup Manager Policy type %T", data)
		}
		resourceMutex.Lock()
		g.Resources = append(g.Resources, g.loadInstanceGroupMangerPolicy(instanceGroupID,
			instanceGroupManagerID,
			instanceGroupManagerPolicyID,
			core.StringNilMapper(instanceGroupManagerPolicy.Name),
			dependsOn))
		resourceMutex.Unlock()
	}
	return nil
}

func (g *InstanceGroupGenerator) handleManagers(sess *vpcv1.VpcV1, instanceGroupID string, managers, dependsOn []string) error {
	var policies errgroup.Group
	for _, instanceGroupManagerID := range managers {
		getInstanceGroupManagerOptions := vpcv1.GetInstanceGroupManagerOptions{
			ID:              &instanceGroupManagerID,
			InstanceGroupID: &instanceGroupID,
		}
		instanceGroupManagerIntf, response, err := sess.GetInstanceGroupManager(&getInstanceGroupManagerOptions)
		if err != nil {
			return fmt.Errorf("error getting InstanceGroup Manager: %w\n%s", err, response)
		}
		instanceGroupManager, ok := instanceGroupManagerIntf.(*vpcv1.InstanceGroupManager)
		if !ok {
			return fmt.Errorf("unexpected InstanceGroup Manager type %T", instanceGroupManagerIntf)
		}
		managerResource := g.loadInstanceGroupManger(instanceGroupID, instanceGroupManagerID, core.StringNilMapper(instanceGroupManager.Name), dependsOn)
		resourceMutex.Lock()
		g.Resources = append(g.Resources, managerResource)
		resourceMutex.Unlock()

		policyIDs := make([]string, 0)

		for i := 0; i < len(instanceGroupManager.Policies); i++ {
			policyIDs = append(policyIDs, *(instanceGroupManager.Policies[i].ID))
		}
		dependsOn1 := makeDependsOn(dependsOn,
			"ibm_is_instance_group_manager."+managerResource.ResourceName)
		policies.Go(func() error {
			return g.handlePolicies(sess, instanceGroupID, instanceGroupManagerID, policyIDs, dependsOn1)
		})
	}
	return policies.Wait()
}

func (g *InstanceGroupGenerator) handleInstanceGroups(sess *vpcv1.VpcV1) error {
	// Support for pagination
	start := ""
	var allrecs []vpcv1.InstanceGroup
	for {
		listInstanceGroupOptions := vpcv1.ListInstanceGroupsOptions{}
		if start != "" {
			listInstanceGroupOptions.Start = &start
		}
		instanceGroupsCollection, response, err := sess.ListInstanceGroups(&listInstanceGroupOptions)
		if err != nil {
			return fmt.Errorf("error fetching InstanceGroups %w\n%s", err, response)
		}
		start = GetNext(instanceGroupsCollection.Next)
		allrecs = append(allrecs, instanceGroupsCollection.InstanceGroups...)
		if start == "" {
			break
		}
	}

	var managers errgroup.Group

	for _, instanceGroup := range allrecs {
		var dependsOn []string
		dependsOn = append(dependsOn,
			"ibm_is_instance_group."+terraformutils.TfSanitize(*instanceGroup.Name))
		instanceGoupID := *instanceGroup.ID
		resourceMutex.Lock()
		g.Resources = append(g.Resources, g.loadInstanceGroup(instanceGoupID, *instanceGroup.Name))
		resourceMutex.Unlock()
		managerIDs := make([]string, 0)
		for i := 0; i < len(instanceGroup.Managers); i++ {
			managerIDs = append(managerIDs, *(instanceGroup.Managers[i].ID))
		}
		managers.Go(func() error {
			return g.handleManagers(sess, instanceGoupID, managerIDs, dependsOn)
		})
	}
	return managers.Wait()
}

// InitResources ...
func (g *InstanceGroupGenerator) InitResources() error {
	apiKey := os.Getenv("IC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("no API key set")
	}

	// Instantiate the service with an API key based IAM authenticator
	sess, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: &core.IamAuthenticator{
			ApiKey: apiKey,
		},
	})
	if err != nil {
		return err
	}

	return g.handleInstanceGroups(sess)
}
