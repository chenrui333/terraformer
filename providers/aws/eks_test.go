// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
)

func TestEksResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{
			name:  "joins non-empty parts",
			parts: []string{"prod", "kube-system", "aws-node", "a-12345678"},
			want:  "prod-kube-system-aws-node-a-12345678",
		},
		{
			name:  "skips empty parts",
			parts: []string{"prod", "", "vpc-cni"},
			want:  "prod-vpc-cni",
		},
		{
			name:  "empty parts",
			parts: []string{"", ""},
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := eksResourceName(tc.parts...); got != tc.want {
				t.Fatalf("eksResourceName(%v) = %q, want %q", tc.parts, got, tc.want)
			}
		})
	}
}

func TestEksArnName(t *testing.T) {
	tests := []struct {
		name string
		arn  string
		want string
	}{
		{
			name: "role arn",
			arn:  "arn:aws:iam::123456789012:role/Admin",
			want: "123456789012-role-Admin",
		},
		{
			name: "path role arn",
			arn:  "arn:aws:iam::123456789012:role/team/Admin",
			want: "123456789012-role-team-Admin",
		},
		{
			name: "root arn",
			arn:  "arn:aws:iam::123456789012:root",
			want: "123456789012-root",
		},
		{
			name: "EKS cluster access policy arn",
			arn:  "arn:aws:eks::aws:cluster-access-policy/AmazonEKSClusterAdminPolicy",
			want: "aws-cluster-access-policy-AmazonEKSClusterAdminPolicy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := eksArnName(tc.arn); got != tc.want {
				t.Fatalf("eksArnName(%q) = %q, want %q", tc.arn, got, tc.want)
			}
		})
	}
}

func TestEksPostConvertHookLinksClusterScopedResources(t *testing.T) {
	cluster := terraformutils.NewSimpleResource("prod", "prod", "aws_eks_cluster", "aws", eksAllowEmptyValues)
	cluster.Item = map[string]interface{}{"name": "prod"}

	addon := terraformutils.NewSimpleResource("prod:vpc-cni", "prod-vpc-cni", "aws_eks_addon", "aws", eksAllowEmptyValues)
	addon.Item = map[string]interface{}{"cluster_name": "prod"}

	g := EksGenerator{}
	g.Resources = []terraformutils.Resource{cluster, addon}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() returned error: %v", err)
	}

	want := "${aws_eks_cluster.tfer--prod.name}"
	if got := g.Resources[1].Item["cluster_name"]; got != want {
		t.Fatalf("cluster_name = %q, want %q", got, want)
	}
}
