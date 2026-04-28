// SPDX-License-Identifier: Apache-2.0

package providerwrapper

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils/terraformerstring"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/providerproto"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

type ProviderWrapper struct {
	Provider     *providerproto.GRPCProvider
	client       *plugin.Client
	rpcClient    plugin.ClientProtocol
	providerName string
	config       cty.Value
	schema       *providerproto.GetProviderSchemaResponse
	retryCount   int
	retrySleepMs int
}

func NewProviderWrapper(providerName string, providerConfig cty.Value, verbose bool, options ...map[string]int) (*ProviderWrapper, error) {
	p := &ProviderWrapper{retryCount: 5, retrySleepMs: 300}
	p.providerName = providerName
	p.config = providerConfig

	if len(options) > 0 {
		retryCount, hasOption := options[0]["retryCount"]
		if hasOption {
			p.retryCount = retryCount
		}
		retrySleepMs, hasOption := options[0]["retrySleepMs"]
		if hasOption {
			p.retrySleepMs = retrySleepMs
		}
	}

	err := p.initProvider(verbose)

	return p, err
}

func (p *ProviderWrapper) Kill() {
	p.client.Kill()
}

func (p *ProviderWrapper) GetSchema() *providerproto.GetProviderSchemaResponse {
	if p.schema == nil {
		r := p.Provider.GetProviderSchema()
		p.schema = &r
	}
	return p.schema
}

func (p *ProviderWrapper) GetReadOnlyAttributes(resourceTypes []string) (map[string][]string, error) {
	r := p.GetSchema()

	if r.Diagnostics.HasErrors() {
		return nil, r.Diagnostics.Err()
	}
	readOnlyAttributes := map[string][]string{}
	for resourceName, obj := range r.ResourceTypes {
		if terraformerstring.ContainsString(resourceTypes, resourceName) {
			readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^id$")
			for k, v := range obj.Block.Attributes {
				if !v.Optional && !v.Required {
					if v.Type.IsListType() || v.Type.IsSetType() {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+k+"\\.(.*)")
					} else {
						readOnlyAttributes[resourceName] = append(readOnlyAttributes[resourceName], "^"+k+"$")
					}
				}
			}
			readOnlyAttributes[resourceName] = p.readObjBlocks(obj.Block.BlockTypes, readOnlyAttributes[resourceName], "-1")
		}
	}
	return readOnlyAttributes, nil
}

func (p *ProviderWrapper) readObjBlocks(block map[string]*configschema.NestedBlock, readOnlyAttributes []string, parent string) []string {
	for k, v := range block {
		if len(v.BlockTypes) > 0 {
			if parent == "-1" {
				readOnlyAttributes = p.readObjBlocks(v.BlockTypes, readOnlyAttributes, k)
			} else {
				readOnlyAttributes = p.readObjBlocks(v.BlockTypes, readOnlyAttributes, parent+"\\.[0-9]+\\."+k)
			}
		}
		fieldCount := 0
		for key, l := range v.Attributes {
			if !l.Optional && !l.Required {
				fieldCount++
				switch v.Nesting {
				case configschema.NestingList:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"($|\\.[0-9]+|\\.#)")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"$")
					}
				case configschema.NestingSet:
					if parent == "-1" {
						readOnlyAttributes = append(readOnlyAttributes, "^"+k+"\\.[0-9]+\\."+key+"$")
					} else {
						readOnlyAttributes = append(readOnlyAttributes, "^"+parent+"\\.(.*)\\."+key+"($|\\.(.*))")
					}
				case configschema.NestingMap:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key)
				default:
					readOnlyAttributes = append(readOnlyAttributes, parent+"\\."+key+"$")
				}
			}
		}
		if fieldCount == len(v.Attributes) && fieldCount > 0 && len(v.BlockTypes) == 0 {
			readOnlyAttributes = append(readOnlyAttributes, "^"+k)
		}
	}
	return readOnlyAttributes
}

func (p *ProviderWrapper) Refresh(info *tfcompat.InstanceInfo, state *tfcompat.InstanceState) (*tfcompat.InstanceState, error) {
	schema := p.GetSchema()
	impliedType := schema.ResourceTypes[info.Type].Block.ImpliedType()
	priorState, err := state.AttrsAsObjectValue(impliedType)
	if err != nil {
		return nil, err
	}
	successReadResource := false
	resp := providerproto.ReadResourceResponse{}
	for i := 0; i < p.retryCount; i++ {
		resp = p.Provider.ReadResource(providerproto.ReadResourceRequest{
			TypeName:   info.Type,
			PriorState: priorState,
			Private:    []byte{},
		})
		if resp.Diagnostics.HasErrors() {
			log.Println(resp.Diagnostics.Err())
			log.Printf("WARN: Fail read resource from provider, wait %dms before retry\n", p.retrySleepMs)
			time.Sleep(time.Duration(p.retrySleepMs) * time.Millisecond)
			continue
		}
		successReadResource = true
		break
	}

	if !successReadResource {
		log.Println("Fail read resource from provider, trying import command")
		// retry with regular import command - without resource attributes
		importResponse := p.Provider.ImportResourceState(providerproto.ImportResourceStateRequest{
			TypeName: info.Type,
			ID:       state.ID,
		})
		if importResponse.Diagnostics.HasErrors() {
			return nil, resp.Diagnostics.Err()
		}
		if len(importResponse.ImportedResources) == 0 {
			return nil, errors.New("not able to import resource for a given ID")
		}
		return tfcompat.NewInstanceStateShimmedFromValue(importResponse.ImportedResources[0].State, int(schema.ResourceTypes[info.Type].Version)), nil
	}

	if resp.NewState.IsNull() {
		msg := fmt.Sprintf("ERROR: Read resource response is null for resource %s", info.Id)
		return nil, errors.New(msg)
	}

	return tfcompat.NewInstanceStateShimmedFromValue(resp.NewState, int(schema.ResourceTypes[info.Type].Version)), nil
}

