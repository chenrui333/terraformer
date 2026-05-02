// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/packethost/packngo"
)

type DeviceGenerator struct {
	EquinixMetalService
}

func (g DeviceGenerator) listDevices(client *packngo.Client) ([]packngo.Device, error) {
	devices, _, err := client.Devices.List(g.GetArgs()["project_id"].(string), nil)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

func (g DeviceGenerator) createResources(deviceList []packngo.Device) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, device := range deviceList {
		resources = append(resources, terraformutils.NewSimpleResource(
			device.ID,
			device.Hostname,
			"metal_device",
			"equinixmetal",
			[]string{}))
	}
	return resources
}

func (g *DeviceGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	output, err := g.listDevices(client)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output)
	return nil
}
