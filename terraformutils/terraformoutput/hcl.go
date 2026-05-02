// SPDX-License-Identifier: Apache-2.0
//
//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package terraformoutput

import (
	"os"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
)

func OutputHclFiles(resources []terraformutils.Resource, provider terraformutils.ProviderGenerator, path string, serviceName string, isCompact bool, output string, sort bool) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	providerConfig := map[string]interface{}{
		"version": providerwrapper.GetProviderVersion(provider.GetName()),
		"source":  terraformutils.ProviderSource(provider.GetName()),
	}

	if providerWithSource, ok := provider.(terraformutils.ProviderWithSource); ok {
		providerConfig["source"] = providerWithSource.GetSource()
	}

	// create provider file
	providerData := provider.GetProviderData()
	providerData["terraform"] = map[string]interface{}{
		"required_providers": []map[string]interface{}{{
			provider.GetName(): providerConfig,
		}},
	}

	providerDataFile, err := terraformutils.Print(providerData, map[string]struct{}{}, output, sort)
	if err != nil {
		return err
	}
	if err := PrintFile(path+"/provider."+GetFileExtension(output), providerDataFile); err != nil {
		return err
	}

	// create outputs files
	outputs := map[string]interface{}{}
	outputsByResource := map[string]map[string]interface{}{}

	for i, r := range resources {
		outputState := map[string]*tfcompat.OutputState{}
		if idKey, ok := resourceOutputIDKey(r); ok {
			outputsByResource[r.InstanceInfo.Type+"_"+r.ResourceName+"_"+idKey] = map[string]interface{}{
				"value": "${" + r.InstanceInfo.Type + "." + r.ResourceName + "." + idKey + "}",
			}
			outputState[r.InstanceInfo.Type+"_"+r.ResourceName+"_"+idKey] = &tfcompat.OutputState{
				Type:  "string",
				Value: r.InstanceState.Attributes[idKey],
			}
		}
		for _, v := range provider.GetResourceConnections() {
			for k, ids := range v {
				if (serviceName != "" && k == serviceName) || (serviceName == "" && k == r.ServiceName()) {
					if _, exist := r.InstanceState.Attributes[ids[1]]; exist {
						key := ids[1]
						if ids[1] == "self_link" || ids[1] == "id" {
							key = r.GetIDKey()
						}
						linkKey := r.InstanceInfo.Type + "_" + r.ResourceName + "_" + key
						outputsByResource[linkKey] = map[string]interface{}{
							"value": "${" + r.InstanceInfo.Type + "." + r.ResourceName + "." + key + "}",
						}
						outputState[linkKey] = &tfcompat.OutputState{
							Type:  "string",
							Value: r.InstanceState.Attributes[ids[1]],
						}
					}
				}
			}
		}
		resources[i].Outputs = outputState
	}
	if len(outputsByResource) > 0 {
		outputs["output"] = outputsByResource
		outputsFile, err := terraformutils.Print(outputs, map[string]struct{}{}, output, sort)
		if err != nil {
			return err
		}
		if err := PrintFile(path+"/outputs."+GetFileExtension(output), outputsFile); err != nil {
			return err
		}
	}

	// group by resource by type
	typeOfServices := map[string][]terraformutils.Resource{}
	for _, r := range resources {
		typeOfServices[r.InstanceInfo.Type] = append(typeOfServices[r.InstanceInfo.Type], r)
	}
	if isCompact {
		err := printFile(resources, "resources", path, output, sort)
		if err != nil {
			return err
		}
	} else {
		for k, v := range typeOfServices {
			fileName := strings.ReplaceAll(k, strings.Split(k, "_")[0]+"_", "")
			err := printFile(v, fileName, path, output, sort)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func printFile(v []terraformutils.Resource, fileName, path, output string, sort bool) error {
	for _, res := range v {
		if res.DataFiles == nil {
			continue
		}
		for fileName, content := range res.DataFiles {
			if err := os.MkdirAll(path+"/data/", os.ModePerm); err != nil {
				return err
			}
			err := os.WriteFile(path+"/data/"+fileName, content, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	tfFile, err := terraformutils.HclPrintResource(v, map[string]interface{}{}, output, sort)
	if err != nil {
		return err
	}
	err = os.WriteFile(path+"/"+fileName+"."+GetFileExtension(output), tfFile, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func resourceOutputIDKey(resource terraformutils.Resource) (string, bool) {
	if resource.InstanceInfo.Type == "kubernetes_manifest" {
		return "", false
	}
	return resource.GetIDKey(), true
}

func PrintFile(path string, data []byte) error {
	return os.WriteFile(path, data, os.ModePerm)
}

func GetFileExtension(outputFormat string) string {
	if outputFormat == "json" {
		return "tf.json"
	}
	return "tf"
}
