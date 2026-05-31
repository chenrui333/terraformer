// SPDX-License-Identifier: Apache-2.0

package yandex

import (
	"context"
	"strings"

	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type yandexSDKConn struct {
	sdk *ycsdk.SDK
}

func yandexGRPCClient(sdk *ycsdk.SDK) grpc.ClientConnInterface {
	return yandexSDKConn{sdk: sdk}
}

func (c yandexSDKConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	conn, err := c.sdk.GetConnection(ctx, yandexFullMethodName(method), opts...)
	if err != nil {
		return err
	}
	return conn.Invoke(ctx, method, args, reply, opts...)
}

func (c yandexSDKConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	conn, err := c.sdk.GetConnection(ctx, yandexFullMethodName(method), opts...)
	if err != nil {
		return nil, err
	}
	return conn.NewStream(ctx, desc, method, opts...)
}

func yandexFullMethodName(method string) protoreflect.FullName {
	method = strings.TrimPrefix(method, "/")
	method = strings.Replace(method, "/", ".", 1)
	return protoreflect.FullName(method)
}
