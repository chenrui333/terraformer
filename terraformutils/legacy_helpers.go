// Copyright 2026 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformutils

import (
	"fmt"
	"hash/crc32"
	"os"
	"reflect"

	"github.com/mitchellh/go-homedir"
)

func HashString(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	return 0
}

func ReadPathOrContents(value string) (string, bool, error) {
	if value == "" {
		return value, false, nil
	}

	path := value
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return path, true, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		if err != nil {
			return string(contents), true, err
		}
		return string(contents), true, nil
	}

	return value, false, nil
}

func Flatten(value map[string]interface{}) map[string]string {
	result := map[string]string{}
	for key, raw := range value {
		flattenValue(result, key, reflect.ValueOf(raw))
	}
	return result
}

func flattenValue(result map[string]string, prefix string, value reflect.Value) {
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Bool:
		result[prefix] = fmt.Sprintf("%t", value.Bool())
	case reflect.Int:
		result[prefix] = fmt.Sprintf("%d", value.Int())
	case reflect.Map:
		flattenMap(result, prefix, value)
	case reflect.Slice:
		flattenSlice(result, prefix, value)
	case reflect.String:
		result[prefix] = value.String()
	default:
		panic(fmt.Sprintf("unknown flatmap value: %s", value))
	}
}

func flattenMap(result map[string]string, prefix string, value reflect.Value) {
	for _, key := range value.MapKeys() {
		if key.Kind() == reflect.Interface {
			key = key.Elem()
		}
		if key.Kind() != reflect.String {
			panic(fmt.Sprintf("%s: map key is not string: %s", prefix, key))
		}
		flattenValue(result, fmt.Sprintf("%s.%s", prefix, key.String()), value.MapIndex(key))
	}
}

func flattenSlice(result map[string]string, prefix string, value reflect.Value) {
	prefix += "."
	result[prefix+"#"] = fmt.Sprintf("%d", value.Len())
	for i := 0; i < value.Len(); i++ {
		flattenValue(result, fmt.Sprintf("%s%d", prefix, i), value.Index(i))
	}
}
