// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"net/http"

	"github.com/vultr/govultr/v3"
)

func listAllVultrResources[T any](ctx context.Context, list func(context.Context, *govultr.ListOptions) ([]T, *govultr.Meta, *http.Response, error)) ([]T, error) {
	opt := &govultr.ListOptions{PerPage: 100}
	var resources []T

	for {
		pageResources, meta, resp, err := list(ctx, opt)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			return nil, err
		}
		resources = append(resources, pageResources...)

		if meta == nil || meta.Links == nil || meta.Links.Next == "" {
			break
		}
		opt.Cursor = meta.Links.Next
	}

	return resources, nil
}
