// SPDX-License-Identifier: Apache-2.0

package typedjson

import (
	"bytes"
	"encoding/json"
)

func UnmarshalObject(raw json.RawMessage) (map[string]interface{}, error) {
	attributes := map[string]interface{}{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&attributes); err != nil {
		return nil, err
	}
	return attributes, nil
}
