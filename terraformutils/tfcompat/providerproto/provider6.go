// Copyright 2026 The Terraformer Authors.
// SPDX-License-Identifier: Apache-2.0

package providerproto

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/internal/tfplugin6"
	"github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"
)

type GRPCProviderPlugin6 struct{}

func (p *GRPCProviderPlugin6) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("terraformer only supports provider gRPC plugins")
}

func (p *GRPCProviderPlugin6) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("terraformer only supports provider gRPC plugins")
}

func (p *GRPCProviderPlugin6) GRPCClient(ctx context.Context, _ *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &GRPCProvider{
		client: provider6Client{client: tfplugin6.NewProviderClient(conn)},
		ctx:    ctx,
	}, nil
}

func (p *GRPCProviderPlugin6) GRPCServer(*plugin.GRPCBroker, *grpc.Server) error {
	return errors.New("terraformer does not serve provider plugins")
}

type provider6Client struct {
	client tfplugin6.ProviderClient
}

func (p provider6Client) GetProviderSchema(ctx context.Context) (GetProviderSchemaResponse, bool) {
	protoResp, err := p.client.GetProviderSchema(
		ctx,
		&tfplugin6.GetProviderSchema_Request{},
		grpc.MaxCallRecvMsgSize(maxSchemaRecvSize),
	)
	if err != nil {
		return GetProviderSchemaResponse{
			Provider:      emptySchema(),
			ResourceTypes: map[string]configschema.Schema{},
			DataSources:   map[string]configschema.Schema{},
			Diagnostics:   diagnosticsFromError(err),
		}, false
	}
	resp := GetProviderSchemaResponse{
		Provider:      proto6ToProviderSchema(protoResp.Provider),
		ProviderMeta:  proto6ToOptionalProviderSchema(protoResp.ProviderMeta),
		ResourceTypes: map[string]configschema.Schema{},
		DataSources:   map[string]configschema.Schema{},
		Diagnostics:   diagnosticsFromProto6(protoResp.Diagnostics),
	}
	for name, schema := range protoResp.ResourceSchemas {
		resp.ResourceTypes[name] = proto6ToProviderSchema(schema)
	}
	for name, schema := range protoResp.DataSourceSchemas {
		resp.DataSources[name] = proto6ToProviderSchema(schema)
	}
	return resp, true
}

func (p provider6Client) ConfigureProvider(ctx context.Context, r ConfigureProviderRequest, schema GetProviderSchemaResponse) ConfigureProviderResponse {
	configType := schema.Provider.Block.ImpliedType()
	mp, err := msgpack.Marshal(r.Config, configType)
	if err != nil {
		return ConfigureProviderResponse{Diagnostics: diagnosticsFromError(err)}
	}
	terraformVersion := r.TerraformVersion
	if terraformVersion == "" {
		terraformVersion = tfcompat.TerraformVersion
	}
	protoResp, err := p.client.ConfigureProvider(ctx, &tfplugin6.ConfigureProvider_Request{
		TerraformVersion: terraformVersion,
		Config:           &tfplugin6.DynamicValue{Msgpack: mp},
	})
	if err != nil {
		return ConfigureProviderResponse{Diagnostics: diagnosticsFromError(err)}
	}
	return ConfigureProviderResponse{Diagnostics: diagnosticsFromProto6(protoResp.Diagnostics)}
}

func (p provider6Client) ReadResource(ctx context.Context, r ReadResourceRequest, schema GetProviderSchemaResponse) ReadResourceResponse {
	resourceSchema, ok := schema.ResourceTypes[r.TypeName]
	if !ok {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(fmt.Errorf("missing schema for resource type %q", r.TypeName))}
	}
	stateType := resourceSchema.Block.ImpliedType()
	mp, err := msgpack.Marshal(r.PriorState, stateType)
	if err != nil {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
	}
	protoReq := &tfplugin6.ReadResource_Request{
		TypeName:     r.TypeName,
		CurrentState: &tfplugin6.DynamicValue{Msgpack: mp},
		Private:      r.Private,
	}
	if shouldSendProviderMeta(r.ProviderMeta, schema.ProviderMeta) {
		metaMP, err := msgpack.Marshal(r.ProviderMeta, schema.ProviderMeta.Block.ImpliedType())
		if err != nil {
			return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
		}
		protoReq.ProviderMeta = &tfplugin6.DynamicValue{Msgpack: metaMP}
	}
	protoResp, err := p.client.ReadResource(ctx, protoReq)
	if err != nil {
		return ReadResourceResponse{Diagnostics: diagnosticsFromError(err)}
	}
	diags := diagnosticsFromProto6(protoResp.Diagnostics)
	newState, err := decodeDynamicValue6(protoResp.NewState, stateType)
	if err != nil {
		diags = append(diags, diagnosticsFromError(err)...)
	}
	return ReadResourceResponse{
		NewState:    newState,
		Private:     protoResp.Private,
		Diagnostics: diags,
	}
}

