// Copyright 2026 The Terraformer Authors.
// SPDX-License-Identifier: Apache-2.0

package providerproto

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/internal/tfplugin5"
	"github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
)

const ProviderPluginName = "provider"

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  4,
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}

var VersionedPlugins = map[int]plugin.PluginSet{
	5: {
		ProviderPluginName: &GRPCProviderPlugin{},
	},
}

type GRPCProviderPlugin struct{}

func (p *GRPCProviderPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("terraformer only supports provider gRPC plugins")
}

func (p *GRPCProviderPlugin) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("terraformer only supports provider gRPC plugins")
}

func (p *GRPCProviderPlugin) GRPCClient(ctx context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &GRPCProvider{
		client: tfplugin5.NewProviderClient(conn),
		ctx:    ctx,
	}, nil
}

func (p *GRPCProviderPlugin) GRPCServer(*plugin.GRPCBroker, *grpc.Server) error {
	return errors.New("terraformer does not serve provider plugins")
}

type GRPCProvider struct {
	client tfplugin5.ProviderClient
	ctx    context.Context
	schema *GetProviderSchemaResponse
}

type GetProviderSchemaResponse struct {
	Provider      configschema.Schema
	ProviderMeta  configschema.Schema
	ResourceTypes map[string]configschema.Schema
	DataSources   map[string]configschema.Schema
	Diagnostics   Diagnostics
}

type ConfigureProviderRequest struct {
	TerraformVersion string
	Config           cty.Value
}

type ConfigureProviderResponse struct {
	Diagnostics Diagnostics
}

type ReadResourceRequest struct {
	TypeName     string
	PriorState   cty.Value
	ProviderMeta cty.Value
	Private      []byte
}

type ReadResourceResponse struct {
	NewState    cty.Value
	Private     []byte
	Diagnostics Diagnostics
}

type ImportResourceStateRequest struct {
	TypeName string
	ID       string
}

type ImportResourceStateResponse struct {
	ImportedResources []ImportedResource
	Diagnostics       Diagnostics
}

type ImportedResource struct {
	TypeName string
	State    cty.Value
	Private  []byte
}

type DiagnosticSeverity int

const (
	DiagnosticWarning DiagnosticSeverity = iota
	DiagnosticError
)

type Diagnostic struct {
	Severity DiagnosticSeverity
	Summary  string
	Detail   string
}

type Diagnostics []Diagnostic

func (d Diagnostics) HasErrors() bool {
	for _, diag := range d {
		if diag.Severity == DiagnosticError {
			return true
		}
	}
	return false
}

func (d Diagnostics) Err() error {
	if !d.HasErrors() {
		return nil
	}
	messages := make([]string, 0, len(d))
	for _, diag := range d {
		if diag.Severity != DiagnosticError {
			continue
		}
		if diag.Detail == "" {
			messages = append(messages, diag.Summary)
		} else {
			messages = append(messages, diag.Summary+": "+diag.Detail)
		}
	}
	return errors.New(strings.Join(messages, "; "))
}

func (p *GRPCProvider) GetProviderSchema() GetProviderSchemaResponse {
	if p.schema != nil {
		return *p.schema
	}

	protoResp, err := p.client.GetSchema(p.ctx, &tfplugin5.GetProviderSchema_Request{})
	if err != nil {
		return GetProviderSchemaResponse{
			Provider:      emptySchema(),
			ResourceTypes: map[string]configschema.Schema{},
			DataSources:   map[string]configschema.Schema{},
			Diagnostics:   diagnosticsFromError(err),
		}
	}
	resp := GetProviderSchemaResponse{
		Provider:      protoToProviderSchema(protoResp.Provider),
		ProviderMeta:  protoToOptionalProviderSchema(protoResp.ProviderMeta),
		ResourceTypes: map[string]configschema.Schema{},
		DataSources:   map[string]configschema.Schema{},
		Diagnostics:   diagnosticsFromProto(protoResp.Diagnostics),
	}
	for name, schema := range protoResp.ResourceSchemas {
		resp.ResourceTypes[name] = protoToProviderSchema(schema)
	}
	for name, schema := range protoResp.DataSourceSchemas {
		resp.DataSources[name] = protoToProviderSchema(schema)
	}
	p.schema = &resp
	return resp
}

