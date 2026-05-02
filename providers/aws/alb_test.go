// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

func TestListenerSupportsCertificates(t *testing.T) {
	tests := []struct {
		name     string
		protocol types.ProtocolEnum
		want     bool
	}{
		{name: "https", protocol: types.ProtocolEnumHttps, want: true},
		{name: "tls", protocol: types.ProtocolEnumTls, want: true},
		{name: "http", protocol: types.ProtocolEnumHttp, want: false},
		{name: "tcp", protocol: types.ProtocolEnumTcp, want: false},
		{name: "udp", protocol: types.ProtocolEnumUdp, want: false},
		{name: "tcp udp", protocol: types.ProtocolEnumTcpUdp, want: false},
		{name: "geneve", protocol: types.ProtocolEnumGeneve, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenerSupportsCertificates(tt.protocol); got != tt.want {
				t.Fatalf("listenerSupportsCertificates(%q) = %t, want %t", tt.protocol, got, tt.want)
			}
		})
	}
}
