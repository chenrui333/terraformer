// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type ComputeGenerator struct {
	OpenStackService
}

// createResources iterate on all openstack_compute_instance_v2
func (g *ComputeGenerator) createResources(list *pagination.Pager, volclient *gophercloud.ServiceClient) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}

	err := list.EachPage(context.Background(), func(ctx context.Context, page pagination.Page) (bool, error) {
		servers, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}

		for _, s := range servers {
			var bds = []map[string]interface{}{}
			var vol []volumes.Volume
			t := map[string]interface{}{}
			if volclient != nil {
				for _, av := range s.AttachedVolumes {
					onevol, err := volumes.Get(ctx, volclient, av.ID).Extract()
					if err == nil {
						vol = append(vol, *onevol)
					}
				}

				sort.SliceStable(vol, func(i, j int) bool {
					return vol[i].Attachments[0].Device < vol[j].Attachments[0].Device
				})

				var bindex = 0
				var dependsOn = ""
				for _, v := range vol {
					if v.Bootable == "true" && v.VolumeImageMetadata != nil {
						bds = append(bds, map[string]interface{}{
							"source_type":           "image",
							"uuid":                  v.VolumeImageMetadata["image_id"],
							"volume_size":           strconv.Itoa(v.Size),
							"boot_index":            strconv.Itoa(bindex),
							"destination_type":      "volume",
							"delete_on_termination": "false",
						})
						bindex++
					} else {
						tv := map[string]interface{}{}
						if dependsOn != "" {
							tv["depends_on"] = []string{dependsOn}
						}

						name := s.Name + strings.ReplaceAll(v.Attachments[0].Device, "/dev/", "")
						rid := s.ID + "/" + v.ID
						resource := terraformutils.NewResource(
							rid,
							name,
							"openstack_compute_volume_attach_v2",
							"openstack",
							map[string]string{},
							[]string{},
							tv,
						)
						dependsOn = "openstack_compute_volume_attach_v2." + terraformutils.TfSanitize(name)
						tv["instance_name"] = terraformutils.TfSanitize(s.Name)
						if v.Name == "" {
							v.Name = v.ID
						}
						tv["volume_name"] = terraformutils.TfSanitize(v.Name)
						resources = append(resources, resource)
					}
				}
			}

			if len(bds) > 0 {
				t = map[string]interface{}{"block_device": bds}
			}

			resource := terraformutils.NewResource(
				s.ID,
				s.Name,
				"openstack_compute_instance_v2",
				"openstack",
				map[string]string{},
				[]string{},
				t,
			)

			resources = append(resources, resource)
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return resources, nil
}

// Generate TerraformResources from OpenStack API,
func (g *ComputeGenerator) InitResources() error {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}

	provider, err := openstack.AuthenticatedClient(context.Background(), opts)
	if err != nil {
		return err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: g.GetArgs()["region"].(string),
	})
	if err != nil {
		return err
	}

	list := servers.List(client, nil)
	volclient, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
		Region: g.GetArgs()["region"].(string)})
	if err != nil {
		log.Println("VolumeImageMetadata requires blockStorage API v3")
		volclient = nil
	}
	resources, err := g.createResources(&list, volclient)
	if err != nil {
		return err
	}
	g.Resources = resources

	return nil
}

func (g *ComputeGenerator) PostConvertHook() error {
	for i, r := range g.Resources {
		if r.InstanceInfo.Type == "openstack_compute_volume_attach_v2" {
			g.Resources[i].Item["volume_id"] = "${openstack_blockstorage_volume_v3." + r.AdditionalFields["volume_name"].(string) + ".id}"
			g.Resources[i].Item["instance_id"] = "${openstack_compute_instance_v2." + r.AdditionalFields["instance_name"].(string) + ".id}"
			delete(g.Resources[i].Item, "volume_name")
			delete(g.Resources[i].Item, "instance_name")
			delete(g.Resources[i].Item, "device")
		}
		if r.InstanceInfo.Type != "openstack_compute_instance_v2" {
			continue
		}

		// Copy "all_metadata.%" to "metadata.%"
		for k, v := range g.Resources[i].InstanceState.Attributes {
			if strings.HasPrefix(k, "all_metadata") {
				newKey := strings.Replace(k, "all_metadata", "metadata", 1)
				g.Resources[i].InstanceState.Attributes[newKey] = v
			}
		}
		// Replace "all_metadata" to "metadata"
		// because "all_metadata" field cannot be set as resource argument
		for k, v := range g.Resources[i].Item {
			if strings.HasPrefix(k, "all_metadata") {
				newKey := strings.Replace(k, "all_metadata", "metadata", 1)
				g.Resources[i].Item[newKey] = v
				delete(g.Resources[i].Item, k)
			}
		}
		if r.AdditionalFields["block_device"] != nil {
			bds := r.AdditionalFields["block_device"].([]map[string]interface{})
			for bi, bd := range bds {
				for k, v := range bd {
					g.Resources[i].InstanceState.Attributes["block_device."+strconv.Itoa(bi)+"."+k] = v.(string)
				}
			}

			g.Resources[i].InstanceState.Attributes["block_device.#"] = strconv.Itoa(len(bds))
		}
	}

	return nil
}