func (p *ProviderWrapper) initProvider(verbose bool) error {
	providerFilePath, err := getProviderFileName(p.providerName)
	if err != nil {
		return err
	}
	options := hclog.LoggerOptions{
		Name:   "plugin",
		Level:  hclog.Error,
		Output: os.Stdout,
	}
	if verbose {
		options.Level = hclog.Trace
	}
	logger := hclog.New(&options)
	p.client = plugin.NewClient(
		&plugin.ClientConfig{
			Cmd:              exec.Command(providerFilePath),
			HandshakeConfig:  providerproto.Handshake,
			VersionedPlugins: providerproto.VersionedPlugins,
			Managed:          true,
			Logger:           logger,
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			AutoMTLS:         true,
		})
	p.rpcClient, err = p.client.Client()
	if err != nil {
		return err
	}
	raw, err := p.rpcClient.Dispense(providerproto.ProviderPluginName)
	if err != nil {
		return err
	}

	p.Provider = raw.(*providerproto.GRPCProvider)

	config, err := p.GetSchema().Provider.Block.CoerceValue(p.config)
	if err != nil {
		return err
	}
	configureResp := p.Provider.ConfigureProvider(providerproto.ConfigureProviderRequest{
		TerraformVersion: tfcompat.TerraformVersion,
		Config:           config,
	})
	if configureResp.Diagnostics.HasErrors() {
		return configureResp.Diagnostics.Err()
	}

	return nil
}

func getProviderFileName(providerName string) (string, error) {
	defaultDataDir := os.Getenv("TF_DATA_DIR")
	if defaultDataDir == "" {
		defaultDataDir = DefaultDataDir
	}
	registryDirs := []string{
		filepath.Join(defaultDataDir, "providers", "registry.terraform.io"),
		filepath.Join(os.Getenv("HOME"), ".terraform.d", "plugins", "registry.terraform.io"),
	}

	var lastErr error
	for _, registryDir := range registryDirs {
		providerFilePath, err := getProviderFileNameFromRegistryDir(registryDir, providerName)
		if err != nil {
			lastErr = errors.Join(lastErr, fmt.Errorf("search provider registry dir %q: %w", registryDir, err))
			continue
		}
		if providerFilePath != "" {
			return providerFilePath, nil
		}
	}
	return "", errors.Join(lastErr, fmt.Errorf("provider %q not found in Terraform registry dirs: %s", providerName, strings.Join(registryDirs, ", ")))
}

func getProviderFileNameFromRegistryDir(registryDir, providerName string) (string, error) {
	providerDirs, err := os.ReadDir(registryDir)
	if err != nil {
		return "", err
	}
	providerFilePath := ""
	for _, providerDir := range providerDirs {
		pluginPath := filepath.Join(registryDir, providerDir.Name(), providerName)
		dirs, err := os.ReadDir(pluginPath)
		if err != nil {
			continue
		}
		for _, versionDir := range dirs {
			if !versionDir.IsDir() {
				continue
			}
			fullPluginPath := filepath.Join(pluginPath, versionDir.Name(), runtime.GOOS+"_"+runtime.GOARCH)
			files, err := os.ReadDir(fullPluginPath)
			if err == nil {
				for _, file := range files {
					if strings.HasPrefix(file.Name(), "terraform-provider-"+providerName) {
						providerFilePath = filepath.Join(fullPluginPath, file.Name())
					}
				}
			}
		}
	}
	return providerFilePath, nil
}

func GetProviderVersion(providerName string) string {
	providerFilePath, err := getProviderFileName(providerName)
	if err != nil {
		log.Println("Can't find provider file path. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#third-party-plugins.")
		return ""
	}
	t := strings.Split(providerFilePath, string(os.PathSeparator))
	providerFileName := t[len(t)-1]
	providerFileNameParts := strings.Split(providerFileName, "_")
	if len(providerFileNameParts) < 2 {
		log.Println("Can't find provider version. Ensure that you are following https://www.terraform.io/docs/configuration/providers.html#plugin-names-and-versions.")
		return ""
	}
	providerVersion := providerFileNameParts[1]
	return "~> " + strings.TrimPrefix(providerVersion, "v")
}