func (p *GRPCProvider) ConfigureProvider(r ConfigureProviderRequest) ConfigureProviderResponse {
	schema := p.GetProviderSchema()
	configType := schema.Provider.Block.ImpliedType()
	mp, err := msgpack.Marshal(r.Config, configType)
	if err != nil {
		return ConfigureProviderResponse{Diagnostics: diagnosticsFromError(err)}
	}
	terraformVersion := r.TerraformVersion
	if terraformVersion == "" {
		terraformVersion = tfcompat.TerraformVersion
	}
	protoResp, err := p.client.Configure(p.ctx, &tfplugin5.Configure_Request{
		TerraformVersion: terraformVersion,
		Config:           &tfplugin5.DynamicValue{Msgpack: mp},
	})
	if err != nil {
		return ConfigureProviderResponse{Diagnostics: diagnosticsFromError(err)}
	}
	return ConfigureProviderResponse{Diagnostics: diagnosticsFromProto(protoResp.Diagnostics)}
}

func (p *GRPCProvider) ReadResource(r ReadResourceRequest) ReadResourceResponse {
	schema := p.GetProviderSchema()
	resourceSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(fmt.Errorf("missing schema for resource type %q", r.TypeName))}
	}
	stateType := resourceSchema.Block.ImpliedType()
	mp, err := msgpack.Marshal(r.PriorState, stateType)
	if err != nil {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
	}
	protoReq := &tfplugin5.ReadResource_Request{
		TypeName:     r.TypeName,
		CurrentState: &tfplugin5.DynamicValue{Msgpack: mp},
		Private:      r.Private,
	}
	if r.ProviderMeta.IsWhollyKnown() && !r.ProviderMeta.IsNull() && schema.ProviderMeta.Block != nil {
		metaMP, err := msgpack.Marshal(r.ProviderMeta, schema.ProviderMeta.Block.ImpliedType())
		if err != nil {
			return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
		}
		protoReq.ProviderMeta = &tfplugin5.DynamicValue{Msgpack: metaMP}
	}
	protoResp, err := p.client.ReadResource(p.ctx, protoReq)
	if err != nil {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
	}
	diags := diagnosticsFromProto(protoResp.Diagnostics)
	newState, err := decodeDynamicValue(protoResp.NewState, stateType)
	if err != nil {
		diags = append(diags, diagnosticsFromError(err)...)
	}
	return ReadResourceResponse{
		NewState:    newState,
		Private:     protoResp.Private,
		Diagnostics: diags,
	}
}

func (p *GRPCProvider) ImportResourceState(r ImportResourceStateRequest) ImportResourceStateResponse {
	protoResp, err := p.client.ImportResourceState(p.ctx, &tfplugin5.ImportResourceState_Request{
		TypeName: r.TypeName,
		Id:       r.ID,
	})
	if err != nil {
		return ImportResourceStateResponse{Diagnostics: diagnosticsFromError(err)}
	}
	resp := ImportResourceStateResponse{Diagnostics: diagnosticsFromProto(protoResp.Diagnostics)}
	schema := p.GetProviderSchema()
	for _, imported := range protoResp.ImportedResources {
		resourceSchema, ok := schema.ResourceTypes[imported.TypeName]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, diagnosticsFromError(fmt.Errorf("missing schema for resource type %q", imported.TypeName))...)
			continue
		}
		state, err := decodeDynamicValue(imported.State, resourceSchema.Block.ImpliedType())
		if err != nil {
			resp.Diagnostics = append(resp.Diagnostics, diagnosticsFromError(err)...)
			continue
		}
		resp.ImportedResources = append(resp.ImportedResources, ImportedResource{
			TypeName: imported.TypeName,
			State:    state,
			Private:  imported.Private,
		})
	}
	return resp
}

