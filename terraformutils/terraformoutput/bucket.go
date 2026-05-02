// SPDX-License-Identifier: Apache-2.0
package terraformoutput

import (
	"context"
	"strings"

	"cloud.google.com/go/storage"
)

type BucketState struct {
	Name string
}

func (b BucketState) BucketGetTfData(path string) interface{} {
	name := strings.ReplaceAll(b.Name, "gs://", "")
	bucketStateData := map[string]interface{}{
		"terraform": map[string]interface{}{
			"backend": []map[string]interface{}{
				{
					"gcs": map[string]interface{}{
						"bucket": name,
						"prefix": b.BucketPrefix(path),
					},
				},
			},
		},
	}
	return bucketStateData
}

func (b BucketState) BucketPrefix(path string) string {
	return strings.TrimSuffix(path, "/")
}

func (b BucketState) BucketUpload(path string, file []byte) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	name := strings.ReplaceAll(b.Name, "gs://", "")
	wc := client.Bucket(name).Object(b.BucketPrefix(path) + "/default.tfstate").NewWriter(ctx)
	if _, err = wc.Write(file); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}
