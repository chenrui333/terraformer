package providerproto

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/internal/tfplugin5"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestGetProviderSchemaAllowsLargeResponses(t *testing.T) {
	largeDescription := strings.Repeat("x", 5<<20)
	listener := bufconn.Listen(8 << 20)
	server := grpc.NewServer()
	tfplugin5.RegisterProviderServer(server, &largeSchemaProvider{description: largeDescription})
	t.Cleanup(server.Stop)
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go func() {
		_ = server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create grpc client: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	provider := &GRPCProvider{
		client: tfplugin5.NewProviderClient(conn),
		ctx:    context.Background(),
	}
	resp := provider.GetProviderSchema()
	if resp.Diagnostics.HasErrors() {
		t.Fatalf("GetProviderSchema returned diagnostics: %v", resp.Diagnostics.Err())
	}
	if got := resp.Provider.Block.Description; got != largeDescription {
		t.Fatalf("schema description length = %d, want %d", len(got), len(largeDescription))
	}
}

func TestShouldSendProviderMeta(t *testing.T) {
	metaSchema := configschema.Schema{Block: configschema.EmptyBlock()}
	testCases := map[string]struct {
		value  cty.Value
		schema configschema.Schema
		want   bool
	}{
		"missing schema": {
			value: cty.EmptyObjectVal,
			want:  false,
		},
		"nil value": {
			value:  cty.NilVal,
			schema: metaSchema,
			want:   false,
		},
		"null value": {
			value:  cty.NullVal(cty.EmptyObject),
			schema: metaSchema,
			want:   false,
		},
		"unknown value": {
			value:  cty.UnknownVal(cty.EmptyObject),
			schema: metaSchema,
			want:   false,
		},
		"known value": {
			value:  cty.EmptyObjectVal,
			schema: metaSchema,
			want:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if got := shouldSendProviderMeta(tc.value, tc.schema); got != tc.want {
				t.Fatalf("shouldSendProviderMeta() = %t, want %t", got, tc.want)
			}
		})
	}
}

type largeSchemaProvider struct {
	tfplugin5.UnimplementedProviderServer

	description string
}

func (p *largeSchemaProvider) GetSchema(context.Context, *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	return &tfplugin5.GetProviderSchema_Response{
		Provider: &tfplugin5.Schema{
			Block: &tfplugin5.Schema_Block{Description: p.description},
		},
	}, nil
}