func decodeDynamicValue(v *tfplugin5.DynamicValue, ty cty.Type) (cty.Value, error) {
	res := cty.NullVal(ty)
	if v == nil {
		return res, nil
	}
	var err error
	switch {
	case len(v.Msgpack) > 0:
		res, err = msgpack.Unmarshal(v.Msgpack, ty)
	case len(v.Json) > 0:
		res, err = ctyjson.Unmarshal(v.Json, ty)
	}
	return res, err
}

func protoToProviderSchema(s *tfplugin5.Schema) configschema.Schema {
	if s == nil {
		return emptySchema()
	}
	return configschema.Schema{
		Version: s.Version,
		Block:   protoToConfigSchema(s.Block),
	}
}

func protoToOptionalProviderSchema(s *tfplugin5.Schema) configschema.Schema {
	if s == nil {
		return configschema.Schema{}
	}
	return protoToProviderSchema(s)
}

func emptySchema() configschema.Schema {
	return configschema.Schema{Block: configschema.EmptyBlock()}
}

func protoToConfigSchema(b *tfplugin5.Schema_Block) *configschema.Block {
	if b == nil {
		return configschema.EmptyBlock()
	}
	block := &configschema.Block{
		Attributes:      map[string]*configschema.Attribute{},
		BlockTypes:      map[string]*configschema.NestedBlock{},
		Description:     b.Description,
		DescriptionKind: schemaStringKind(b.DescriptionKind),
		Deprecated:      b.Deprecated,
	}
	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description:     a.Description,
			DescriptionKind: schemaStringKind(a.DescriptionKind),
			Required:        a.Required,
			Optional:        a.Optional,
			Computed:        a.Computed,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
		}
		if len(a.Type) == 0 {
			attr.Type = cty.DynamicPseudoType
		} else {
			attrType, err := ctyjson.UnmarshalType(a.Type)
			if err != nil {
				panic(err)
			}
			attr.Type = attrType
		}
		block.Attributes[a.Name] = attr
	}
	for _, nested := range b.BlockTypes {
		block.BlockTypes[nested.TypeName] = protoToNestedBlock(nested)
	}
	return block
}

func protoToNestedBlock(b *tfplugin5.Schema_NestedBlock) *configschema.NestedBlock {
	if b == nil {
		return &configschema.NestedBlock{Block: *configschema.EmptyBlock()}
	}
	nested := &configschema.NestedBlock{
		Nesting:  schemaNestingMode(b.Nesting),
		MinItems: int(b.MinItems),
		MaxItems: int(b.MaxItems),
	}
	nested.Block = *protoToConfigSchema(b.Block)
	return nested
}

func schemaStringKind(k tfplugin5.StringKind) configschema.StringKind {
	switch k {
	case tfplugin5.StringKind_MARKDOWN:
		return configschema.StringMarkdown
	default:
		return configschema.StringPlain
	}
}

func schemaNestingMode(n tfplugin5.Schema_NestedBlock_NestingMode) configschema.NestingMode {
	switch n {
	case tfplugin5.Schema_NestedBlock_SINGLE:
		return configschema.NestingSingle
	case tfplugin5.Schema_NestedBlock_GROUP:
		return configschema.NestingGroup
	case tfplugin5.Schema_NestedBlock_LIST:
		return configschema.NestingList
	case tfplugin5.Schema_NestedBlock_SET:
		return configschema.NestingSet
	case tfplugin5.Schema_NestedBlock_MAP:
		return configschema.NestingMap
	default:
		return 0
	}
}

func diagnosticsFromProto(ds []*tfplugin5.Diagnostic) Diagnostics {
	diags := make(Diagnostics, 0, len(ds))
	for _, d := range ds {
		severity := DiagnosticWarning
		if d.Severity == tfplugin5.Diagnostic_ERROR {
			severity = DiagnosticError
		}
		diags = append(diags, Diagnostic{
			Severity: severity,
			Summary:  d.Summary,
			Detail:   d.Detail,
		})
	}
	return diags
}

func diagnosticsFromError(err error) Diagnostics {
	if err == nil {
		return nil
	}
	return Diagnostics{{
		Severity: DiagnosticError,
		Summary:  err.Error(),
	}}
}