func (p provider6Client) ImportResourceState(ctx context.Context, r ImportResourceStateRequest, schema GetProviderSchemaResponse) ImportResourceStateResponse {
	protoResp, err := p.client.ImportResourceState(ctx, &tfplugin6.ImportResourceState_Request{
		TypeName: r.TypeName,
		Id:       r.ID,
	})
	if err != nil {
		return ImportResourceStateResponse{Diagnostics: diagnosticsFromError(err)}
	}
	resp := ImportResourceStateResponse{Diagnostics: diagnosticsFromProto6(protoResp.Diagnostics)}
	for _, imported := range protoResp.ImportedResources {
		resourceSchema, ok := schema.ResourceTypes[imported.TypeName]
		if !ok {
			resp.Diagnostics = append(resp.Diagnostics, diagnosticsFromError(fmt.Errorf("missing schema for resource type %q", imported.TypeName))...)
			continue
		}
		state, err := decodeDynamicValue6(imported.State, resourceSchema.Block.ImpliedType())
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

func decodeDynamicValue6(v *tfplugin6.DynamicValue, ty cty.Type) (cty.Value, error) {
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

func proto6ToProviderSchema(s *tfplugin6.Schema) configschema.Schema {
	if s == nil {
		return emptySchema()
	}
	return configschema.Schema{
		Version: s.Version,
		Block:   proto6ToConfigSchema(s.Block),
	}
}

func proto6ToOptionalProviderSchema(s *tfplugin6.Schema) configschema.Schema {
	if s == nil {
		return configschema.Schema{}
	}
	return proto6ToProviderSchema(s)
}

func proto6ToConfigSchema(b *tfplugin6.Schema_Block) *configschema.Block {
	if b == nil {
		return configschema.EmptyBlock()
	}
	block := &configschema.Block{
		Attributes:      map[string]*configschema.Attribute{},
		BlockTypes:      map[string]*configschema.NestedBlock{},
		Description:     b.Description,
		DescriptionKind: schemaStringKind6(b.DescriptionKind),
		Deprecated:      b.Deprecated,
	}
	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description:     a.Description,
			DescriptionKind: schemaStringKind6(a.DescriptionKind),
			Required:        a.Required,
			Optional:        a.Optional,
			Computed:        a.Computed,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
		}
		if len(a.Type) > 0 {
			attrType, err := ctyjson.UnmarshalType(a.Type)
			if err != nil {
				panic(err)
			}
			attr.Type = attrType
		} else if a.NestedType == nil {
			attr.Type = cty.DynamicPseudoType
		}
		if a.NestedType != nil {
			attr.NestedType = proto6ObjectToConfigSchema(a.NestedType)
		}
		block.Attributes[a.Name] = attr
	}
	for _, nested := range b.BlockTypes {
		block.BlockTypes[nested.TypeName] = proto6ToNestedBlock(nested)
	}
	return block
}

func proto6ToNestedBlock(b *tfplugin6.Schema_NestedBlock) *configschema.NestedBlock {
	if b == nil {
		return &configschema.NestedBlock{Block: *configschema.EmptyBlock()}
	}
	nested := &configschema.NestedBlock{
		Nesting:  schemaNestingMode6(b.Nesting),
		MinItems: int(b.MinItems),
		MaxItems: int(b.MaxItems),
	}
	nested.Block = *proto6ToConfigSchema(b.Block)
	return nested
}

func proto6ObjectToConfigSchema(b *tfplugin6.Schema_Object) *configschema.Object {
	if b == nil {
		return nil
	}
	// Protocol 6 still carries these deprecated fields in schema objects, and
	// configschema.Object preserves them for compatibility with provider schemas.
	minItems := int(b.MinItems) //nolint:staticcheck
	maxItems := int(b.MaxItems) //nolint:staticcheck
	object := &configschema.Object{
		Attributes: map[string]*configschema.Attribute{},
		Nesting:    schemaObjectNestingMode6(b.Nesting),
		MinItems:   minItems,
		MaxItems:   maxItems,
	}
	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description:     a.Description,
			DescriptionKind: schemaStringKind6(a.DescriptionKind),
			Required:        a.Required,
			Optional:        a.Optional,
			Computed:        a.Computed,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
		}
		if len(a.Type) > 0 {
			attrType, err := ctyjson.UnmarshalType(a.Type)
			if err != nil {
				panic(err)
			}
			attr.Type = attrType
		} else if a.NestedType == nil {
			attr.Type = cty.DynamicPseudoType
		}
		if a.NestedType != nil {
			attr.NestedType = proto6ObjectToConfigSchema(a.NestedType)
		}
		object.Attributes[a.Name] = attr
	}
	return object
}

func schemaStringKind6(k tfplugin6.StringKind) configschema.StringKind {
	switch k {
	case tfplugin6.StringKind_MARKDOWN:
		return configschema.StringMarkdown
	default:
		return configschema.StringPlain
	}
}

func schemaNestingMode6(n tfplugin6.Schema_NestedBlock_NestingMode) configschema.NestingMode {
	switch n {
	case tfplugin6.Schema_NestedBlock_SINGLE:
		return configschema.NestingSingle
	case tfplugin6.Schema_NestedBlock_GROUP:
		return configschema.NestingGroup
	case tfplugin6.Schema_NestedBlock_LIST:
		return configschema.NestingList
	case tfplugin6.Schema_NestedBlock_SET:
		return configschema.NestingSet
	case tfplugin6.Schema_NestedBlock_MAP:
		return configschema.NestingMap
	default:
		return 0
	}
}

func schemaObjectNestingMode6(n tfplugin6.Schema_Object_NestingMode) configschema.NestingMode {
	switch n {
	case tfplugin6.Schema_Object_SINGLE:
		return configschema.NestingSingle
	case tfplugin6.Schema_Object_LIST:
		return configschema.NestingList
	case tfplugin6.Schema_Object_SET:
		return configschema.NestingSet
	case tfplugin6.Schema_Object_MAP:
		return configschema.NestingMap
	default:
		return 0
	}
}

func diagnosticsFromProto6(ds []*tfplugin6.Diagnostic) Diagnostics {
	diags := make(Diagnostics, 0, len(ds))
	for _, d := range ds {
		severity := DiagnosticWarning
		if d.Severity == tfplugin6.Diagnostic_ERROR {
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
